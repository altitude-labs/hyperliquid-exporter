package monitors

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func testTCPByteTracker() *tcpPeerByteTracker {
	return newTCPPeerByteTracker(prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_tcp_bytes_total", Help: "Test bytes.",
	}, []string{"ip", "direction"}))
}

func assertTCPBytes(t *testing.T, tracker *tcpPeerByteTracker, ip, direction string, want float64) {
	t.Helper()
	var m dto.Metric
	if err := tracker.bytes.WithLabelValues(ip, direction).Write(&m); err != nil {
		t.Fatal(err)
	}
	if got := m.GetCounter().GetValue(); got != want {
		t.Fatalf("%s %s: got %v, want %v", ip, direction, got, want)
	}
}

func TestTCPPeerBytesBaselineDeltasAndReconnect(t *testing.T) {
	tracker := testTCPByteTracker()
	now := time.Now()
	observe := func(samples ...tcpSocketSample) {
		t.Helper()
		if err := tracker.observe(samples, now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(tcpPeerBytesInterval)
	}
	const ip = "203.0.113.1"
	// Two sockets (e.g. both ports) aggregate into one IP. Existing lifetime
	// totals are baselined and repeated snapshots never inflate consumption.
	observe(tcpSocketSample{1, ip, 1000, 10000}, tcpSocketSample{2, ip, 2000, 20000})
	assertTCPBytes(t, tracker, ip, "in", 0)
	assertTCPBytes(t, tracker, ip, "out", 0)
	observe(tcpSocketSample{1, ip, 1100, 12000}, tcpSocketSample{2, ip, 2200, 23000})
	assertTCPBytes(t, tracker, ip, "in", 300)
	assertTCPBytes(t, tracker, ip, "out", 5000)
	observe(tcpSocketSample{1, ip, 1100, 12000}, tcpSocketSample{2, ip, 2200, 23000})
	assertTCPBytes(t, tracker, ip, "out", 5000)
	// Disconnecting removes socket state but keeps the peer's counters.
	observe()
	observe(tcpSocketSample{3, ip, 50, 500})
	assertTCPBytes(t, tracker, ip, "in", 350)
	assertTCPBytes(t, tracker, ip, "out", 5500)
	// A regressing source never underflows into a huge traffic spike.
	observe(tcpSocketSample{3, ip, 5, 10})
	observe(tcpSocketSample{3, ip, 15, 30})
	assertTCPBytes(t, tracker, ip, "in", 360)
	assertTCPBytes(t, tracker, ip, "out", 5520)
}

func TestTCPPeerBytesRejectsPartialInputAndExpiresIdlePeers(t *testing.T) {
	tracker := testTCPByteTracker()
	now := time.Now()
	const ip = "203.0.113.1"
	baseline := []tcpSocketSample{{1, ip, 100, 1000}}
	if err := tracker.observe(baseline, now); err != nil {
		t.Fatal(err)
	}
	duplicate := []tcpSocketSample{{1, ip, 200, 2000}, {1, ip, 200, 2000}}
	if tracker.observe(duplicate, now) == nil {
		t.Fatal("duplicate cookies must discard the entire dump")
	}
	if tracker.sockets[1].sent != 1000 {
		t.Fatal("rejected dump changed baseline")
	}
	if err := tracker.observe(nil, now.Add(tcpPeerBytesTTL)); err != nil {
		t.Fatal(err)
	}
	if len(tracker.peers) != 0 || len(tracker.sockets) != 0 {
		t.Fatal("idle peer state was not removed")
	}
	ch := make(chan prometheus.Metric, 2)
	tracker.bytes.Collect(ch)
	if len(ch) != 0 {
		t.Fatal("expired metric series were not removed")
	}
}

func TestTCPPeerBytesCardinality(t *testing.T) {
	tracker := testTCPByteTracker()
	now := time.Now()
	samples := make([]tcpSocketSample, tcpPeerBytesPeerCap)
	for i := range samples {
		samples[i] = tcpSocketSample{cookie: uint64(i + 1), ip: netip.AddrFrom4([4]byte{10, 0, byte(i >> 8), byte(i)}).String()}
	}
	if err := tracker.observe(samples, now); err != nil {
		t.Fatal(err)
	}
	// Admit a replacement without evicting any currently active IPs.
	samples[0] = tcpSocketSample{cookie: 99999, ip: "203.0.113.9", sent: 42}
	if err := tracker.observe(samples, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(tracker.peers) != tcpPeerBytesPeerCap {
		t.Fatal("peer cap not maintained")
	}
	if _, exists := tracker.peers["10.0.0.0"]; exists {
		t.Fatal("old disconnected peer not evicted")
	}
	assertTCPBytes(t, tracker, "203.0.113.9", "out", 42)
	samples = append(samples, tcpSocketSample{cookie: 99998, ip: "203.0.113.10"})
	if tracker.observe(samples, now) == nil {
		t.Fatal("dump exceeding the active IP cap must fail")
	}
	if tracker.observe(make([]tcpSocketSample, tcpPeerBytesSockCap+1), now) == nil {
		t.Fatal("dump exceeding the socket cap must fail")
	}
}

func socketDiagFixture(ip string, localPort, remotePort uint16) []byte {
	data := make([]byte, inetDiagMessageSize+4+216)
	addr := netip.MustParseAddr(ip)
	if addr.Is4() {
		data[0] = linuxAFInet
		v := addr.As4()
		copy(data[24:28], v[:])
	} else {
		data[0] = linuxAFInet6
		v := addr.As16()
		copy(data[24:40], v[:])
	}
	binary.BigEndian.PutUint16(data[4:6], localPort)
	binary.BigEndian.PutUint16(data[6:8], remotePort)
	binary.NativeEndian.PutUint32(data[44:48], 42)
	binary.NativeEndian.PutUint32(data[48:52], 7)
	attrs := data[inetDiagMessageSize:]
	binary.NativeEndian.PutUint16(attrs[:2], uint16(len(attrs)))
	binary.NativeEndian.PutUint16(attrs[2:4], inetDiagInfo)
	binary.NativeEndian.PutUint64(attrs[4+tcpInfoBytesRecv:], 123456789)
	binary.NativeEndian.PutUint64(attrs[4+tcpInfoBytesSent:], 987654321)
	return data
}

func TestParseP2PSocketDiag(t *testing.T) {
	for _, tc := range []struct {
		ip, wantIP    string
		local, remote uint16
	}{
		{"203.0.113.9", "203.0.113.9", 4001, 50000},
		{"2001:db8::42", "2001:db8::42", 50000, 4002},
		{"::ffff:203.0.113.9", "203.0.113.9", 4002, 50000},
	} {
		t.Run(tc.ip, func(t *testing.T) {
			s, relevant, err := parseP2PSocketDiag(socketDiagFixture(tc.ip, tc.local, tc.remote))
			if err != nil || !relevant || s.ip != tc.wantIP || s.cookie != (7<<32)|42 || s.received != 123456789 || s.sent != 987654321 {
				t.Fatalf("decoded %+v, relevant=%v, err=%v", s, relevant, err)
			}
		})
	}
	// RPC and other host traffic never enters the P2P counters.
	if _, relevant, err := parseP2PSocketDiag(socketDiagFixture("203.0.113.9", 3001, 50000)); err != nil || relevant {
		t.Fatalf("RPC socket included: relevant=%v, err=%v", relevant, err)
	}
}

func TestParseP2PSocketDiagRejectsMissingOrTruncatedInfo(t *testing.T) {
	valid := socketDiagFixture("203.0.113.9", 4001, 50000)
	for _, data := range [][]byte{nil, valid[:10], valid[:inetDiagMessageSize], valid[:inetDiagMessageSize+2], valid[:len(valid)-1]} {
		if _, _, err := parseP2PSocketDiag(data); err == nil {
			t.Fatalf("accepted truncated message of length %d", len(data))
		}
	}
	oldKernel := valid[:inetDiagMessageSize+4+136]
	binary.NativeEndian.PutUint16(oldKernel[inetDiagMessageSize:], 4+136)
	if _, _, err := parseP2PSocketDiag(oldKernel); err == nil {
		t.Fatal("old TCP_INFO must fail explicitly rather than report zero outgoing bytes")
	}
}
