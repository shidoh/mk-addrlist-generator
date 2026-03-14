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
- **Flexible timeout formats:**
  - Days (e.g., "1d", "7d")
  - Hours and minutes (e.g., "12h30m")
  - Minutes and seconds (e.g., "45m30s")
  - Complex durations (e.g., "2d3h45m30s")
- **Data processing:**
  - Deduplication within source types
  - CIDR aggregation (merge adjacent networks)
- **Observability:**
  - Health check endpoints (`/health`, `/ready`, `/live`)
  - Prometheus metrics (`/metrics`)
  - Structured JSON logging
- **Production ready:**
  - Graceful shutdown
  - Multi-arch Docker images (amd64, arm64, arm/v7)
  - Kubernetes/Helm support

## Installation

### Using Pre-built Binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/shidoh/mk-addrlist-generator/releases).

### Using Docker

The Docker image is available on GitHub Container Registry:

```bash
docker pull ghcr.io/shidoh/mk-addrlist-generator:latest
```

Multi-arch images are available for:
- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`

### Building from Source

```bash
go install github.com/shidoh/mk-addrlist-generator@latest
```

Or clone and build:

```bash
git clone https://github.com/shidoh/mk-addrlist-generator.git
cd mk-addrlist-generator
go build -o mk-addrlist-generator .
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
- `-o, --output` - Output file (default: stdout)
- `-a, --aggregate` - Enable CIDR aggregation
- `-d, --deduplicate` - Enable deduplication (default: `true`)

### Validate

Validate the configuration file:

```bash
mk-addrlist-generator validate -c config.yaml
```

### Version

Print version information:

```bash
mk-addrlist-generator version
```

## Configuration

The service is configured using a YAML file. Here's an example configuration:

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
```

## HTTP API

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | API information |
| `GET /health`, `/healthz` | Health check |
| `GET /ready`, `/readyz` | Readiness check |
| `GET /live`, `/livez` | Liveness check |
| `GET /metrics` | Prometheus metrics |
| `GET /lists/all` | Get all address lists |
| `GET /list/:name` | Get a specific list |
| `GET /lists` | List all available list names |
| `GET /stats` | Get statistics for all lists |

### Query Parameters

| Parameter | Values | Description |
|-----------|--------|-------------|
| `format` | `mikrotik`, `plain`, `json`, `nftables` | Output format |
| `aggregate` | `true`, `false` | Enable CIDR aggregation |
| `deduplicate` | `true`, `false` | Enable deduplication (default: true) |

### Examples

#### Get All Lists

```bash
curl http://localhost:8080/lists/all
```

Example response (MikroTik format):
```
/ip/firewall/address-list/remove [ find where list="externallists" ];
:global externallistsAddIP;
:set externallistsAddIP do={
:do { /ip/firewall/address-list/add list=externallists address=$1 comment="$2" timeout=$3; } on-error={ }
}
$externallistsAddIP "192.168.1.1" "crowdsecurity/external" "3h59m54s"
$externallistsAddIP "10.0.0.0/24" "crowdsecurity/external" "3h59m54s"

:set externallistsAddIP;
```

#### Get Specific List

```bash
curl http://localhost:8080/list/staticlist
```

#### Get in Plain Format

```bash
curl http://localhost:8080/lists/all?format=plain
```

Example response:
```
192.168.1.1
10.0.0.0/24
172.16.1.0/24
```

#### Get in JSON Format

```bash
curl http://localhost:8080/list/staticlist?format=json
```

Example response:
```json
{
  "list_name": "staticlist",
  "timestamp": "2024-01-15T10:30:00Z",
  "count": 3,
  "entries": [
    {"address": "172.16.1.0/24", "comment": "static/static", "timeout": "45m30s"},
    {"address": "8.8.8.8", "comment": "static/static", "timeout": "45m30s"},
    {"address": "172.27.0.0/21", "comment": "static/static", "timeout": "45m30s"}
  ]
}
```

#### Get in nftables Format

```bash
curl http://localhost:8080/list/staticlist?format=nftables
```

Example response:
```
# nftables set definition for staticlist
# Generated at 2024-01-15T10:30:00Z
# Total entries: 3

define staticlist_v4 = {
    172.16.1.0/24,
    8.8.8.8/32,
    172.27.0.0/21,
}

define staticlist_v6 = {
}
```

#### Get with CIDR Aggregation

```bash
curl "http://localhost:8080/list/staticlist?aggregate=true"
```

## Running with Docker

### Docker Compose

1. Create configuration:
```bash
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

### Standalone Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/shidoh/mk-addrlist-generator:latest
```

## Running in Kubernetes

### Using Helm

1. Add the Helm repository:
```bash
helm repo add mk-addrlist-generator oci://ghcr.io/shidoh/charts
```

2. Install the chart:
```bash
helm install mk-addrlist-generator mk-addrlist-generator/mk-addrlist-generator
```

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

## Prometheus Metrics

Available metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `mk_addrlist_http_requests_total` | Counter | Total HTTP requests |
| `mk_addrlist_http_request_duration_seconds` | Histogram | HTTP request duration |
| `mk_addrlist_list_entries_total` | Gauge | Entries per list |
| `mk_addrlist_list_generation_duration_seconds` | Histogram | List generation duration |
| `mk_addrlist_list_generation_errors_total` | Counter | List generation errors |

## Using with ipset/iptables

```bash
# Create ipset and populate it
ipset create blocklist hash:net
curl -s http://localhost:8080/list/staticlist?format=plain | while read ip; do
  ipset add blocklist "$ip" 2>/dev/null
done

# Use in iptables
iptables -I INPUT -m set --match-set blocklist src -j DROP
```

## Using with nftables

```bash
# Download and include the nftables set
curl -s http://localhost:8080/list/staticlist?format=nftables > /etc/nftables.d/staticlist.nft

# Include in your nftables config
# include "/etc/nftables.d/staticlist.nft"
```

## Development

### Prerequisites

- Go 1.21 or later
- Docker (for containerization)
- golangci-lint (for linting)

### Building

```bash
# Build
go build -o mk-addrlist-generator .

# Run tests
go test -v ./...

# Run linter
golangci-lint run

# Build Docker image
docker build -t mk-addrlist-generator .
```

### Running Locally

```bash
./mk-addrlist-generator serve -c config.yaml -v
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
