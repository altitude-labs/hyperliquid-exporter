package monitors

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

// Linux UAPI layouts, decoded without unsafe casts so truncated TCP_INFO
// from older kernels is rejected rather than silently reported as zero.
// https://github.com/torvalds/linux/blob/master/include/uapi/linux/inet_diag.h
// https://github.com/torvalds/linux/blob/master/include/uapi/linux/tcp.h
const (
	inetDiagMessageSize = 72
	inetDiagInfo        = 2
	tcpInfoBytesRecv    = 128
	tcpInfoBytesSent    = 200
	linuxAFInet         = 2
	linuxAFInet6        = 10
)

func isP2PPort(port uint16) bool { return port == 4001 || port == 4002 }

func parseP2PSocketDiag(data []byte) (tcpSocketSample, bool, error) {
	var s tcpSocketSample
	if len(data) < inetDiagMessageSize {
		return s, false, fmt.Errorf("short inet_diag message: %d bytes", len(data))
	}
	if !isP2PPort(binary.BigEndian.Uint16(data[4:6])) && !isP2PPort(binary.BigEndian.Uint16(data[6:8])) {
		return s, false, nil
	}
	switch data[0] {
	case linuxAFInet:
		s.ip = netip.AddrFrom4([4]byte(data[24:28])).String()
	case linuxAFInet6:
		s.ip = netip.AddrFrom16([16]byte(data[24:40])).Unmap().String()
	default:
		return s, false, fmt.Errorf("unexpected inet_diag address family %d", data[0])
	}
	s.cookie = uint64(binary.NativeEndian.Uint32(data[44:48])) | uint64(binary.NativeEndian.Uint32(data[48:52]))<<32
	if s.cookie == 0 || s.cookie == ^uint64(0) {
		return s, false, fmt.Errorf("P2P socket has no usable kernel cookie")
	}
	for attrs := data[inetDiagMessageSize:]; len(attrs) > 0; {
		if len(attrs) < 4 {
			return s, false, fmt.Errorf("short inet_diag attribute header")
		}
		n := int(binary.NativeEndian.Uint16(attrs[:2]))
		if n < 4 || n > len(attrs) {
			return s, false, fmt.Errorf("invalid inet_diag attribute length %d", n)
		}
		if binary.NativeEndian.Uint16(attrs[2:4])&0x3fff == inetDiagInfo {
			info := attrs[4:n]
			if len(info) < tcpInfoBytesSent+8 {
				return s, false, fmt.Errorf("TCP_INFO lacks byte counters (need Linux 4.19+): %d bytes", len(info))
			}
			s.received = binary.NativeEndian.Uint64(info[tcpInfoBytesRecv : tcpInfoBytesRecv+8])
			s.sent = binary.NativeEndian.Uint64(info[tcpInfoBytesSent : tcpInfoBytesSent+8])
			return s, true, nil
		}
		aligned := (n + 3) &^ 3
		if aligned > len(attrs) {
			return s, false, fmt.Errorf("short inet_diag attribute padding")
		}
		attrs = attrs[aligned:]
	}
	return s, false, fmt.Errorf("P2P socket missing TCP_INFO")
}
