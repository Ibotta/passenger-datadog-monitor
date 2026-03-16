# passenger-datadog-monitor

Polls `passenger-status --show=xml` every 10 seconds and emits Passenger health metrics to Datadog via StatsD.

Forked from [Sjeanpierre/passenger-datadog-monitor](https://github.com/Sjeanpierre/passenger-datadog-monitor).

## Metrics

**Aggregated** (`passenger.*`): request queue depth, pool usage, process uptime (min/avg/max gauges), total memory (gauge), memory distribution (histogram), total processed requests (count delta).

**Per-process** (`passenger.process.*`, tagged by `pid`): memory (histogram), thread count (gauge), idle time (gauge), requests processed (count delta).

## Usage in Dockerfile

```dockerfile
COPY --from=ghcr.io/ibotta/passenger-datadog-monitor:v<version> \
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
| `-tags` | (none) | Tags for all metrics — comma-separated, space-separated, or mixed (e.g. `source:api,service:my-service` or `source:api service:my-service`) |

### supervisord example

Space-separated tags (`DD_TAGS`) can be passed directly using supervisord's `%(ENV_...)s` interpolation:

```ini
[program:passenger-datadog-monitor]
command=/usr/local/bin/passenger-datadog-monitor -host=%(ENV_STATSD_ENDPOINT_IP)s -port=%(ENV_STATSD_ENDPOINT_PORT)s -tags="version:%(ENV_DD_VERSION)s service:%(ENV_DD_SERVICE)s env:%(ENV_DD_ENV)s %(ENV_DD_TAGS)s"
```

## Development

```sh
make build    # compile binary to bin/
make test     # run tests with race detector
make lint     # run golangci-lint
make docker   # build Docker image locally
```
