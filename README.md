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

The exporter must share the node's **network namespace**. For a host-based node,
run the exporter on the host, or use Docker `network_mode: host` (remove `ports:`
in that mode). A read-only mount of the node's files alone does not expose its
sockets. Linux must provide `NETLINK_INET_DIAG` and `TCP_INFO` byte fields
(Linux 4.19+); no packet-capture capability or root access is normally needed.

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
