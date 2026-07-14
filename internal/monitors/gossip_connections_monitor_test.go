package monitors

import (
	"testing"
	"time"
)

func TestParseGossipConnectionLine_KnownEvents(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		wantEvent string
		wantIP    string
	}{
		{
			name:      "verified gossip rpc",
			line:      `["2026-05-25T06:59:53.606639758",["verified gossip rpc",{"Ip":"203.0.113.75"}]]`,
			wantEvent: "verified_gossip_rpc",
			wantIP:    "203.0.113.75",
		},
		{
			name:      "performing checks on stream",
			line:      `["2026-05-25T06:59:53.6",["performing checks on stream",{"foo":1}]]`,
			wantEvent: "performing_checks_on_stream",
		},
		{
			name:      "error checking connection",
			line:      `["2026-05-25T06:59:53.6",["error checking connection",{"err":"x"}]]`,
			wantEvent: "error_checking_connection",
		},
		{
			name:      "rejecting gossip stream because max peers reached",
			line:      `["2026-05-25T06:59:53.6",["rejecting gossip stream because max peers reached",{}]]`,
			wantEvent: "rejecting_gossip_stream_max_peers_reached",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseGossipConnectionLine([]byte(c.line))
			if !ok {
				t.Fatalf("expected ok=true on %q", c.line)
			}
			if got.eventType != c.wantEvent {
				t.Errorf("event type = %q, want %q", got.eventType, c.wantEvent)
			}
			if got.ip != c.wantIP {
				t.Errorf("ip = %q, want %q", got.ip, c.wantIP)
			}
			if got.timestamp.IsZero() {
				t.Error("expected parsed source timestamp")
			}
		})
	}
}

func TestParseGossipConnectionLine_OtherFallback(t *testing.T) {
	line := []byte(`["2026-05-25T06:59:53.6",["some new event we have not seen yet",{}]]`)
	got, ok := parseGossipConnectionLine(line)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got.eventType != "other" {
		t.Errorf("expected 'other', got %q", got.eventType)
	}
}

func TestParseGossipConnectionLine_NestedAndSocketIPs(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "nested remote ip with port",
			line: `["2026-05-25T06:59:53.6",["handle_stream_connection",{"peer":{"remote_ip":"64.31.48.111:4001"}}]]`,
			want: "64.31.48.111",
		},
		{
			name: "ipv6 socket",
			line: `["2026-05-25T06:59:53.6",["got tcp greeting",{"peer_ip":"[2001:db8::1]:4002"}]]`,
			want: "2001:db8::1",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseGossipConnectionLine([]byte(c.line))
			if !ok {
				t.Fatal("expected parse success")
			}
			if got.ip != c.want {
				t.Errorf("ip = %q, want %q", got.ip, c.want)
			}
		})
	}
}

func TestParseGossipConnectionLine_DoesNotScrapeErrorText(t *testing.T) {
	line := []byte(`["2026-05-25T06:59:53.6",["error checking connection",{"err":"dial 64.31.51.137:4001: refused"}]]`)
	got, ok := parseGossipConnectionLine(line)
	if !ok {
		t.Fatal("expected parse success")
	}
	if got.ip != "" {
		t.Errorf("unexpected IP attribution from error text: %q", got.ip)
	}
}

func TestGossipPeerMetricTracker_BoundsAndExpiresPeers(t *testing.T) {
	tracker := newGossipPeerMetricTracker(2, time.Hour)
	base := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)

	tracker.observe("192.0.2.1", "verified_gossip_rpc", base)
	tracker.observe("192.0.2.2", "verified_gossip_rpc", base.Add(time.Minute))
	tracker.observe("192.0.2.3", "verified_gossip_rpc", base.Add(2*time.Minute))

	if len(tracker.peers) != 2 {
		t.Fatalf("peer count = %d, want 2", len(tracker.peers))
	}
	if _, exists := tracker.peers["192.0.2.1"]; exists {
		t.Error("oldest peer was not LRU-evicted")
	}

	tracker.evictExpired(base.Add(2 * time.Hour))
	if len(tracker.peers) != 0 {
		t.Errorf("peer count after TTL eviction = %d, want 0", len(tracker.peers))
	}
}

func TestParseGossipConnectionLine_Malformed(t *testing.T) {
	for _, line := range [][]byte{
		[]byte(""),
		[]byte("not json"),
		[]byte("{}"),
		[]byte(`["only timestamp"]`),
		[]byte(`["ts", "not an array"]`),
		[]byte(`["ts", []]`),
	} {
		_, ok := parseGossipConnectionLine(line)
		if ok {
			t.Errorf("expected fail on %q", line)
		}
	}
}
