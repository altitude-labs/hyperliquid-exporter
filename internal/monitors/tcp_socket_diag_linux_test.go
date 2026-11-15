//go:build linux

package monitors

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
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
