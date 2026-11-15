# Hyperliquid Exporter Metrics Reference

> **Legend - `_total` on Gauge-typed metrics:** Gauge-typed metrics whose names end in `_total` are cumulative-since-source-restart snapshots (or current counts), NOT Prometheus counters: they reset to 0 when the source process (hl-node/hl-visor) or the exporter restarts, and some are point-in-time counts that can DECREASE. Do not apply `rate()`/`increase()` to them; use the raw value, or `delta()`/`changes()` with awareness of resets. True counters (correctly consumed with `rate()`) include `hl_p2p_tcp_bytes_total`, `hl_info_endpoint_failures_total`, `hl_node_parent_peer_switches_total`, the `hl_p2p_gossip_*_total` event counters, `hl_node_public_ip_changes_total`, `hl_mempool_events_total`, and the `hl_mempool_txs_*_total` counters.

## HyperCore Metrics

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_core_block_height` | Gauge | - | Current block height. In dual-state mode this tracks the **fast** state only (slow state never updates it). | - |
| `hl_core_blocks_processed_total` | Counter | - | Total blocks processed | `--replica-metrics` |
| `hl_core_block_time_milliseconds` | Histogram | `state_type` | Time between blocks in milliseconds. `state_type` (`fast`/`slow`) is present only in dual-state mode; on the legacy single `block_times` path the metric is emitted without it. | - |
| `hl_core_latest_block_time` | Gauge | - | Unix timestamp of latest block | - |
| `hl_core_last_processed_round` | Gauge | - | Last processed consensus round | `--replica-metrics` |
| `hl_core_last_processed_time` | Gauge | - | Unix timestamp of last processed block | `--replica-metrics` |
| `hl_core_operations_per_block` | Histogram | - | Distribution of operations per block | `--replica-metrics` |
| `hl_core_operations_total` | Counter | `type`, `category` | Total individual operations by type and category | `--replica-metrics` |
| `hl_core_orders_per_block` | Histogram | - | Distribution of orders per block | `--replica-metrics` |
| `hl_core_orders_total` | Counter | - | Total orders placed | `--replica-metrics` |
| `hl_core_rounds_processed_total` | Counter | - | Total consensus rounds processed. Equals `hl_core_blocks_processed_total` (incremented once per block; does not separately count timeout/QC rounds). | `--replica-metrics` |
| `hl_core_tx_per_block` | Histogram | - | Distribution of transactions per block | `--replica-metrics` |
| `hl_core_tx_total` | Counter | `type` | Total transactions/actions by type | `--replica-metrics` |

Metrics marked with `--replica-metrics` also require hl-node to be running with `--replica-cmds-style actions-and-responses`.

## EVM Metrics

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_evm_account_count` | Gauge | - | Total number of EVM accounts | - |
| `hl_evm_base_fee_gwei` | Gauge | `block_type` | Current base fee in Gwei | - |
| `hl_evm_base_fee_gwei_distribution` | Histogram | `block_type` | Distribution of base fees (0-1000+ Gwei buckets) | - |
| `hl_evm_block_height` | Gauge | - | Current EVM block height | - |
| `hl_evm_block_time_milliseconds` | Histogram | - | Time between EVM blocks | - |
| `hl_evm_contract_create_total` | Counter | `block_type` | Total contract creations | - |
| `hl_evm_contract_tx_total` | Counter | `contract_address`, `contract_name`, `is_token`, `type`, `symbol`, `block_type` | Contract interactions by address. `contract_address` is high-cardinality (`--contract-metrics-limit` bounds the resolver cache, NOT the number of series); `symbol` is present only for tokens. | `--contract-metrics` |
| `hl_evm_gas_limit` | Gauge | `block_type` | Gas limit per block | - |
| `hl_evm_gas_limit_distribution` | Histogram | - | Distribution of gas limits across blocks | - |
| `hl_evm_gas_used` | Gauge | `block_type` | Gas used per block | - |
| `hl_evm_gas_util` | Gauge | `block_type` | Gas utilization fraction (0-1, gasUsed/gasLimit) | - |
| `hl_evm_high_gas_limit_blocks_total` | Counter | `threshold` | Count of high gas limit blocks by threshold | - |
| `hl_evm_last_high_gas_block_height` | Gauge | - | Height of last high gas block | - |
| `hl_evm_last_high_gas_block_limit` | Gauge | - | Gas limit of last high gas block | - |
| `hl_evm_last_high_gas_block_time` | Gauge | - | Unix timestamp of last high gas block | - |
| `hl_evm_last_high_gas_block_used` | Gauge | - | Gas used in last high gas block | - |
| `hl_evm_latest_block_time` | Gauge | - | Unix timestamp of latest EVM block | - |
| `hl_evm_max_gas_limit_seen` | Gauge | - | Maximum gas limit observed | - |
| `hl_evm_max_priority_fee_gwei` | Gauge | `block_type` | Current maximum priority fee (Gwei) | - |
| `hl_evm_max_priority_fee_gwei_distribution` | Histogram | `block_type` | Distribution of priority fees (0-1000+ Gwei buckets) | - |
| `hl_evm_tx_per_block` | Histogram | `block_type` | Distribution of transactions per block | - |
| `hl_evm_tx_type_total` | Counter | `type`, `block_type` | Transaction counts by type (Legacy, Eip1559, Eip2930, Eip4844) | - |


## Machine Metrics

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_metal_apply_duration_milliseconds` | Histogram | `state_type` | Distribution of block application durations | - |
| `hl_metal_last_processed_round` | Gauge | - | Last round processed by replica monitor. Alias of `hl_core_last_processed_round` (same producer, same value). | `--replica-metrics` |
| `hl_metal_last_processed_time` | Gauge | - | Unix timestamp of last processing by replica monitor. Alias of `hl_core_last_processed_time` (same producer, same value). | `--replica-metrics` |
| `hl_metal_parse_duration` | Gauge | - | Duration of replica transaction parsing in seconds | `--replica-metrics` |

## P2P Network Metrics

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_p2p_non_val_peer_connections` | Gauge | `verified` | Number of non-validator peer connections by verification status | gossip_rpc logs |
| `hl_p2p_non_val_peers_total` | Gauge | - | Total number of connected non-validator peers | gossip_rpc logs |

P2P metrics track non-validator peer connections only

## Software Version Metrics

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_software_up_to_date` | Gauge | - | Whether software is up to date (0=outdated, 1=current). UNSET (absent, not 0) until both the local version probe and the upstream binary check have succeeded at least once. | - |
| `hl_software_version` | Gauge | `date`, `commit` | Software version info (always 1, version in labels) | - |


### Consensus metrics

### Basic Validator Metrics

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_consensus_active_stake` | Gauge | - | Total stake of active validators | - |
| `hl_consensus_inactive_stake` | Gauge | - | Total stake of inactive validators | - |
| `hl_consensus_jailed_stake` | Gauge | - | Total stake of jailed validators | - |
| `hl_consensus_not_jailed_stake` | Gauge | - | Total stake of non-jailed validators | - |
| `hl_consensus_proposer_count_total` | Counter | `validator`, `signer`, `name` | Blocks proposed per validator | - |
| `hl_consensus_total_stake` | Gauge | - | Total network stake | - |
| `hl_consensus_validator_active_status` | Gauge | `validator`, `signer`, `name` | Validator active status (0=inactive, 1=active) | - |
| `hl_consensus_validator_count` | Gauge | - | Total number of validators | - |
| `hl_consensus_validator_jailed_status` | Gauge | `validator`, `signer`, `name` | Validator jail status (0=not jailed, 1=jailed) | - |
| `hl_consensus_validator_rtt` | Gauge | `validator`, `moniker`, `ip` | Validator response time in milliseconds | Requires RTT monitoring enabled (auto-enabled on validator nodes) |
| `hl_consensus_validator_stake` | Gauge | `validator`, `signer`, `name` | Stake amount per validator | - |

The above are available to all node types. Note that the stake/status series (`hl_consensus_*_stake`, `hl_consensus_total_stake`, `hl_consensus_validator_stake`, `hl_consensus_validator_count`, `hl_consensus_validator_active_status`, `hl_consensus_validator_jailed_status`, `hl_consensus_proposer_count_total`) are **network-wide**, fetched from `api.hyperliquid.xyz` (`validatorSummaries`) - so they are identical on every node and not node-local. The below metrics require access to consensus (running a validator).

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_consensus_current_round` | Gauge | - | Current consensus round from block messages | Validator node |
| `hl_consensus_heartbeat_ack_delay_ms` | Histogram | - | heartbeat acknowledgement delays | Validator node |
| `hl_consensus_heartbeat_ack_received_total` | Counter | `from_validator`, `to_validator`, `from_name`, `to_name` | Heartbeat acknowledgments between validator pairs | Validator node |
| `hl_consensus_heartbeat_sent_total` | Counter | `validator`, `signer`, `name` | Total heartbeats sent by validators | Validator node |
| `hl_consensus_heartbeat_status` | Gauge | `validator`, `signer`, `name`, `status_type` | Heartbeat health metrics (status_type: since_last_success, last_ack_duration) | Validator node |
| `hl_consensus_validator_connectivity` | Gauge | `validator`, `peer`, `validator_name`, `peer_name` | 0 (or series present) = pair currently disconnected; the series is removed when the pair reconnects (value 1 is never emitted). Alert on `==0` / presence, not `==1`. | Validator node |
| `hl_consensus_vote_round` | Gauge | `validator`, `signer`, `name` | Last voting round for each validator | Validator node |
| `hl_consensus_vote_time_diff_seconds` | Gauge | `validator`, `signer`, `name` | Parse-time age of the LAST OBSERVED vote, computed once and frozen. It does NOT climb while a validator is silent, so it cannot by itself detect a stalled validator. | Validator node |
| `hl_consensus_qc_participation_percent` | Gauge | `validator`, `signer`, `name` | Percentage of recent blocks where validator signed QC (100 blocks sliding) | Validator node |
| `hl_consensus_qc_signatures_total` | Counter | `validator`, `signer`, `name` | Cumulative QC signatures by each validator | Validator node |
| `hl_consensus_qc_size` | Histogram | - | Distribution of QC signer counts per block | Validator node |
| `hl_consensus_rounds_per_block` | Gauge | - | Round advance for the most recent block (single sample, ~1 on a healthy chain), not a moving average | Validator node |
| `hl_consensus_qc_round_lag` | Gauge | - | Average difference between block round and QC round | Validator node |
| `hl_consensus_tc_blocks_total` | Counter | `proposer`, `signer`, `name` | Total blocks proposed containing timeout certificates | Validator node |
| `hl_consensus_tc_participation_total` | Counter | `validator`, `signer`, `name` | Total timeout votes sent by each validator | Validator node |
| `hl_consensus_tc_size` | Histogram | - | Distribution of timeout vote counts in TC blocks | Validator node |
| `hl_consensus_validator_latency_seconds` | Gauge | `validator`, `signer`, `name` | Per-validator raw latency (latest sample). Same `hl_consensus_*` family as the EMA panel; populated by the validator_latency monitor. | Validator node with latency files |
| `hl_consensus_validator_latency_round` | Gauge | `validator`, `signer`, `name` | Consensus round when latency was last measured | Validator node with latency monitoring |
| `hl_consensus_validator_latency_ema_seconds` | Gauge | `validator`, `signer`, `name` | Exponential moving average of validator latency | Validator node with latency monitoring |

## Visor / Sync

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_visor_height` | Gauge | - | Latest applied block height | - |
| `hl_visor_initial_height` | Gauge | - | Height at current visor start | - |
| `hl_visor_blocks_applied` | Gauge | - | height − initial_height | - |
| `hl_visor_freeze_abci_height` | Gauge | - | Replay floor | - |
| `hl_visor_blocks_above_freeze` | Gauge | - | visor_height − freeze_height | - |
| `hl_visor_consensus_ahead_of_wall_seconds` | Gauge | - | consensus_time − wall_clock_time | - |
| `hl_visor_reference_lag_seconds` | Gauge | - | Reference-node lag when populated | - |
| `hl_visor_reference_lag_populated` | Gauge | - | 1 if the visor sample carried a `reference_lag` field, else 0 | - |
| `hl_visor_scheduled_freeze_height` | Gauge | - | Scheduled freeze height, 0 if none | - |
| `hl_visor_last_observation_age_seconds` | Gauge | - | Age of latest visor sample | - |
| `hl_node_snapshot_last_height` | Gauge | - | Last successful snapshot height | - |
| `hl_node_snapshot_last_age_seconds` | Gauge | - | Seconds since last snapshot | - |
| `hl_node_snapshot_known_count` | Gauge | - | Snapshot sentinels retained | - |
| `hl_evm_db_checkpoint_height` | Gauge | `tier` | EVM DB checkpoint per tier (fast/slow) | - |
| `hl_evm_db_checkpoint_lag_blocks` | Gauge | - | fast − slow checkpoint | - |
| `hl_node_observed_runs` | Gauge | - | hl-node runs observed via replica_cmds dirs. Point-in-time count of currently retained run dirs that DROPS when those dirs are pruned, so do not `rate()`/`increase()`. | - |
| `hl_node_observed_run_start_seconds` | Gauge | - | Newest run-dir mtime | - |

## Node Process

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_node_process_up` | Gauge | `process` | 1 if alive in /proc, 0 otherwise | Linux only |
| `hl_node_process_start_time_seconds` | Gauge | `process` | Process start | Linux only |
| `hl_node_process_cpu_seconds_total` | Gauge | `process` | Cumulative CPU time | Linux only |
| `hl_node_process_rss_bytes` | Gauge | `process` | VmRSS | Linux only |
| `hl_node_process_virt_bytes` | Gauge | `process` | VmSize | Linux only |
| `hl_node_process_threads` | Gauge | `process` | OS thread count | Linux only |
| `hl_node_process_open_fds` | Gauge | `process` | Open FD count | Linux only |

## Subsystem Latency

These expose node-precomputed per-node quantiles as gauges with no `_bucket` series - valid only per-instance; do NOT `avg()`/`quantile()` them across nodes or steps.

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_node_subsystem_latency_mean_seconds` | Gauge | `subsystem` | Windowed mean | - |
| `hl_node_subsystem_latency_median_seconds` | Gauge | `subsystem` | Windowed median | - |
| `hl_node_subsystem_latency_p90_seconds` | Gauge | `subsystem` | p90 | - |
| `hl_node_subsystem_latency_p95_seconds` | Gauge | `subsystem` | p95 | - |
| `hl_node_subsystem_latency_max_seconds` | Gauge | `subsystem` | Windowed max | - |
| `hl_node_subsystem_latency_stddev_seconds` | Gauge | `subsystem` | Std dev | - |
| `hl_node_subsystem_work_fraction` | Gauge | `subsystem` | Fraction of wall-clock doing work | - |
| `hl_node_subsystem_samples_total` | Gauge | `subsystem` | Cumulative samples | - |

## Critical Messages

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_node_bugs` | Gauge | `source` | `bug!` events since source start | - |
| `hl_node_crits` | Gauge | `source` | `crit!` events since source start | - |
| `hl_node_crit_locations` | Gauge | `source` | Distinct (file, line) sites that fired | - |
| `hl_node_critical_messages_base_time_seconds` | Gauge | `source` | Cycle start (source process start) | - |

## P2P / Peer Traffic

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_p2p_peer_traffic` | Gauge | `ip`, `direction` | Per-peer raw hl-node tcp_traffic value (comparative only, not bytes; ~0..1 observed), top-16 + `ip="other"` | - |
| `hl_p2p_total_traffic` | Gauge | `direction` | Raw hl-node tcp_traffic value (comparative only, not bytes; ~0..1 observed), summed across peers | - |
| `hl_p2p_peer_count` | Gauge | `direction` | Distinct peers in sample | - |
| `hl_p2p_sample_age_seconds` | Gauge | - | Age of latest tcp_traffic sample | - |
| `hl_node_parent_peer_info` | Gauge | `ip` | 1 for current parent peer | - |
| `hl_node_parent_peer_traffic` | Gauge | - | Inbound raw hl-node tcp_traffic value from parent (comparative only, not bytes; ~0..1 observed) | - |
| `hl_node_parent_peer_tenure_seconds` | Gauge | - | Time current parent has held role | - |
| `hl_node_parent_peer_switches_total` | Counter | - | Cumulative parent changes | - |
| `hl_p2p_gossip_events_total` | Counter | `event_type` | Gossip events by tag | - |
| `hl_p2p_gossip_peer_events_total` | Counter | `ip`, `event_type` | Gossip events attributed to a structured remote IP | `--per-peer-metrics` |
| `hl_p2p_gossip_events_unattributed_total` | Counter | `event_type` | Gossip events whose structured payload did not contain a valid remote IP | - |
| `hl_p2p_tcp_connections` | Gauge | `port`, `state` | TCP state counts per listening port | Linux only |
| `hl_p2p_tcp_bytes_total` | Counter | `ip`, `direction` | Observed TCP payload bytes since collection baseline, summed across P2P sockets by remote IP | Linux, `--per-peer-metrics` |
| `hl_p2p_tcp_collection_success` | Gauge | - | Whether the latest complete TCP byte collection succeeded (1/0) | Linux, `--per-peer-metrics` |
| `hl_p2p_tcp_last_success_seconds` | Gauge | - | Unix timestamp of the latest successful complete TCP byte collection | Linux, `--per-peer-metrics` |
| `hl_p2p_tcp_observed_sockets` | Gauge | - | P2P sockets with byte counters in the latest successful dump; zero can mean no active peers or the wrong network namespace | Linux, `--per-peer-metrics` |

`hl_p2p_gossip_peer_events_total` accepts only the bounded `event_type` allowlist documented by the exporter; unknown tags use `event_type="other"`. Active IPs are limited to 2048 with a 24-hour TTL. Evicting an IP deletes its active series, so a later reappearance is a normal counter reset that `rate()` and `increase()` handle. The parser accepts structured `Ip`, `ip`, `remote_ip`, and similar `*_ip` fields, but deliberately does not extract addresses from free-form error strings. Use Loki for exact event payloads and sub-scrape timing.

### P2P TCP payload by peer IP

`hl_p2p_tcp_bytes_total` is a **true Prometheus counter** derived from per-socket
Linux `TCP_INFO` byte totals. Use `rate()` for bytes/s and `increase()` for observed
range volumes. This is independent of the unitless `tcp_traffic` source and the
LZ4 source gauges. Example Grafana queries:

```promql
# Outbound payload in bits/s, per node and remote IP.
8 * sum by (env, instance, ip) (
  rate(hl_p2p_tcp_bytes_total{direction="out"}[$__rate_interval])
)

# Inbound payload volume over the selected dashboard range, in bytes.
sum by (env, instance, ip) (
  increase(hl_p2p_tcp_bytes_total{direction="in"}[$__range])
)
```

The standalone `grafana/hyperliquid-p2p-traffic-dashboard.json` includes throughput
curves, range volumes, a joined per-IP table and collection health. Throughput
queries use the byte counters directly; they do not impose a fixed freshness
cutoff that depends on the scrape interval. Check collection health for failed
polls; an unsuccessful poll leaves counters at their previous values until the
next successful dump. Configure Prometheus to scrape at approximately 15 seconds;
allow at least two scrapes before expecting a rate.

Collection success only confirms a complete kernel dump, even when it contains
zero P2P sockets. Check `hl_p2p_tcp_observed_sockets` to distinguish a working
socket dump from actual peer visibility. The dashboard shows **No P2P sockets**
for successful but empty dumps, **Collecting** when sockets are visible,
and **Incomplete** when the latest collection failed. Inspect
`hl_p2p_tcp_last_success_seconds` separately to diagnose a stalled monitor. With multiple nodes selected,
the worst state is displayed. Zero sockets does not by itself prove a namespace
mismatch: the node may have no established P2P connections.

**Coverage:** the exporter polls IPv4 and IPv6 sockets in its own network namespace
(or the one selected with `--tcp-netns PATH`) every 15 seconds, selecting connections whose local **or** remote port is 4001 or
4002. This includes accepted client connections and connections our node initiates.
IPv4-mapped IPv6 addresses are normalized to IPv4. Multiple sockets and both P2P
ports are summed per IP. The kernel's socket cookie prevents reconnections from
resetting a peer's cumulative exporter counter. `direction="out"` uses
`tcpi_bytes_sent` (including retransmitted payload); `direction="in"` uses
`tcpi_bytes_received`. TCP/IP/Ethernet headers and pure ACKs are excluded. These
are TCP payload bytes, including application framing and any compressed content,
not uncompressed application data or full interface bandwidth.

The first complete dump establishes the baseline without counting socket traffic
from before collection startup. New sockets contribute their initially observed
bytes, and subsequent dumps add only positive deltas. The monitor retains up to
8192 current sockets and 2048 peer IPs. Disconnected IP counters expire after 24
hours without a socket observation, or are evicted oldest-first to admit an active
IP. There is no moving top-N or `ip="other"` bucket. Dumps that exceed an active
limit, lack required byte fields, or are interrupted/truncated are discarded;
collection health becomes 0, and the previous baseline is retained for recovery.
An exporter restart or per-IP eviction is a normal Prometheus counter reset.

**Limits:** sockets that open and close between polls, the final bytes after a
socket's last observation, and traffic during exporter downtime can be missed.
Prometheus cannot infer the initial value of a newly scraped counter series, so
the first observed bytes for a new IP can also be absent from range increases.
Prometheus range functions also extrapolate from scrape samples. This measures
observed client consumption for operations, not exact billing. Downstream clients
cannot be inferred from TCP byte direction alone: upstream/validator peers are
also measured; select known client IPs. Shared IPs/NAT aggregate their clients.

The collector needs access to the **network namespace of hl-node**. A Docker bind
mount of node data is insufficient. An exporter on the host can use
`--per-peer-metrics --tcp-netns /proc/<container-pid>/ns/net` to collect a
bridge-mode container while its HTTP endpoint stays on the host. This requires
`CAP_SYS_ADMIN` and access to that namespace file (typically also `CAP_SYS_PTRACE`
for a non-root service user). Update the path and restart the exporter if the
container PID changes. There is no Docker daemon integration. Only diagnostic
socket creation enters the target namespace, on a dedicated locked thread that
is discarded on exit. Both address-family dumps use the same socket, ensuring a
single namespace per poll. Invalid paths or namespace-entry failures cause a
collection error; they never fall back to collecting host sockets. The separate
TCP connection-state monitor continues to describe the exporter's own namespace.
The kernel must support `NETLINK_INET_DIAG` and sufficiently recent `TCP_INFO`
fields (Linux 4.19+); sandbox policies must allow the netlink socket. No external
`ss` binary or packet capture is used. The monitor idles on non-Linux systems.

Examples:

```promql
# Per-IP gossip-RPC activity over ten minutes.
sum by (instance, ip) (
  increase(hl_p2p_gossip_peer_events_total{event_type="verified_gossip_rpc"}[10m])
)

# Repeated stream setup/check failures by IP.
sum by (instance, ip) (
  increase(hl_p2p_gossip_peer_events_total{
    event_type=~"handle_stream_connection|performing_checks_on_stream|got_tcp_greeting|error_checking_connection"
  }[5m])
)

# Same remote IP active against multiple scraped nodes.
count by (ip) (
  sum by (instance, ip) (
    increase(hl_p2p_gossip_peer_events_total[10m])
  ) > 0
)
```

## Peer Set

**Which peer-count metric?** Several metrics count peers using different denominators:
- `hl_p2p_peers` - unique peer IPs with positive traffic in the latest tcp_traffic sample.
- `hl_p2p_peer_count{direction}` - peers in the latest sample per direction; INCLUDES idle peers at value=0, so `>=` `hl_p2p_peers`.
- `hl_p2p_unique_peers_seen{window}` - rolling-window unique peer IPs (5m/1h/24h).
- `hl_p2p_non_val_peers_total` - gossip non-validator peers.

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_p2p_unique_peers_seen` | Gauge | `window` | Distinct peer IPs seen in window (5m/1h/24h) | - |
| `hl_p2p_peers` | Gauge | - | Unique peer IPs with positive traffic in the latest tcp_traffic sample ("right now"). Point-in-time count (peers in the latest sample) that can DECREASE, so do not `rate()`/`increase()`. | - |
| `hl_p2p_peers_added_total` | Counter | - | Cumulative new peers since exporter start. In-process monotonic (resets only on exporter restart), so `rate()`/`increase()` are valid. | - |
| `hl_p2p_peers_evicted_total` | Counter | - | Cumulative evictions since exporter start. In-process monotonic (resets only on exporter restart), so `rate()`/`increase()` are valid. | - |
| `hl_p2p_peer_last_seen_seconds` | Gauge | `ip` | Last-seen Unix timestamp per known peer IP, from traffic or structured gossip events | `--per-peer-metrics` |
| `hl_p2p_peer_first_seen_seconds` | Gauge | `ip` | First-seen Unix timestamp per known peer IP | `--per-peer-metrics` |

## Disk

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_node_disk_used_bytes` | Gauge | - | NODE_HOME tree size | - |
| `hl_node_disk_free_bytes` | Gauge | - | Filesystem bytes available | - |
| `hl_node_disk_total_bytes` | Gauge | - | Filesystem total bytes | - |
| `hl_node_disk_subdir_bytes` | Gauge | `subdir` | Tracked NODE_HOME subdir sizes | - |

## Info Probe

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_info_endpoint_up` | Gauge | - | 1 if probe returned 200 + non-empty body | `--probe-info-endpoint` |
| `hl_info_endpoint_latency_seconds` | Histogram | - | Probe latency | `--probe-info-endpoint` |
| `hl_info_endpoint_last_success_seconds` | Gauge | - | Last successful probe | `--probe-info-endpoint` |
| `hl_info_endpoint_failures_total` | Counter | - | Cumulative probe failures | `--probe-info-endpoint` |

## Extended: LZ4

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_p2p_lz4_compression_ratio` | Gauge | `ip`, `direction` | Per-peer ratio (compressed÷uncompressed) | `--extended-metrics` |
| `hl_p2p_lz4_bytes_total` | Gauge | `ip`, `direction` | Per-peer compressed bytes since source start | `--extended-metrics` |
| `hl_p2p_lz4_packets_total` | Gauge | `ip`, `direction` | Per-peer compressed packets since source start | `--extended-metrics` |
| `hl_p2p_lz4_global_ratio` | Gauge | - | Global ratio | `--extended-metrics` |
| `hl_p2p_lz4_global_bytes_total` | Gauge | - | Global compressed bytes since source start | `--extended-metrics` |
| `hl_p2p_lz4_global_packets_total` | Gauge | - | Global compressed packets since source start | `--extended-metrics` |
| `hl_tcp_lz4_latency_seconds` | Gauge | `direction`, `port`, `quantile` | Per-port lz4 latency. `quantile` ∈ {p50, p90, p95, max, mean, std_dev} (the `work_frac` duty-cycle moved to `hl_tcp_lz4_work_fraction`). | `--extended-metrics` |
| `hl_tcp_lz4_work_fraction` | Gauge | `direction`, `port` | Fraction of wall-clock time (0..1 duty cycle) the lz4 codec spent doing work per direction/port. Unitless; split out of `hl_tcp_lz4_latency_seconds`. | `--extended-metrics` |

## Extended: Per-step Latency

These expose node-precomputed per-node quantiles as gauges with no `_bucket` series - valid only per-instance; do NOT `avg()`/`quantile()` them across nodes or steps.

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_latency_bucket_guard_seconds` | Gauge | `step`, `quantile` | bucket_guard sub-step latency. `quantile` ∈ {p50, p90, p95, max, mean, std_dev} (the `work_frac` duty-cycle moved to `hl_latency_bucket_guard_work_fraction`). | `--extended-metrics` |
| `hl_latency_bucket_guard_work_fraction` | Gauge | `step` | Fraction of wall-clock time (0..1 duty cycle) each bucket_guard sub-step spent doing work. Unitless; split out of `hl_latency_bucket_guard_seconds`. | `--extended-metrics` |

## Extended: Operator Health

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_node_log_lines` | Gauge | `stream`, `level` | Lines in day's log file | `--extended-metrics` |
| `hl_node_log_bytes` | Gauge | `stream`, `level` | Day's log file size | `--extended-metrics` |
| `hl_node_public_ip_info` | Gauge | `ip` | 1 for current public IP | `--extended-metrics` |
| `hl_node_public_ip_age_seconds` | Gauge | - | IP heartbeat age | `--extended-metrics` |
| `hl_node_public_ip_changes_total` | Counter | - | Cumulative IP changes | `--extended-metrics` |
| `hl_node_operator_config_age_seconds` | Gauge | `file` | file_mod_time_tracker mtime age | `--extended-metrics` |
| `hl_node_tmp_bytes` | Gauge | - | `$NODE_HOME/tmp` bytes | `--extended-metrics` |
| `hl_node_tmp_stale_files` | Gauge | - | tmp files older than 24h | `--extended-metrics` |
| `hl_node_shell_exec_pending` | Gauge | - | shell_rs_out file count | `--extended-metrics` |

## Extended: Tokio Runtime

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_tokio_task_poll_seconds_total` | Gauge | `task` | Cumulative poll time | `--extended-metrics` |
| `hl_tokio_task_polls_total` | Gauge | `task` | Cumulative polls | `--extended-metrics` |
| `hl_tokio_task_slow_polls_total` | Gauge | `task` | Cumulative slow polls | `--extended-metrics` |
| `hl_tokio_task_long_delays_total` | Gauge | `task` | Cumulative long delays | `--extended-metrics` |
| `hl_tokio_task_idle_seconds_total` | Gauge | `task` | Cumulative idle time | `--extended-metrics` |
| `hl_tokio_task_dropped_total` | Gauge | `task` | Cumulative dropped tasks | `--extended-metrics` |

## Extended: Crit Locations

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_node_crit_location` | Gauge | `file`, `line` | Per-site crit count, capped at 32 | `--extended-metrics` |
| `hl_node_crit_location_last_seen_seconds` | Gauge | `file`, `line` | Last activation timestamp | `--extended-metrics` |

## Extended: RocksDB

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_rocksdb_sst_files` | Gauge | `db` | SST file count per RocksDB | `--extended-metrics` |
| `hl_rocksdb_write_stalls_total` | Gauge | `db`, `reason` | Cumulative write stalls by reason | `--extended-metrics` |
| `hl_rocksdb_block_cache_usage_bytes` | Gauge | `db` | Block cache usage | `--extended-metrics` |

## Validator (validator-only, always-on)

These auto-detect via filesystem: each monitor checks for its source directory at startup and idles silently on non-validator nodes (no flag needed).

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_consensus_committed_blocks` | Gauge | - | Cumulative blocks committed since hl-node start. Resets to 0 on restart. | Validator node |
| `hl_consensus_committed_txs` | Gauge | - | Cumulative transactions committed since hl-node start. | Validator node |
| `hl_consensus_committed_tx_bytes` | Gauge | - | Cumulative bytes of committed transactions since hl-node start. | Validator node |
| `hl_consensus_dropped_txs` | Gauge | - | Cumulative dropped txs (mempool/consensus shedding load). Non-zero rate is operator-actionable. | Validator node |
| `hl_consensus_round_qc` | Gauge | - | QC-decided rounds since hl-node start. Tracks committed_blocks on a healthy network. | Validator node |
| `hl_consensus_round_tc` | Gauge | - | Timeout-certificate rounds (view changes) since hl-node start. | Validator node |
| `hl_consensus_round_catchup` | Gauge | - | Catch-up events (validator was behind and recovered) since hl-node start. | Validator node |
| `hl_consensus_rpc_requests_registered` | Gauge | - | Validator-RPC requests this validator served. | Validator node |
| `hl_consensus_rpc_requests_sent` | Gauge | - | Validator-RPC requests this validator initiated. | Validator node |
| `hl_mempool_events_total` | Counter | `event_type`, `status` | Mempool log events by type (add_tx, verify_block, committed, dropping_blocks, size_stats, …). `status` carries the outcome for add_tx and verify_block; empty for others. | Validator node |
| `hl_mempool_size` | Gauge | `component` | Mempool depth from "Size stats" events. components: committed_tx_hashes, uncommitted_txs, blocks, rpc_requests. | Validator node |
| `hl_node_replay_events` | Gauge | - | Current count of retained replay-from-checkpoint dirs (decreases on pruning) under data/node_logs/replay/. Point-in-time count, so do not `rate()`/`increase()`. | Validator node |
| `hl_node_replay_last_seconds` | Gauge | - | Unix timestamp of the most recent replay event. | Validator node |
| `hl_node_replay_last_height` | Gauge | - | Block height of the most recent replay event. | Validator node |
| `hl_node_operator_config_failed_loads` | Gauge | - | Count of `*_FAILED_LOAD` sidecars in file_mod_time_tracker/. Non-zero = silent misconfiguration (operator pushed a config hl-node rejected). | `--extended-metrics` |

## Split-Client Mempool Transactions

These auto-detect `$NODE_HOME/data/mempool_txs/hourly/`, which appears when `split_client_blocks: true` is active and every upstream hop is forwarding split client blocks. The raw files contain uncommitted order-flow details; the exporter emits only aggregate counters and histograms with bounded labels.

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_mempool_txs_seen_total` | Counter | - | Uncommitted mempool transaction records observed in `data/mempool_txs`. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_bytes_total` | Counter | - | Bytes read from complete JSONL records in `data/mempool_txs`. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_signed_actions_total` | Counter | `type` | Signed actions observed in split-client mempool transaction records, by allowlisted action type. New action types are bucketed as `other`. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_operations_total` | Counter | `type` | Individual operations observed in split-client mempool transactions. `orders`, `cancels`, and `modifies` count array elements; scalar actions count as 1. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_order_operations_total` | Counter | `side`, `tif` | Order-like operations by side and time-in-force. Includes `order`, `modify`, and `batchModify` entries. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_signed_actions_per_record` | Histogram | - | Distribution of `signed_actions` array length per split-client mempool transaction record. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_operations_per_record` | Histogram | - | Distribution of individual operation count per split-client mempool transaction record. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_latest_time` | Gauge | - | Unix timestamp from the latest split-client mempool transaction record processed. | Node with `data/mempool_txs/hourly/` |
| `hl_mempool_txs_sample_age_seconds` | Gauge | - | Wall-clock age of the latest split-client mempool transaction record processed. | Node with `data/mempool_txs/hourly/` |

## Validator (validator-only, --extended-metrics)

Adds per-step latency breakdowns from the two validator-only sub-step subsystems.

These expose node-precomputed per-node quantiles as gauges with no `_bucket` series - valid only per-instance; do NOT `avg()`/`quantile()` them across nodes or steps.

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_latency_consensus_seconds` | Gauge | `step`, `quantile` | Consensus state-machine latency per input type. step ∈ {HandleStateInput::Block, HandleStateInput::Vote, …, TxCommit, WallClockBlockGap}. quantile ∈ {p50, p90, p95, max, mean, std_dev} (the `work_frac` duty-cycle moved to `hl_latency_consensus_work_fraction`). | Validator node + `--extended-metrics` |
| `hl_latency_consensus_work_fraction` | Gauge | `step` | Fraction of wall-clock time (0..1 duty cycle) each consensus state-machine step spent doing work. Unitless; split out of `hl_latency_consensus_seconds`. | Validator node + `--extended-metrics` |
| `hl_latency_l1_task_seconds` | Gauge | `step`, `quantile` | L1 block-apply phase latency. step ∈ {BeginBlock, DeliverSignedActions, EndBlock, RecoverUsers}. quantile ∈ {p50, p90, p95, max, mean, std_dev} (the `work_frac` duty-cycle moved to `hl_latency_l1_task_work_fraction`). | Validator node + `--extended-metrics` |
| `hl_latency_l1_task_work_fraction` | Gauge | `step` | Fraction of wall-clock time (0..1 duty cycle) each L1 block-apply phase spent doing work. Unitless; split out of `hl_latency_l1_task_seconds`. | Validator node + `--extended-metrics` |

## Go Runtime

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_go_heap_inuse_bytes` | Gauge | - | Heap in use (raw bytes) | - |
| `hl_go_heap_idle_bytes` | Gauge | - | Heap idle (raw bytes) | - |
| `hl_go_heap_objects` | Gauge | - | Live heap objects | - |
| `hl_go_sys_bytes` | Gauge | - | System memory acquired (raw bytes) | - |
| `hl_go_num_goroutines` | Gauge | - | Live goroutines | - |

## Exporter Self-observability

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_exporter_build_info` | Gauge | `version`, `commit`, `go_version` | Build info, always 1 | - |
| `hl_exporter_ready` | Gauge | - | Mirrors `/readyz` | - |
| `hl_exporter_monitor_started_seconds` | Gauge | `monitor` | Monitor goroutine launch time | - |
| `hl_exporter_monitor_last_tick_seconds` | Gauge | `monitor` | Last successful processing cycle | - |
| `hl_exporter_monitor_panics_total` | Counter | `monitor` | Recovered panics. In-process monotonic since exporter start (resets only on exporter restart), so `rate()`/`increase()` are valid. | - |
| `hl_exporter_monitor_errors_total` | Counter | `monitor` | Errors reported. In-process monotonic since exporter start (resets only on exporter restart), so `rate()`/`increase()` are valid. | - |

### Consensus monitor health (OTLP-path)

These four are emitted via the OTLP/observable-instrument path (not the client_golang registry), so they appear on the Prometheus `/metrics` endpoint through the OTel->Prometheus bridge. The names below are exactly as exposed: the three counters already carry `_total` (the bridge does not double-suffix), and `hl_consensus_monitor_last_processed` declares `WithUnit("timestamp")`, which is not in the bridge's unit-suffix table, so no `_seconds`/`_timestamp` suffix is appended.

| Metric | Type | Labels | Description | Requirements |
|--------|------|--------|-------------|--------------|
| `hl_timeout_rounds_total` | Counter | `suspect` | Timeout rounds observed by the round-advance monitor, by suspected stalled validator | Validator node |
| `hl_consensus_monitor_last_processed` | Gauge | `monitor_type` | Unix timestamp of the last line processed, per monitor (`monitor_type` ∈ `consensus`/`status`) | - |
| `hl_consensus_monitor_lines_processed_total` | Counter | `monitor_type` | Lines processed by the consensus/status monitor | - |
| `hl_consensus_monitor_errors_total` | Counter | `monitor_type` | Errors encountered by the consensus/status monitor | - |

## HTTP endpoints

| Path | Status |
|------|--------|
| `/metrics` | Prometheus exposition |
| `/livez` | 200 while running |
| `/readyz` | 200 once every monitor has launched |
| `/health` | Legacy alias for `/livez` |

## Label Definitions

### Common Labels
- `validator`: Validator address
- `signer`: Signer address
- `name`: Human-readable validator name
- `type`: Transaction or operation type
- `category`: Operation category (for operations_total)
- `block_type`: EVM block type (standard / small / high / other); attached automatically whenever EVM metrics are enabled (no separate flag)

### v3 Labels
- `source`, `process`: `hl-node` or `hl-visor`
- `subsystem`: hl-node subsystem name
- `step`: sub-step within bucket_guard
- `quantile`: `p50` / `p90` / `p95` / `max` / `mean` / `std_dev` (the unitless `work_frac` duty-cycle is no longer a quantile value - it lives in the dedicated `*_work_fraction` gauges)
- `direction`: `in` / `out`
- `ip`: peer IP (or literal `other`)
- `port`, `state`: TCP port + state (`ESTABLISHED`, `TIME_WAIT`, …)
- `event_type`: gossip event tag
- `tier`: `fast` / `slow`
- `db`: RocksDB name
- `reason`: write-stall reason
- `task`: Tokio task name
- `file`, `line`: source location
- `monitor`: monitor name
- `monitor_type`: consensus-monitor source stream (`consensus` / `status`)
- `suspect`: suspected stalled validator (on `hl_timeout_rounds_total`)
- `subdir`: tracked NODE_HOME subdir
- `stream`, `level`: `infra`/`trade`/`visor`, `error`/`warn`
- `event_type`, `status`: mempool event tag + per-event outcome (add_tx/verify_block carry status)
- `component`: mempool capacity bucket - committed_tx_hashes / uncommitted_txs / blocks / rpc_requests
- `side`: split-client mempool order side (`buy`, `sell`, `unknown`)
- `tif`: split-client mempool order time-in-force (`Alo`, `Ioc`, `Gtc`, `FrontendMarket`, `unknown`, `other`)

## Metric Requirements

### Replica Metrics (`--replica-metrics`)
Require the flag AND hl-node running with `--replica-cmds-style actions-and-responses`:
- All `hl_core_tx_*`, `hl_core_operations_*`, `hl_core_orders_total`, `hl_core_orders_per_block`
- `hl_core_*_processed_*`
- `hl_metal_parse_duration`, `hl_metal_last_processed_*`

### Validator Node Metrics
Validator node with consensus/status logs:
- All `hl_consensus_*` except basic info from api.hyperliquid.xyz/info

### `--probe-info-endpoint`
- All `hl_info_endpoint_*`

### `--extended-metrics`
Bundle: LZ4, per-step latency (bucket_guard + tcp_lz4, plus validator-only consensus + l1_task_latency), operator health, Tokio runtime, crit locations, RocksDB.

### Linux-only
- All `hl_node_process_*`
- `hl_p2p_tcp_connections`
- `hl_p2p_tcp_bytes_total`, `hl_p2p_tcp_collection_success`, `hl_p2p_tcp_last_success_seconds` (`--per-peer-metrics`)

### Flag-gated metric default values
Metrics gated by `--probe-info-endpoint` or `--extended-metrics` still appear at zero when the flag is off (standard `promauto` behavior). See [UPGRADING.md](../UPGRADING.md).
