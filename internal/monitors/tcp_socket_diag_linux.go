//go:build linux

package monitors

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func readP2PTCPSockets(namespace string) ([]tcpSocketSample, error) {
	fd, err := openTCPDiagSocket(namespace)
	if err != nil {
		return nil, err // Never silently fall back to host sockets.
	}
	defer unix.Close(fd)
	var samples []tcpSocketSample
	for _, family := range []uint8{linuxAFInet, linuxAFInet6} {
		rows, err := dumpP2PTCPSocketsOnSocket(fd, family)
		if family == linuxAFInet6 && (errors.Is(err, unix.EAFNOSUPPORT) || errors.Is(err, unix.EPROTONOSUPPORT)) {
			continue // IPv6 disabled on this host.
		}
		if err != nil {
			return nil, err // Never publish a partial family/dump as complete.
		}
		samples = append(samples, rows...)
		if len(samples) > tcpPeerBytesSockCap {
			return nil, fmt.Errorf("P2P socket count exceeds %d", tcpPeerBytesSockCap)
		}
	}
	return samples, nil
}

func dumpP2PTCPSockets(family uint8) ([]tcpSocketSample, error) {
	fd, err := openTCPDiagSocket("")
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	return dumpP2PTCPSocketsOnSocket(fd, family)
}

// A socket keeps the namespace in which it was created. Only socket creation
// enters the target namespace; all dump IO can then run on the normal thread.
func openTCPDiagSocket(namespace string) (int, error) {
	var fd int
	var err error
	if namespace == "" {
		fd, err = unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_INET_DIAG)
	} else {
		fd, err = openNamespacedTCPDiagSocket(namespace)
	}
	if err != nil {
		return -1, fmt.Errorf("open inet_diag: %w", err)
	}
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		unix.Close(fd)
		return -1, fmt.Errorf("bind inet_diag: %w", err)
	}
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &unix.Timeval{Sec: 2}); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

func openNamespacedTCPDiagSocket(namespace string) (int, error) {
	nsFD, err := unix.Open(namespace, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, fmt.Errorf("open TCP network namespace %q: %w", namespace, err)
	}
	type result struct {
		fd  int
		err error
	}
	ch := make(chan result, 1)
	goSafe("tcp_peer_bytes", func() {
		// Intentionally never unlock this worker. Go destroys a locked thread
		// when its goroutine exits, so a target namespace cannot leak into the
		// runtime pool, even on a panic or failure to restore a host namespace.
		runtime.LockOSThread()
		defer unix.Close(nsFD)
		r := result{-1, fmt.Errorf("TCP namespace socket worker did not complete")}
		defer func() { ch <- r }()
		if err := unix.Setns(nsFD, unix.CLONE_NEWNET); err != nil {
			r.err = fmt.Errorf("enter TCP network namespace %q: %w (requires CAP_SYS_ADMIN and namespace file access)", namespace, err)
			return
		}
		r.fd, r.err = unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_INET_DIAG)
	})
	r := <-ch
	return r.fd, r.err
}

func dumpP2PTCPSocketsOnSocket(fd int, family uint8) ([]tcpSocketSample, error) {
	// nlmsghdr + inet_diag_req_v2. Ask for full sockets with TCP_INFO,
	// excluding listeners, incomplete handshakes and TIME_WAIT entries.
	// FIN_WAIT2 can also be a timewait mini-socket without TCP_INFO.
	const sockDiagByFamily = 20
	request := make([]byte, 16+56)
	binary.NativeEndian.PutUint32(request[0:4], uint32(len(request)))
	binary.NativeEndian.PutUint16(request[4:6], sockDiagByFamily)
	binary.NativeEndian.PutUint16(request[6:8], unix.NLM_F_REQUEST|unix.NLM_F_DUMP)
	binary.NativeEndian.PutUint32(request[8:12], uint32(family))
	request[16] = family
	request[17] = unix.IPPROTO_TCP
	request[18] = 1 << (inetDiagInfo - 1)
	binary.NativeEndian.PutUint32(request[20:24], (1<<1)|(1<<4)|(1<<8)|(1<<9)|(1<<11))
	if err := unix.Sendto(fd, request, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return nil, fmt.Errorf("request inet_diag: %w", err)
	}

	var samples []tcpSocketSample
	buf := make([]byte, 1<<20)
	deadline := time.Now().Add(2 * time.Second)
	// Bound a dump even if a noisy namespace never finishes sending it.
	for batch := 0; batch < 1024; batch++ {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("inet_diag dump deadline exceeded")
		}
		n, _, flags, from, err := unix.Recvmsg(fd, buf, nil, 0)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("receive inet_diag: %w", err)
		}
		if sender, ok := from.(*unix.SockaddrNetlink); !ok || sender.Pid != 0 || flags&unix.MSG_TRUNC != 0 {
			return nil, fmt.Errorf("invalid or truncated inet_diag response")
		}
		messages, err := syscall.ParseNetlinkMessage(buf[:n])
		if err != nil {
			return nil, err
		}
		for _, msg := range messages {
			if msg.Header.Seq != uint32(family) || msg.Header.Flags&unix.NLM_F_DUMP_INTR != 0 {
				return nil, fmt.Errorf("interrupted or mismatched inet_diag dump")
			}
			switch msg.Header.Type {
			case unix.NLMSG_DONE, unix.NLMSG_ERROR:
				if len(msg.Data) >= 4 {
					code := int32(binary.NativeEndian.Uint32(msg.Data[:4]))
					if code < 0 {
						return nil, fmt.Errorf("inet_diag: %w", unix.Errno(-code))
					}
				} else if msg.Header.Type == unix.NLMSG_ERROR {
					return nil, fmt.Errorf("short inet_diag error")
				}
				if msg.Header.Type == unix.NLMSG_DONE {
					return samples, nil
				}
			case sockDiagByFamily:
				s, relevant, err := parseP2PSocketDiag(msg.Data)
				if err != nil {
					return nil, err
				}
				if relevant {
					samples = append(samples, s)
					if len(samples) > tcpPeerBytesSockCap {
						return nil, fmt.Errorf("P2P socket count exceeds %d", tcpPeerBytesSockCap)
					}
				}
			default:
				return nil, fmt.Errorf("unexpected inet_diag message %d", msg.Header.Type)
			}
		}
	}
	return nil, fmt.Errorf("inet_diag dump exceeded batch limit")
}
