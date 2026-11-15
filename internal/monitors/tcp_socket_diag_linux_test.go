//go:build linux

package monitors

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Exercise the actual kernel ABI, not only synthetic netlink fixtures.
func TestP2PSocketDiagLiveTCPBytes(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:4001")
	if err != nil {
		t.Skipf("loopback P2P test port unavailable: %v", err)
	}
	defer listener.Close()
	client, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	for _, conn := range []net.Conn{client, server} {
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := dumpP2PTCPSockets(linuxAFInet)
	if err != nil {
		t.Fatal(err)
	}
	baseline := make(map[uint64]tcpSocketSample)
	for _, s := range before {
		baseline[s.cookie] = s
	}
	payload := bytes.Repeat([]byte("p"), 4096)
	if _, err := server.Write(payload); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(client, make([]byte, len(payload))); err != nil {
		t.Fatal(err)
	}
	after, err := dumpP2PTCPSockets(linuxAFInet)
	if err != nil {
		t.Fatal(err)
	}
	var observedIn, observedOut bool
	for _, s := range after {
		prev, existed := baseline[s.cookie]
		if !existed || s.ip != "127.0.0.1" {
			continue
		}
		observedIn = observedIn || s.received >= prev.received+uint64(len(payload))
		observedOut = observedOut || s.sent >= prev.sent+uint64(len(payload))
	}
	if !observedIn || !observedOut {
		t.Fatalf("kernel byte increments missing: inbound=%v outbound=%v", observedIn, observedOut)
	}
}

func TestTCPNamespaceErrorDoesNotFallBackToHost(t *testing.T) {
	for _, path := range []string{filepath.Join(t.TempDir(), "missing"), "/dev/null"} {
		if _, err := readP2PTCPSockets(path); err == nil {
			t.Fatalf("invalid namespace %q must fail rather than collect host sockets", path)
		}
	}
}

// Requires CAP_SYS_ADMIN. Check the socket's actual kernel namespace cookie,
// and that the calling thread stayed on the host, without relying on port 4001
// being available on a live node.
func TestTCPNamespaceSocketIsolation(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hostNamespace, err := os.Readlink("/proc/thread-self/ns/net")
	if err != nil {
		t.Fatal(err)
	}
	hostSocket, err := openTCPDiagSocket("")
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(hostSocket)
	hostCookie, err := unix.GetsockoptUint64(hostSocket, unix.SOL_SOCKET, unix.SO_NETNS_COOKIE)
	if errors.Is(err, unix.ENOPROTOOPT) {
		t.Skip("kernel does not expose SO_NETNS_COOKIE for isolation verification")
	}
	if err != nil {
		t.Fatal(err)
	}
	type namespaceResult struct {
		fd  int
		err error
	}
	ch := make(chan namespaceResult, 1)
	goSafe("tcp_peer_bytes", func() {
		runtime.LockOSThread() // This thread is discarded when the worker exits.
		r := namespaceResult{-1, fmt.Errorf("namespace test worker did not complete")}
		defer func() { ch <- r }()
		if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
			r.err = err
			return
		}
		r.fd, r.err = unix.Open("/proc/thread-self/ns/net", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	})
	result := <-ch
	if errors.Is(result.err, unix.EPERM) {
		t.Skip("CAP_SYS_ADMIN required to create an isolated test network namespace")
	}
	if result.err != nil {
		t.Fatal(result.err)
	}
	defer unix.Close(result.fd)
	namespace := fmt.Sprintf("/proc/self/fd/%d", result.fd)
	socket, err := openTCPDiagSocket(namespace)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(socket)
	targetCookie, err := unix.GetsockoptUint64(socket, unix.SOL_SOCKET, unix.SO_NETNS_COOKIE)
	if err != nil {
		t.Fatal(err)
	}
	if targetCookie == hostCookie {
		t.Fatal("diagnostic socket was opened in the host namespace")
	}
	for _, family := range []uint8{linuxAFInet, linuxAFInet6} {
		rows, err := dumpP2PTCPSocketsOnSocket(socket, family)
		if family == linuxAFInet6 && (errors.Is(err, unix.EAFNOSUPPORT) || errors.Is(err, unix.EPROTONOSUPPORT)) {
			continue
		}
		if err != nil || len(rows) != 0 {
			t.Fatalf("empty target namespace: %d sockets, err=%v", len(rows), err)
		}
	}
	after, err := os.Readlink("/proc/thread-self/ns/net")
	if err != nil || after != hostNamespace {
		t.Fatalf("caller namespace changed: before=%s after=%s err=%v", hostNamespace, after, err)
	}
}
