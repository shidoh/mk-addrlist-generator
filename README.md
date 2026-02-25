# MikroTik Address List Generator

A service that generates MikroTik address lists from various sources (URLs, files, static addresses) and provides them via HTTP API.

## Features

- Multiple source types:
  - External URLs (HTTP/HTTPS)
  - Local files
  - Static addresses in configuration
- Flexible timeout formats:
  - Days (e.g., "1d", "7d")
  - Hours and minutes (e.g., "12h30m")
  - Minutes and seconds (e.g., "45m30s")
  - Complex durations (e.g., "2d3h45m30s")
- HTTP API endpoints:
  - `/lists/all` - Get all address lists
  - `/list/<name>` - Get a specific list by name
- Configurable comment prefixes
- Docker and Kubernetes support

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

## Running with Docker

1. Create a configuration file:
   ```bash
   cp config.example.yaml config/config.yaml
   ```

2. Edit the configuration file:
   ```bash
   vim config.yaml
   ```

3. Start the service using Docker Compose:
   ```bash
   docker-compose up -d
   ```

## Running in Kubernetes

1. Add the Helm repository:
   ```bash
   helm repo add mk-addrlist-generator https://example.com/charts
   ```

2. Install the chart:
   ```bash
   helm install mk-addrlist-generator mk-addrlist-generator/mk-addrlist-generator
   ```

## API Usage

### Get All Lists

```bash
curl http://localhost:8080/lists/all
```

Example response:
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

### Get Specific List

```bash
curl http://localhost:8080/list/staticlist
```

Example response:
```
/ip/firewall/address-list/remove [ find where list="staticlist" ];
:global staticlistAddIP;
:set staticlistAddIP do={
:do { /ip/firewall/address-list/add list=staticlist address=$1 comment="$2" timeout=$3; } on-error={ }
}
$staticlistAddIP "172.16.1.0/24" "static" "45m30s"
$staticlistAddIP "8.8.8.8" "static" "45m30s"
$staticlistAddIP "172.27.0.0/21" "static" "45m30s"

:set staticlistAddIP;
```

## Development

### Prerequisites

- Go 1.21 or later
- Docker (for containerization)
- Kubernetes (for deployment)

### Building

```bash
go build
```

### Testing

```bash
go test -v ./...
```

### Running Locally

```bash
./mk-addrlist-generator --config config.yaml