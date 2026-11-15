# Hyperliquid Exporter

A Go-based exporter that collects and exposes metrics for Hyperliquid node operators. This exporter produces metrics for HyperCore, HyperEVM, and HyperBFT, covering block production, transaction flow, validator performance, stake distribution, EVM activity, split-client mempool transaction flow, and consensus events. For a full list of metrics, see [docs/metrics.md](docs/metrics.md).

## Quick Start

### Installation

```bash
git clone https://github.com/validaoxyz/hyperliquid-exporter.git $HOME/hyperliquid-exporter
cd $HOME/hyperliquid-exporter
make build
```

### Basic Usage

```bash
./bin/hl_exporter start --chain mainnet [OPTIONS]

OPTIONS:
  --replica-metrics          # Transaction metrics (requires node --replica-cmds-style)
  --evm-metrics             # EVM chain metrics
  --contract-metrics        # Per-contract transaction tracking
  --metrics-port N          # Prometheus port (default 8086)
  --probe-info-endpoint     # Active HTTP probe of --serve-info
  --extended-metrics        # Extra monitors (lz4, logs, RocksDB, etc.)
  --per-peer-metrics        # Per-IP timestamps, gossip counters and Linux TCP payload bytes
  --tcp-netns PATH          # Optional target network namespace for TCP payload collection
  --skip-version-check      # For containerized deployments
  --skip-update-check       # For containerized deployments
  --otlp                    # Enable OTLP export (requires --alias and --otlp-endpoint)
  --alias "validator-name"  # Node alias for OTLP
  --otlp-endpoint "url"     # OTLP endpoint URL
```

Example: `./bin/hl_exporter start --chain mainnet --replica-metrics --evm-metrics`.

By default, the exporter:
- Exposes Prometheus metrics on `:8086/metrics`, liveness on `/livez`, readiness on `/readyz`
- Looks for log files in `$HOME/hl` and binaries in `$HOME/`
- Auto-detects optional data sources such as `data/mempool_txs/hourly/` when `split_client_blocks` is enabled
- Uses `info` log level
- Disables OTLP export

### P2P client traffic

Run the exporter with `--per-peer-metrics` on Linux and import
[`grafana/hyperliquid-p2p-traffic-dashboard.json`](grafana/hyperliquid-p2p-traffic-dashboard.json)
into Grafana. Select the Prometheus datasource, environment, node and peer IPs.
The dashboard shows inbound/outbound TCP payload throughput, observed volumes
over the selected time range, and collection health. It uses real byte counters
from Linux `TCP_INFO`, independently of the unitless `hl_p2p_peer_traffic` metric.

The TCP collector must access the node's **network namespace**. For a host-based
node, run the exporter on the host, or use Docker `network_mode: host`. For an
exporter on the host and a node in a bridge container, use `--tcp-netns` as below.
A read-only mount of the node's files alone does not expose its sockets. Linux
must provide `NETLINK_INET_DIAG` and `TCP_INFO` byte fields (Linux 4.19+).
Collection in the exporter's own namespace normally needs no root access;
entering another namespace requires additional privileges.

Collection samples connections with local or remote port 4001/4002 every 15
seconds and aggregates all sockets by remote IP. `in` and `out` mean bytes
received/sent by our node, regardless of who initiated the connection. These
metrics do not automatically distinguish downstream clients from upstream or
validator peers; select the relevant client IPs to measure their consumption.
Headers and pure ACKs are excluded; outbound payload retransmissions are included.
Existing socket totals are baselined at startup. Connections that open and close
between polls, and final bytes after the last sample, can be missed. Volumes are
operational estimates from observed counters, not billing-grade accounting.
See [metric coverage and queries](docs/metrics.md#p2p-tcp-payload-by-peer-ip).

#### When hl-node runs in Docker

First check its network mode (replace `hl-node` with the actual container name):

```bash
docker inspect --format '{{.HostConfig.NetworkMode}}' hl-node
```

**Exporter running on the host:** for a bridge-mode node, obtain its current PID:

```bash
docker inspect --format '{{.State.Pid}}' hl-node
```

For a PID of `1605`, append these flags to the existing exporter command
(`ExecStart` for systemd), preserving all its other options:

```text
--per-peer-metrics --tcp-netns /proc/1605/ns/net
```

Only the diagnostic socket is opened in that network namespace. The exporter
HTTP endpoint stays on the host at its existing address/port; the node can stay
running. The exporter does not invoke Docker or need access to its daemon.
If the container restarts and its PID changes, update the path and restart the
exporter. A missing or inaccessible path fails collection explicitly, without
falling back to the host's sockets.

The service needs `CAP_SYS_ADMIN` for `setns`, plus permission to open the target
namespace. For a non-root systemd user reading a root-owned container's namespace,
`CAP_SYS_PTRACE` is generally also required. Keep the service user and add to its
`[Service]` configuration:

```ini
AmbientCapabilities=CAP_SYS_ADMIN CAP_SYS_PTRACE
CapabilityBoundingSet=CAP_SYS_ADMIN CAP_SYS_PTRACE
```

Reload systemd and restart **only the exporter** after installing the updated
binary and editing its command. Service sandbox settings must permit `setns`,
`AF_NETLINK`, and access to `/proc/<pid>/ns/net`. These are substantial privileges;
an exporter sharing the node's network namespace is an alternative if they are
not appropriate for the service. `--tcp-netns` targets TCP **payload bytes only**;
the existing `/proc` TCP connection-state monitor still describes the exporter's
own namespace.

**Exporter running in Docker:**

If it reports `host`, an exporter running on the host or with `network_mode: host`
sees the same sockets. If hl-node has its own namespace (for example a bridge
network), putting the exporter on the same bridge is not enough. An exporter
container can share the running node's namespace using:

```yaml
services:
  hl_exporter:
    # Keep the existing image/build, commands and volumes.
    network_mode: "container:hl-node"
    # Remove ports: and networks: from the exporter service.
```

Use `network_mode: "service:hl-node"` instead when `hl-node` is the node's service
name in the same Compose project. Keep `--per-peer-metrics` in the exporter's
command. Recreate only the exporter, for example with
`docker compose up -d --no-deps --force-recreate hl_exporter`; the node can stay
running. If the node container is recreated later, recreate the exporter too so
it joins the new namespace.

**Prometheus access changes with the namespace:** the exporter listens on 8086
inside the node's namespace. Prometheus must reach the node container's address
on that port (for example its service name on a shared user-defined network).
The exporter's old published host port is no longer provided. If Prometheus is
remote and needs a published port, plan that endpoint before changing the
exporter; adding a new port mapping to the node would require recreating it.

For diagnosis, run the following PromQL queries per instance:

```promql
hl_p2p_tcp_observed_sockets
count by (instance) (hl_p2p_tcp_bytes_total)
```

Collection can succeed with zero visible sockets. The dashboard reports
**No P2P sockets** in this case; check the network namespace and the node's live
4001/4002 connections before waiting for throughput graphs.


## `vals` subcommand

`hl_exporter vals` extracts the full Hyperliquid validator set (IP, moniker, address) straight from the node's ABCI state files (`data/periodic_abci_states`), reusing the exporter's internal ABCI reader rather than scraping Prometheus - the metrics only surface the RTT-probed subset, not the complete validator IP list. It emits the `ip,moniker,address,vp` CSV the validao.xyz node-map pipeline consumes (`vp` is left empty, since the map renderer only uses `ip`).

Three modes:

```bash
# one-shot: write the CSV to stdout (or --out FILE)
hl_exporter vals --node-home /home/ubuntu/hl

# serve: HTTP server exposing the CSV at /vals/<chain> and a header-less
# IP-per-line list at /nodes/<chain>.txt on :8087, regenerated hourly
hl_exporter vals --serve --addr 0.0.0.0:8087 --node-home /home/ubuntu/hl

# backfill: one validator-count row per day from historical states (JSONL)
hl_exporter vals --backfill --since 2025-05-31 --out f.jsonl
```

In `--serve` mode, `/vals/<chain>` is the validator set from ABCI state, while `/nodes/<chain>.txt` is the "all nodes" view: this box's own 24h peer set, fetched from a local peer-counter snapshot endpoint (`peer_ips_24h`), configurable via `--peer-counter-url` (default `http://127.0.0.1:19046/snapshot`). The `/nodes` route depends on that peer-counter running on the box; if it's unreachable the cache keeps its last good value.

## Run with Systemd
To run the exporter as a systemd service:

Create a systemd service file:
```
echo "[Unit]
Description=HyperLiquid Prometheus Exporter
After=network.target

[Service]
# The working directory where the script is located
WorkingDirectory=$HOME/hyperliquid-exporter

# Command to execute the script
ExecStart=/usr/local/bin/hl_exporter start --chain $CHAIN [options]

# Restart the service if it crashes
Restart=always
RestartSec=10

# Run the service as the current user
User=$USER
Group=$USER

[Install]
WantedBy=multi-user.target" | sudo tee /etc/systemd/system/hyperliquid-exporter.service
```

## Run with Docker

Use Docker to run `hl_exporter` in a container:

1. Edit the `docker-compose.yml` to change the local hyperliquid home folder :
Example : Replace
`- <HYPERLIQUID LOCAL HOME>:/hl:ro`
by
`- /home/hyperliquid/hl:/hl:ro` 

2. Build the image and run the container :
```bash
docker compose up -d
```


## Documentation

- [Metrics Reference](docs/metrics.md) - All metrics with descriptions and labels
- [CHANGELOG](CHANGELOG.md)
- [UPGRADING](UPGRADING.md) - v2 -> v3 notes
