# MikroTik Address List Generator

A Go service that generates MikroTik address lists from various sources (URLs, files, or static lists) and provides them via HTTP API.

## Features

- Generate MikroTik-compatible address list scripts
- Multiple source types in a single list:
  - External URLs (HTTP/HTTPS)
  - Local files
  - Static addresses
- Configurable timeouts and comments per list
- HTTP API endpoints for accessing lists
- Support for multiple independent lists
- Automatic deduplication of addresses

## Configuration

The service uses a YAML configuration file. See [`config.example.yaml`](config.example.yaml) for a complete example.

### Configuration Structure

```yaml
config:
  timeout: 4h  # Global default timeout
  commentPrefix: "Default comment"  # Global default comment prefix

lists:
  blocklist:  # List name used in API calls
    timeout: 3h59m54s  # Optional, overrides global
    commentPrefix: "Combined blocklist entry"  # Optional, overrides global
    urls:  # Optional: URLs to fetch addresses from
      - https://example.com/blocklist1.txt
      - https://example.com/blocklist2.txt
    files:  # Optional: Local files to read addresses from
      - lists/local-blocklist.txt
    addresses:  # Optional: Static addresses/networks
      - 172.16.1.0/24
      - 8.8.8.8
```

Each list can combine multiple sources:
- URLs for external blocklists
- Files for local address lists
- Static addresses defined in the configuration
- Addresses are automatically deduplicated

## API Endpoints

- `GET /lists/all` - Returns all configured address lists
- `GET /list/:name` - Returns a specific list by name (e.g., `/list/blocklist`)

## Deployment Options

### Local Build

```bash
go build -o mk-addrlist-generator
```

### Docker

Using Docker Compose:
```bash
docker-compose up --build
```

### Kubernetes

Using Helm:
```bash
helm install mk-addrlist mk-addrlist-generator
```

## MikroTik Integration

### Automatic List Update Script

You can use the following script in MikroTik to automatically fetch and apply the address lists:

```routeros
# Address List Update Script
:local name "mk-list-fetcher"
:local url "http://<service-ip>/lists/all"  # Or use /list/<name> for a specific list
:local fileName "list_generated.rsc"

# Log the start of the update process
:log info "$name starting address list update"

# Fetch the list from the service
:log info "$name fetching list from $url"
/tool fetch url="$url" mode=http dst-path=$fileName

# Check if the file was downloaded successfully
:if ([:len [/file find name=$fileName]] > 0) do={
    # Import and apply the list
    :log info "$name importing address list"
    /import file-name=$fileName
    :log info "$name address list update completed"
    
    # Optional: Remove the temporary file
    /file remove $fileName
} else={
    :log error "$name failed to fetch address list from $url"
}
```

### Scheduling Updates

To automatically update the lists periodically, add a scheduler entry:

```routeros
/system scheduler
add interval=1h name=update-lists on-event=":global UpdateLists [:parse [/file get list_update_script.rsc contents]]; \$UpdateLists" \
    policy=ftp,reboot,read,write,policy,test,password,sniff,sensitive,romon start-time=startup
```

Replace `<service-ip>` with your service's address and adjust the interval as needed.

## Example Output

The service generates MikroTik-compatible scripts that look like this:

```routeros
/ip/firewall/address-list/remove [ find where list="blocklist" ];
:global blocklistAddIP;
:set blocklistAddIP do={
:do { /ip/firewall/address-list/add list=blocklist address=$1 comment="$2" timeout=$3; } on-error={ }
}
$blocklistAddIP "192.168.1.0/24" "Combined blocklist entry" "4h"
$blocklistAddIP "10.0.0.0/8" "Combined blocklist entry" "4h"

:set blocklistAddIP;