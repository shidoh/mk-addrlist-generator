# MikroTik Address List Generator

A service that generates MikroTik address lists from various sources (URLs, files, static addresses) and provides them via HTTP API.

[![CI](https://github.com/shidoh/mk-addrlist-generator/actions/workflows/ci.yml/badge.svg)](https://github.com/shidoh/mk-addrlist-generator/actions/workflows/ci.yml)
[![Release](https://github.com/shidoh/mk-addrlist-generator/actions/workflows/release.yml/badge.svg)](https://github.com/shidoh/mk-addrlist-generator/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shidoh/mk-addrlist-generator)](https://goreportcard.com/report/github.com/shidoh/mk-addrlist-generator)

## Features

- **Multiple source types:**
  - External URLs (HTTP/HTTPS)
  - Local files
  - Static addresses in configuration
- **Output formats:**
  - `mikrotik` - MikroTik RouterOS script format (default)
  - `plain` - Plain text format (one IP/network per line)
  - `json` - JSON format for integrations
  - `nftables` - nftables set format
- **IPv4 and IPv6:** addresses are split by family automatically — the RouterOS
  script emits separate `/ip/` and `/ipv6/` sections, nftables output emits
  separate `<list>_v4` and `<list>_v6` sets
- **Flexible timeout formats:**
  - Days (e.g., `1d`, `7d`)
  - Hours and minutes (e.g., `12h30m`)
  - Minutes and seconds (e.g., `45m30s`)
  - Complex durations (e.g., `2d3h45m30s`)
  - `"0"` for permanent entries (no timeout)
- **Data processing:**
  - Input validation — anything that is not a valid IP or CIDR is excluded from
    every output format
  - Deduplication within source types
  - CIDR aggregation (merge adjacent networks)
- **Observability:**
  - Health check endpoints (`/health`, `/ready`, `/live`)
  - Prometheus metrics (`/metrics`)
  - Structured JSON logging
- **Production ready:**
  - Graceful shutdown
  - Multi-arch Docker images (amd64, arm64)
  - Kubernetes/Helm support

## Installation

### Using Pre-built Binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/shidoh/mk-addrlist-generator/releases).

Release archives are built for Linux and macOS on amd64 and arm64, Linux also on
arm/v7, and Windows on amd64.

### Using Docker

The Docker image is available on GitHub Container Registry:

```bash
docker pull ghcr.io/shidoh/mk-addrlist-generator:latest
```

Images are published for `linux/amd64` and `linux/arm64`.

The image ships `config.example.yaml` as `/app/config.yaml`. That example points
at file sources which do not exist inside the image, so mount your own
configuration over it — see [Running with Docker](#running-with-docker).

### Building from Source

The module is declared as `mk-addrlist-generator` (no GitHub path prefix), so
`go install github.com/shidoh/...` does not work. Clone and build:

```bash
git clone https://github.com/shidoh/mk-addrlist-generator.git
cd mk-addrlist-generator
go build -o mk-addrlist-generator .
```

## Quick Start

There is no `config.yaml` in the repository — create one from the example first:

```bash
cp config.example.yaml config.yaml
$EDITOR config.yaml
./mk-addrlist-generator validate -c config.yaml
./mk-addrlist-generator serve -c config.yaml -l :8080
```

Then point a router at it:

```
/tool fetch url="http://server:8080/list/staticlist" mode=http dst-path=staticlist.rsc
/import staticlist.rsc
```

## CLI Commands

### Serve (HTTP Server)

Start the HTTP server:

```bash
mk-addrlist-generator serve -c config.yaml -l :8080
```

Options:
- `-c, --config` - Path to configuration file (default: `config.yaml`)
- `-l, --listen` - Address to listen on (default: `:8080`)
- `--metrics` - Enable Prometheus metrics (default: `true`)
- `--health` - Enable health check endpoints (default: `true`)
- `--shutdown-timeout` - Graceful shutdown timeout (default: `30s`)
- `-v, --verbose` - Enable verbose logging

### Generate (CLI Generation)

Generate address lists without starting the server:

```bash
# Generate all lists in MikroTik format
mk-addrlist-generator generate -c config.yaml

# Generate a specific list in plain format
mk-addrlist-generator generate -c config.yaml -n mylist -f plain

# Generate with CIDR aggregation and save to file
mk-addrlist-generator generate -c config.yaml -a -o output.txt

# Generate in JSON format
mk-addrlist-generator generate -c config.yaml -f json
```

Options:
- `-c, --config` - Path to configuration file
- `-n, --name` - Generate only this list (default: all lists)
- `-f, --format` - Output format: `mikrotik`, `plain`, `json`, `nftables`
- `-o, --output` - Output file, created with mode `0600` (default: stdout)
- `-a, --aggregate` - Enable CIDR aggregation
- `-d, --deduplicate` - Deduplication, enabled by default. To turn it off pass
  `--deduplicate=false` (or `-d=false`); a bare `-d` sets it to `true`

Generated data goes to stdout, diagnostics go to stderr, so `... -f plain 2>/dev/null`
is safe to pipe.

An unrecognised `--format` value currently falls back to `mikrotik` instead of
failing, so check spelling before feeding the output to a router.

### Validate

Validate the configuration file:

```bash
mk-addrlist-generator validate -c config.yaml
```

Prints the resolved timeout and source counts per list. See
[Configuration validation](#configuration-validation) for what is checked.

### Version

Print version information:

```bash
mk-addrlist-generator version
```

## Configuration

The service is configured using a YAML file:

```yaml
config:
  timeout: 1d # Default timeout for all lists
  commentPrefix: "crowdsecurity" # Default comment prefix for all lists

lists:
  externallists:
    timeout: 3h59m54s # Override default timeout
    commentPrefix: "crowdsecurity/external" # Override default comment prefix
    urls:
      - https://lists.example.com/blocklist1.txt
      - https://lists.example.com/blocklist2.txt

  fileslist:
    timeout: 12h30m
    commentPrefix: "crowdsecurity/local"
    files:
      - /etc/mikrotik/lists/list1.txt
      - /etc/mikrotik/lists/list2.txt

  staticlist:
    timeout: 45m30s
    commentPrefix: "static"
    addresses:
      - 172.16.1.0/24
      - 8.8.8.8
      - 172.27.0.0/21
      - 2001:db8::1
```

`config.example.yaml` in the repository root is the same example.
`config/config.example.yaml` shows a larger setup with an allowlist and a
dedicated IPv6 list.

### Timeout format

Timeouts use a custom format, not Go's `time.ParseDuration`:

```
[<n>d][<n>h][<n>m][<n>s]
```

- Components are optional, but their order is fixed: `1h30m` is valid, `30m1h` is not
- Only these four units exist — no weeks, no milliseconds, no fractions
- Lowercase only, no surrounding whitespace
- `"0"` means a permanent entry: the generated script calls `add` without a
  `timeout=` argument. Quote it, otherwise YAML turns it into a number
- Any other zero-valued spelling (`0s`, `0d0h0m0s`) is rejected
- A timeout is required: it must be set on the list or in `config.timeout`

Values are normalised through Go's duration formatting on output, so `7d`
reaches the router as `168h0m0s` and `12h30m` as `12h30m0s`. This is the same
duration, just spelled differently.

### Comments

The comment written to each address-list entry is the resolved `commentPrefix`
plus a suffix naming the source it came from:

| Source | Comment |
|--------|---------|
| `urls` | `<prefix>/external` |
| `files` | `<prefix>/file` |
| `addresses` | `<prefix>/static` |

With `commentPrefix: "static"` a static address is therefore commented
`static/static`.

### Source file and URL format

Files and URL bodies are read line by line:

- Empty lines are skipped
- Lines starting with `#` are skipped
- Everything after a `#` on a line is stripped as an inline comment
- Lines that are not a valid IP address or CIDR network are dropped, logged as a
  warning, and counted in `invalid_entries` in `/stats`
- Lines longer than 64 KiB abort reading the source

Semicolon comments (`;`) are **not** recognised — such lines are treated as
addresses and dropped as invalid.

### Configuration validation

`validate`, `generate` and `serve` all reject a configuration that:

- defines no lists, or a list with no `urls`, `files` or `addresses`
- has a list whose timeout cannot be resolved — neither the list nor
  `config.timeout` sets one
- has a malformed or out-of-range timeout
- has a list name that does not match `^[A-Za-z0-9_-]{1,64}$`
- has a `commentPrefix` containing `"`, `;`, `$`, `{`, `}`, `\`, CR or LF

The last two rules exist because the list name and the comment are substituted
into the generated RouterOS script; characters that terminate a quoted argument
there would turn list contents into commands.

Static addresses are validated at generation time, not by `validate`.

## HTTP API

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | API information |
| `GET /info` | API information (same payload as `/`) |
| `GET /health`, `/healthz` | Health check (version, uptime, Go version) |
| `GET /ready`, `/readyz` | Readiness check |
| `GET /live`, `/livez` | Liveness check |
| `GET /metrics` | Prometheus metrics |
| `GET /lists/all` | Get all address lists |
| `GET /list/:name` | Get a specific list |
| `GET /lists` | List all available list names |
| `GET /stats` | Per-list counts from the last generation |

### Query Parameters

| Parameter | Values | Description |
|-----------|--------|-------------|
| `format` | `mikrotik`, `plain`, `json`, `nftables` | Output format (default: `mikrotik`) |
| `aggregate` | `true` | Enable CIDR aggregation (default: off) |
| `deduplicate` | `true`, `false` | Deduplication within each source type (default: on) |

`aggregate` and `deduplicate` are compared against the literal string `true`,
so `aggregate=1` and `aggregate=TRUE` read as false. An unknown `format` falls
back to `mikrotik` with status 200.

Errors are returned as JSON (`{"error": "..."}`) regardless of the requested
format, so a router fetching `format=mikrotik` receives JSON when generation
fails. Note the shell: quote URLs that contain `?` under zsh.

### Examples

#### Get All Lists

```bash
curl http://localhost:8080/lists/all
```

Example response (MikroTik format, single list shown):
```
/ip/firewall/address-list/remove [ find where list="externallists" ];
:global externallistsAddIP;
:set externallistsAddIP do={
:if ($3 != "") do={
:do { /ip/firewall/address-list/add list=externallists address=$1 comment="$2" timeout=$3; } on-error={ }
} else={
:do { /ip/firewall/address-list/add list=externallists address=$1 comment="$2"; } on-error={ }
}
}

$externallistsAddIP "192.168.1.1" "crowdsecurity/external/static" "3h59m54s"
$externallistsAddIP "10.0.0.0/24" "crowdsecurity/external/static" "3h59m54s"

:set externallistsAddIP;
```

The script removes the existing address list before repopulating it, and an
IPv6 section with `:global <list>AddIPv6` follows when the list contains IPv6
addresses.

#### Get Specific List

```bash
curl http://localhost:8080/list/staticlist
```

An unknown list returns `404` with `{"error":"list <name> not found"}`.

#### Get in Plain Format

```bash
curl "http://localhost:8080/list/staticlist?format=plain"
```

Example response:
```
172.16.1.0/24
8.8.8.8
172.27.0.0/21
2001:db8::1
```

Addresses are emitted exactly as configured; `/lists/all` separates lists with a
blank line and does not label them, so prefer `/list/<name>` for scripting.

#### Get in JSON Format

```bash
curl "http://localhost:8080/list/staticlist?format=json"
```

Example response:
```json
{
  "list_name": "staticlist",
  "timestamp": "2026-08-29T11:30:54Z",
  "count": 4,
  "entries": [
    {
      "address": "172.16.1.0/24",
      "comment": "static/static",
      "timeout": "45m30s"
    }
  ]
}
```

`/lists/all?format=json` wraps these objects: `{"timestamp": ..., "lists": {"<name>": {...}}}`.

#### Get in nftables Format

```bash
curl "http://localhost:8080/list/staticlist?format=nftables"
```

Example response:
```
# nftables set definition for staticlist
# Generated at 2026-08-29T11:30:54Z
# Total entries: 4

define staticlist_v4 = {
    172.16.1.0/24,
    8.8.8.8,
    172.27.0.0/21,
}

define staticlist_v6 = {
    2001:db8::1,
}

# Example usage:
# nft add set inet filter staticlist_v4 { type ipv4_addr; flags interval; elements = $staticlist_v4 }
# nft add set inet filter staticlist_v6 { type ipv6_addr; flags interval; elements = $staticlist_v6 }
```

A list with no addresses of one family still emits an empty `define` block for
that family.

#### Get with CIDR Aggregation

```bash
curl "http://localhost:8080/list/staticlist?aggregate=true"
```

Aggregation parses every address, so output is normalised and sorted: bare hosts
become `/32` or `/128`, adjacent networks merge, and networks contained in a
larger one are dropped. Without `aggregate=true` addresses pass through as
written.

#### Statistics

```bash
curl http://localhost:8080/stats
```

```json
{
  "staticlist": {
    "name": "staticlist",
    "total_entries": 4,
    "url_entries": 0,
    "file_entries": 0,
    "static_entries": 4,
    "invalid_entries": 0
  }
}
```

Counts come from the last generation of each list in this process. Lists never
generated report zeros, and `invalid_entries` is how many source lines were
dropped as not being valid addresses.

## Operational Notes

- **No caching.** Every request regenerates from scratch and re-fetches every
  URL of every list involved. Ten routers polling `/lists/all` produce ten full
  downloads from each upstream.
- **No authentication or rate limiting.** All endpoints, including `/metrics`
  and `/stats`, are open. Put the service behind a reverse proxy, firewall or
  mTLS if it is reachable from untrusted networks.
- **`/lists/all` is all-or-nothing.** If one source fails, the whole request
  returns `500` — including the lists that generated fine. Fetch per list if
  partial results matter.
- **Fetch timeouts are fixed.** Each URL gets 30 s and URLs are fetched
  sequentially, while the HTTP server aborts a response after 60 s. A list with
  several slow sources can therefore be cut off mid-response.
- **State is per process.** `/stats` and the entry-count metric reflect only the
  replica that served the request, and reset on restart. With several replicas
  the numbers differ between pods.
- **Configuration is read once at startup.** Editing the file — or a mounted
  ConfigMap — has no effect until the process restarts.

## Running with Docker

### Docker Compose

1. Create the configuration. `config/config.yaml` is deliberately not in the
   repository, and compose mounts exactly that path:
```bash
mkdir -p config
cp config.example.yaml config/config.yaml
```

2. Start the service:
```bash
docker-compose up -d
```

3. With monitoring (Prometheus + Grafana):
```bash
docker-compose --profile monitoring up -d
```

The monitoring profile starts Grafana with the default `admin`/`admin`
credentials and publishes Prometheus with its lifecycle API enabled — keep both
off untrusted networks or override them.

Compose builds the image from the local Dockerfile rather than pulling from the
registry.

### Standalone Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/shidoh/mk-addrlist-generator:latest
```

## Running in Kubernetes

### Using Helm

The chart is published as an OCI artifact, so there is no repository to add:

```bash
helm install mk-addrlist-generator \
  oci://ghcr.io/shidoh/charts/mk-addrlist-generator \
  --version <chart-version> \
  -f my-values.yaml
```

`config.lists` is empty by default and the service refuses to start without at
least one list, so values are mandatory:

```yaml
image:
  tag: "v0.0.5" # pin a released tag: the chart default is appVersion 0.1.0, for which no image exists

config:
  timeout: 1d
  commentPrefix: "blocklist"
  lists:
    blocklist:
      timeout: 12h
      urls:
        - https://lists.example.com/blocklist.txt
```

The deployment annotates pods with a checksum of the rendered ConfigMap, so
`helm upgrade` restarts them when the configuration changes. Editing the
ConfigMap by hand does not.

### Health Checks for Kubernetes

The service provides standard health endpoints for Kubernetes probes:

```yaml
livenessProbe:
  httpGet:
    path: /live
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

Readiness only reports whether lists are configured; it does not check that
sources are reachable.

## Prometheus Metrics

Available metrics:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `mk_addrlist_http_requests_total` | Counter | `method`, `endpoint`, `status` | Total HTTP requests |
| `mk_addrlist_http_request_duration_seconds` | Histogram | `method`, `endpoint` | HTTP request duration |
| `mk_addrlist_list_entries_total` | Gauge | `list_name`, `source_type` | Entries per list |
| `mk_addrlist_list_generation_duration_seconds` | Histogram | `list_name`, `format` | List generation duration |
| `mk_addrlist_list_generation_errors_total` | Counter | `list_name` | List generation errors |

`source_type` is one of `url`, `file`, `static`, `total`. Requests that match no
route are recorded under `endpoint="unmatched"`.

`mk_addrlist_list_entries_total` is refreshed when `/stats` is requested, not on
every generation, so scraping `/metrics` alone leaves it at zero.

## Using with ipset/iptables

```bash
# Create ipset and populate it
ipset create blocklist hash:net
curl -s "http://localhost:8080/list/staticlist?format=plain" | while read ip; do
  ipset add blocklist "$ip" 2>/dev/null
done

# Use in iptables
iptables -I INPUT -m set --match-set blocklist src -j DROP
```

## Using with nftables

```bash
# Download and include the nftables set
curl -s "http://localhost:8080/list/staticlist?format=nftables" > /etc/nftables.d/staticlist.nft

# Include in your nftables config
# include "/etc/nftables.d/staticlist.nft"
```

## Development

### Prerequisites

- Go 1.26 or later
- Docker (for containerization)
- golangci-lint (for linting)

### Building

```bash
# Build
go build -o mk-addrlist-generator .

# Run tests with the race detector, as CI does
go test -race ./...

# Run linter
golangci-lint run --timeout=5m

# Build Docker image
docker build -t mk-addrlist-generator .
```

### Running Locally

```bash
cp config.example.yaml config.yaml
./mk-addrlist-generator serve -c config.yaml -v
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
