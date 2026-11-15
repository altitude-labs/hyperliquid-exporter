package monitors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/validaoxyz/hyperliquid-exporter/internal/config"
	"github.com/validaoxyz/hyperliquid-exporter/internal/logger"
	"github.com/validaoxyz/hyperliquid-exporter/internal/metrics"
)

const (
	tcpPeerBytesInterval = 15 * time.Second
	tcpPeerBytesPeerCap  = 2048
	tcpPeerBytesSockCap  = 8192
	tcpPeerBytesTTL      = 24 * time.Hour
)

var errTCPSocketStatsUnsupported = errors.New("TCP socket byte collection requires Linux")

type tcpSocketSample struct {
	cookie   uint64 // Kernel socket identity, stable across dumps, unlike a port tuple.
	ip       string
	received uint64
	sent     uint64
}

// tcpPeerByteTracker turns socket-lifetime totals into exporter counters.
// The first complete dump establishes a baseline: existing traffic must not
// appear as new client consumption when the exporter starts.
type tcpPeerByteTracker struct {
	sockets     map[uint64]tcpSocketSample
	peers       map[string]time.Time
	initialized bool
	bytes       *prometheus.CounterVec
}

func newTCPPeerByteTracker(bytes *prometheus.CounterVec) *tcpPeerByteTracker {
	return &tcpPeerByteTracker{
		sockets: make(map[uint64]tcpSocketSample),
		peers:   make(map[string]time.Time),
		bytes:   bytes,
	}
}

func (t *tcpPeerByteTracker) observe(samples []tcpSocketSample, now time.Time) error {
	if len(samples) > tcpPeerBytesSockCap {
		return fmt.Errorf("P2P socket count exceeds %d; dump discarded", tcpPeerBytesSockCap)
	}
	active := make(map[string]bool)
	next := make(map[uint64]tcpSocketSample, len(samples))
	for _, s := range samples {
		if _, exists := next[s.cookie]; exists {
			return fmt.Errorf("duplicate P2P socket cookie %d; dump discarded", s.cookie)
		}
		next[s.cookie] = s
		active[s.ip] = true
	}
	if len(active) > tcpPeerBytesPeerCap {
		return fmt.Errorf("active P2P IP count exceeds %d; dump discarded", tcpPeerBytesPeerCap)
	}

	remove := func(ip string) {
		t.bytes.DeleteLabelValues(ip, "in")
		t.bytes.DeleteLabelValues(ip, "out")
		delete(t.peers, ip)
	}
	for ip, lastSeen := range t.peers {
		if !active[ip] && now.Sub(lastSeen) >= tcpPeerBytesTTL {
			remove(ip)
		}
	}
	for ip := range active {
		if _, known := t.peers[ip]; !known && len(t.peers) >= tcpPeerBytesPeerCap {
			// Evict only disconnected peers; current connections keep stable counters.
			var oldest string
			for candidate, seen := range t.peers {
				if !active[candidate] && (oldest == "" || seen.Before(t.peers[oldest])) {
					oldest = candidate
				}
			}
			remove(oldest) // There is always an inactive peer when active <= cap.
		}
		t.peers[ip] = now
		t.bytes.WithLabelValues(ip, "in").Add(0)
		t.bytes.WithLabelValues(ip, "out").Add(0)
	}

	for _, s := range samples {
		if !t.initialized {
			continue
		}
		previous, exists := t.sockets[s.cookie]
		if !exists {
			// A newly observed socket was opened after our baseline. Capture
			// its initial bytes too, rather than losing the first poll interval.
			previous = tcpSocketSample{}
		}
		if s.received >= previous.received {
			t.bytes.WithLabelValues(s.ip, "in").Add(float64(s.received - previous.received))
		}
		if s.sent >= previous.sent {
			t.bytes.WithLabelValues(s.ip, "out").Add(float64(s.sent - previous.sent))
		}
	}
	t.sockets = next
	t.initialized = true
	return nil
}

// StartTCPPeerBytesMonitor samples kernel TCP_INFO counters in the node's
// network namespace. This is opt-in with --per-peer-metrics. Short-lived
// sockets and the tail after the last observed sample can be missed; these
// are operational measurements, not a billing ledger.
func StartTCPPeerBytesMonitor(ctx context.Context, cfg config.Config, errCh chan<- error) {
	tracker := newTCPPeerByteTracker(metrics.HLP2PTCPBytesTotal)
	ticker := time.NewTicker(tcpPeerBytesInterval)
	defer ticker.Stop()
	for {
		samples, err := readP2PTCPSockets()
		if errors.Is(err, errTCPSocketStatsUnsupported) {
			logger.InfoComponent("tcp_peer_bytes", "%v; monitor idle", err)
			<-ctx.Done()
			return
		}
		if err == nil {
			err = tracker.observe(samples, time.Now())
		}
		if err != nil {
			metrics.HLP2PTCPCollectionSuccess.Set(0)
			select {
			case errCh <- err:
			default:
			}
		} else {
			metrics.HLP2PTCPCollectionSuccess.Set(1)
			metrics.HLP2PTCPLastSuccessSeconds.Set(float64(time.Now().Unix()))
			metrics.MarkMonitorTick("tcp_peer_bytes")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
