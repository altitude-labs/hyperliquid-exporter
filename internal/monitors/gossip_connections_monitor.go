package monitors

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/validaoxyz/hyperliquid-exporter/internal/config"
	"github.com/validaoxyz/hyperliquid-exporter/internal/logger"
	"github.com/validaoxyz/hyperliquid-exporter/internal/metrics"
	"github.com/validaoxyz/hyperliquid-exporter/internal/peerset"
)

// gossipConnectionEvents lists the event tag names the node emits to
// gossip_connections logs. The mapping was derived from a live mainnet
// peer's most recent hour-file. New tags appearing in the future will
// still be counted but will land in an "other" bucket so cardinality
// stays predictable.
//
// Operator semantics:
//   - rejecting_gossip_stream_max_peers_reached: capacity exhausted!
//   - error_checking_connection: peer churn / unreliable peers
//   - verified_gossip_rpc:       healthy handshake (counter rate = arrival rate)
var gossipConnectionAllowlist = map[string]string{
	"verified gossip rpc":                               "verified_gossip_rpc",
	"handle_stream_connection":                          "handle_stream_connection",
	"performing checks on stream":                       "performing_checks_on_stream",
	"got tcp greeting":                                  "got_tcp_greeting",
	"error checking connection":                         "error_checking_connection",
	"rejecting gossip stream because max peers reached": "rejecting_gossip_stream_max_peers_reached",
}

// gossipConnectionsPollInterval is short because the file is append-only
// and operators want to see peer-rejection events promptly.
const gossipConnectionsPollInterval = 10 * time.Second

const (
	gossipPeerMetricCap = 2048
	gossipPeerMetricTTL = 24 * time.Hour
)

type gossipConnectionEvent struct {
	eventType string
	ip        string
	timestamp time.Time
}

type gossipPeerMetricState struct {
	lastSeen   time.Time
	eventTypes map[string]struct{}
}

// gossipPeerMetricTracker bounds the active per-IP counter series. Counters
// are deleted when an IP expires or is LRU-evicted; Prometheus marks the old
// series stale and handles a later reappearance as a counter reset.
type gossipPeerMetricTracker struct {
	cap   int
	ttl   time.Duration
	peers map[string]*gossipPeerMetricState
}

func newGossipPeerMetricTracker(capacity int, ttl time.Duration) *gossipPeerMetricTracker {
	return &gossipPeerMetricTracker{
		cap:   capacity,
		ttl:   ttl,
		peers: make(map[string]*gossipPeerMetricState, capacity),
	}
}

// StartGossipConnectionsMonitor watches
// $NODE_HOME/data/node_logs/gossip_connections/hourly. Each line is of
// the form ["<iso>", ["<event_tag>", {<event_specific_payload>}]]. We
// extract only the tag and count occurrences, keeping cardinality flat.
func StartGossipConnectionsMonitor(ctx context.Context, cfg config.Config, errCh chan<- error) {
	root := filepath.Join(cfg.NodeHome, "data", "node_logs", "gossip_connections", "hourly")

	if _, err := os.Stat(root); err != nil {
		logger.InfoComponent("gossip_connections",
			"gossip_connections directory not present (%s); monitor idle", root)
		<-ctx.Done()
		return
	}

	logger.InfoComponent("gossip_connections", "watching %s", root)

	var currentFile string
	var currentOffset int64
	var peerMetrics *gossipPeerMetricTracker
	if cfg.EnablePerPeerMetrics {
		peerMetrics = newGossipPeerMetricTracker(gossipPeerMetricCap, gossipPeerMetricTTL)
	}

	tick := func() {
		latest, err := latestHourlyFile(root)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				logger.DebugComponent("gossip_connections", "find latest: %v", err)
			}
			return
		}
		if latest != currentFile {
			// On startup or rotation, jump to end so we don't replay history
			// (counters would otherwise spike on every restart).
			info, err := os.Stat(latest)
			if err != nil {
				return
			}
			logger.InfoComponent("gossip_connections", "switched to %s (starting at EOF=%d)", latest, info.Size())
			currentFile = latest
			currentOffset = info.Size()
			return
		}
		if peerMetrics != nil {
			peerMetrics.evictExpired(time.Now())
		}
		newOffset, n, err := readGossipConnectionEvents(currentFile, currentOffset, peerMetrics)
		if err != nil {
			logger.DebugComponent("gossip_connections", "read %s: %v", currentFile, err)
			return
		}
		currentOffset = newOffset
		if n > 0 {
			metrics.MarkMonitorTick("gossip_connections")
		}
	}

	ticker := time.NewTicker(gossipConnectionsPollInterval)
	defer ticker.Stop()
	tick()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
}

// readGossipConnectionEvents reads any new bytes appended to path since
// offset and increments the aggregate and optional per-IP counters for each
// parseable event. Returns the new file offset and number of lines processed.
func readGossipConnectionEvents(path string, offset int64, peerMetrics *gossipPeerMetricTracker) (int64, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return offset, 0, err
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, 0, err
	}
	reader := bufio.NewReaderSize(f, 1<<20)
	processed := 0
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if event, ok := parseGossipConnectionLine(line); ok {
				metrics.HLP2PGossipEventsTotal.WithLabelValues(event.eventType).Inc()
				if event.ip == "" {
					metrics.HLP2PGossipEventsUnattributedTotal.WithLabelValues(event.eventType).Inc()
				} else {
					observedAt := event.timestamp
					if observedAt.IsZero() {
						observedAt = time.Now()
					}
					if evictedIP, evicted := peerset.Default().Register(event.ip, "", observedAt); evicted && peerMetrics != nil {
						peerMetrics.remove(evictedIP)
					}
					if peerMetrics != nil {
						peerMetrics.observe(event.ip, event.eventType, observedAt)
					}
				}
				processed++
			}
			offset += int64(len(line))
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return offset, processed, err
		}
	}
	return offset, processed, nil
}

// parseGossipConnectionLine returns the sanitized event type, source
// timestamp, and remote IP when the structured payload contains one. It does
// not scrape arbitrary error text for addresses: unattributed events remain
// visible through hl_p2p_gossip_events_unattributed_total instead of risking
// a false attribution. Untracked tags land in the bounded "other" bucket.
func parseGossipConnectionLine(line []byte) (gossipConnectionEvent, bool) {
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 || line[0] != '[' {
		return gossipConnectionEvent{}, false
	}
	var outer [2]json.RawMessage
	if err := json.Unmarshal(line, &outer); err != nil {
		return gossipConnectionEvent{}, false
	}
	var timestampString string
	_ = json.Unmarshal(outer[0], &timestampString)
	timestamp, _ := parseVisorTime(timestampString)

	var inner []json.RawMessage
	if err := json.Unmarshal(outer[1], &inner); err != nil || len(inner) < 1 {
		return gossipConnectionEvent{}, false
	}
	var tag string
	if err := json.Unmarshal(inner[0], &tag); err != nil {
		return gossipConnectionEvent{}, false
	}
	eventType := "other"
	if name, known := gossipConnectionAllowlist[tag]; known {
		eventType = name
	}

	var ip string
	for _, payload := range inner[1:] {
		if candidate, ok := extractStructuredIP(payload); ok {
			ip = candidate
			break
		}
	}
	return gossipConnectionEvent{eventType: eventType, ip: ip, timestamp: timestamp}, true
}

func extractStructuredIP(raw json.RawMessage) (string, bool) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return findStructuredIP(value)
}

func findStructuredIP(value any) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		// Prefer fields explicitly named as IPs before descending into nested
		// payloads. Known hl-node variants use Ip; remote_ip and peer_ip are
		// accepted for forward compatibility.
		for _, preferred := range []string{"remote_ip", "peer_ip", "ip"} {
			for key, candidate := range typed {
				if normalizeIPField(key) != preferred {
					continue
				}
				if text, ok := candidate.(string); ok {
					if ip, valid := normalizeIP(text); valid {
						return ip, true
					}
				}
			}
		}
		for key, candidate := range typed {
			if !isIPField(key) {
				continue
			}
			if text, ok := candidate.(string); ok {
				if ip, valid := normalizeIP(text); valid {
					return ip, true
				}
			}
		}
		for _, nested := range typed {
			if ip, ok := findStructuredIP(nested); ok {
				return ip, true
			}
		}
	case []any:
		for _, nested := range typed {
			if ip, ok := findStructuredIP(nested); ok {
				return ip, true
			}
		}
	}
	return "", false
}

func isIPField(key string) bool {
	normalized := normalizeIPField(key)
	return normalized == "ip" || strings.HasSuffix(normalized, "_ip")
}

func normalizeIPField(key string) string {
	return strings.ToLower(strings.ReplaceAll(key, "-", "_"))
}

func normalizeIP(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if ip := net.ParseIP(strings.Trim(value, "[]")); ip != nil {
		return ip.String(), true
	}
	host, _, err := net.SplitHostPort(value)
	if err != nil {
		return "", false
	}
	if zone := strings.LastIndexByte(host, '%'); zone >= 0 {
		host = host[:zone]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}
	return ip.String(), true
}

func (t *gossipPeerMetricTracker) observe(ip, eventType string, observedAt time.Time) {
	state, exists := t.peers[ip]
	if !exists {
		if t.cap <= 0 {
			return
		}
		if len(t.peers) >= t.cap {
			t.remove(t.oldestIP())
		}
		state = &gossipPeerMetricState{eventTypes: make(map[string]struct{})}
		t.peers[ip] = state
	}
	if observedAt.After(state.lastSeen) {
		state.lastSeen = observedAt
	}
	state.eventTypes[eventType] = struct{}{}
	metrics.HLP2PGossipPeerEventsTotal.WithLabelValues(ip, eventType).Inc()
}

func (t *gossipPeerMetricTracker) evictExpired(now time.Time) {
	cutoff := now.Add(-t.ttl)
	for ip, state := range t.peers {
		if state.lastSeen.Before(cutoff) {
			t.remove(ip)
		}
	}
}

func (t *gossipPeerMetricTracker) oldestIP() string {
	var oldestIP string
	var oldestTime time.Time
	for ip, state := range t.peers {
		if oldestIP == "" || state.lastSeen.Before(oldestTime) {
			oldestIP = ip
			oldestTime = state.lastSeen
		}
	}
	return oldestIP
}

func (t *gossipPeerMetricTracker) remove(ip string) {
	state, exists := t.peers[ip]
	if !exists {
		return
	}
	for eventType := range state.eventTypes {
		metrics.HLP2PGossipPeerEventsTotal.DeleteLabelValues(ip, eventType)
	}
	delete(t.peers, ip)
}
