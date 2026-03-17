# CLAUDE.md

## Build & Development Commands

```sh
make build    # CGO_ENABLED=0 cross-compile to bin/passenger-datadog-monitor (linux)
make test     # go test -v -race ./...
make lint     # golangci-lint run ./... (config: .golangci.yml)
make docker   # docker build locally
make tidy     # go mod tidy
make clean    # remove bin/ and clear test cache
```

Run a single test: `go test -v -race -run TestFunctionName ./...`

## Architecture

A single-package (`package main`) Go daemon that polls `passenger-status --show=xml` every 10 seconds and emits metrics to Datadog via StatsD.

**Data flow:** `main.go` loop → `retrievePassengerStats()` shells out to `passenger-status` → `parsePassengerXML()` → seven `chart*` functions in `metrics.go` emit to StatsD client.

**Source files:**
- `main.go` — CLI flags, StatsD client init, polling loop
- `passenger.go` — XML structs, shell-out to `passenger-status`, XML parsing, process helpers
- `metrics.go` — All metric computation and StatsD emission; includes `deltaTracker` for converting monotonic counters to per-interval deltas

**Test fixtures:** `sample_data/*.xml` contains example passenger-status XML output used by tests.

## Linting

golangci-lint with: errcheck, govet, ineffassign, staticcheck, gocritic, revive, gosec (G204 excluded since the binary intentionally shells out).

## Release

Tags matching `v*` trigger GoReleaser via GitHub Actions, publishing Docker images to GHCR.
