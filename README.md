# passenger-datadog-monitor

Polls `passenger-status --show=xml` every 10 seconds and emits Passenger health metrics to Datadog via StatsD.

Forked from [Sjeanpierre/passenger-datadog-monitor](https://github.com/Sjeanpierre/passenger-datadog-monitor).

## Metrics

**Aggregated** (`passenger.*`): processed requests, memory usage, process uptime (min/max/avg/total), request queue depth, pool usage.

**Per-process** (`passenger.process.*`, tagged by `pid`): memory, thread count, idle time, requests processed.

## Usage in Dockerfile

```dockerfile
COPY --from=<registry>/passenger-datadog-monitor:v<version> \
  /usr/local/bin/passenger-datadog-monitor /usr/local/bin/passenger-datadog-monitor
```

The binary runs as a daemon and sends metrics to the local StatsD agent:

```sh
passenger-datadog-monitor -host=$STATSD_HOST -port=$STATSD_PORT
```

## Flags

| Flag | Default | Description |
|:-----|:--------|:------------|
| `-host` | `127.0.0.1` | StatsD host |
| `-port` | `8125` | StatsD UDP port |
| `-print` | `false` | Print metrics to stdout for debugging |

## Development

```sh
make build    # compile binary to bin/
make test     # run tests with race detector
make lint     # run golangci-lint
make docker   # build Docker image locally
```
