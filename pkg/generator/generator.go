package generator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mk-addrlist-generator/pkg/config"
	"mk-addrlist-generator/pkg/netutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	// FormatMikrotik is the default MikroTik script format
	FormatMikrotik OutputFormat = "mikrotik"
	// FormatPlain outputs just IP addresses/networks, one per line
	FormatPlain OutputFormat = "plain"
	// FormatJSON outputs entries as JSON
	FormatJSON OutputFormat = "json"
	// FormatNftables outputs nftables set format
	FormatNftables OutputFormat = "nftables"
)

// AllFormats returns all supported output formats
func AllFormats() []OutputFormat {
	return []OutputFormat{FormatMikrotik, FormatPlain, FormatJSON, FormatNftables}
}

// ParseFormat parses a format string into OutputFormat
func ParseFormat(s string) OutputFormat {
	switch strings.ToLower(s) {
	case "mikrotik":
		return FormatMikrotik
	case "plain":
		return FormatPlain
	case "json":
		return FormatJSON
	case "nftables":
		return FormatNftables
	default:
		return FormatMikrotik
	}
}

const scriptTemplate = `{{if .IPv4Entries}}
/ip/firewall/address-list/remove [ find where list="{{.ListName}}" ];
:global {{.ListName}}AddIP;
:set {{.ListName}}AddIP do={
:do { /ip/firewall/address-list/add list={{.ListName}} address=$1 comment="$2" timeout=$3; } on-error={ }
}
{{range .IPv4Entries}}
${{$.ListName}}AddIP "{{.Address}}" "{{.Comment}}" "{{.Timeout}}"{{end}}

:set {{.ListName}}AddIP;
{{end}}{{if .IPv6Entries}}
/ipv6/firewall/address-list/remove [ find where list="{{.ListName}}" ];
:global {{.ListName}}AddIPv6;
:set {{.ListName}}AddIPv6 do={
:do { /ipv6/firewall/address-list/add list={{.ListName}} address=$1 comment="$2" timeout=$3; } on-error={ }
}
{{range .IPv6Entries}}
${{$.ListName}}AddIPv6 "{{.Address}}" "{{.Comment}}" "{{.Timeout}}"{{end}}

:set {{.ListName}}AddIPv6;
{{end}}`

const nftablesTemplate = `# nftables set definition for {{.ListName}}
# Generated at {{.Timestamp}}
# Total entries: {{len .Entries}}

define {{.ListName}}_v4 = {
{{range .IPv4Entries}}    {{.Address}},
{{end}}}

define {{.ListName}}_v6 = {
{{range .IPv6Entries}}    {{.Address}},
{{end}}}

# Example usage:
# nft add set inet filter {{.ListName}}_v4 { type ipv4_addr; flags interval; elements = ${{.ListName}}_v4 }
# nft add set inet filter {{.ListName}}_v6 { type ipv6_addr; flags interval; elements = ${{.ListName}}_v6 }
`

// GeneratorOptions configures generator behavior
type GeneratorOptions struct {
	// HTTPTimeout for fetching external URLs
	HTTPTimeout time.Duration
}

// DefaultOptions returns default generator options
func DefaultOptions() GeneratorOptions {
	return GeneratorOptions{
		HTTPTimeout: 30 * time.Second,
	}
}

// RequestOptions configures per-request generation behavior.
// These are not stored on the Generator — each call passes its own copy.
type RequestOptions struct {
	// Deduplicate removes duplicate addresses within each source type
	Deduplicate bool
	// Aggregate merges adjacent CIDR networks
	Aggregate bool
}

// DefaultRequestOptions returns default per-request options
func DefaultRequestOptions() RequestOptions {
	return RequestOptions{
		Deduplicate: true,
		Aggregate:   false,
	}
}

type Generator struct {
	cfg        *config.Config
	options    GeneratorOptions
	mu         sync.RWMutex
	httpClient *http.Client
	statsCache map[string]ListStats
}

type ScriptData struct {
	ListName    string
	Timestamp   string
	Entries     []Entry
	IPv4Entries []Entry
	IPv6Entries []Entry
}

type Entry struct {
	Address string `json:"address"`
	Comment string `json:"comment"`
	Timeout string `json:"timeout"`
}

// JSONOutput represents the JSON output structure
type JSONOutput struct {
	ListName  string  `json:"list_name"`
	Timestamp string  `json:"timestamp"`
	Count     int     `json:"count"`
	Entries   []Entry `json:"entries"`
}

// JSONAllOutput represents JSON output for all lists
type JSONAllOutput struct {
	Timestamp string                `json:"timestamp"`
	Lists     map[string]JSONOutput `json:"lists"`
}

func NewGenerator(cfg *config.Config) *Generator {
	return NewGeneratorWithOptions(cfg, DefaultOptions())
}

func NewGeneratorWithOptions(cfg *config.Config, options GeneratorOptions) *Generator {
	return &Generator{
		cfg:        cfg,
		options:    options,
		statsCache: make(map[string]ListStats),
		httpClient: &http.Client{
			Timeout: options.HTTPTimeout,
		},
	}
}

// SetOptions updates generator options
func (g *Generator) SetOptions(options GeneratorOptions) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.options = options
	g.httpClient.Timeout = options.HTTPTimeout
}

// GetOptions returns current generator options
func (g *Generator) GetOptions() GeneratorOptions {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.options
}

func (g *Generator) GenerateAll() (string, error) {
	return g.GenerateAllWithFormat(FormatMikrotik, DefaultRequestOptions())
}

func (g *Generator) GenerateAllWithFormat(format OutputFormat, reqOpts RequestOptions) (string, error) {
	// Special handling for JSON format - combine all lists
	if format == FormatJSON {
		return g.generateAllJSON(reqOpts)
	}

	var result strings.Builder

	for name, list := range g.cfg.Lists {
		script, err := g.GenerateListWithFormat(name, list, format, reqOpts)
		if err != nil {
			return "", fmt.Errorf("error generating list %s: %v", name, err)
		}
		result.WriteString(script)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (g *Generator) generateAllJSON(reqOpts RequestOptions) (string, error) {
	output := JSONAllOutput{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Lists:     make(map[string]JSONOutput),
	}

	for name, list := range g.cfg.Lists {
		entries, err := g.collectEntries(name, list, reqOpts)
		if err != nil {
			return "", fmt.Errorf("error collecting entries for list %s: %v", name, err)
		}

		output.Lists[name] = JSONOutput{
			ListName:  name,
			Timestamp: output.Timestamp,
			Count:     len(entries),
			Entries:   entries,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %v", err)
	}

	return string(data), nil
}

func (g *Generator) GenerateList(name string, list config.List) (string, error) {
	return g.GenerateListWithFormat(name, list, FormatMikrotik, DefaultRequestOptions())
}

func (g *Generator) GenerateListWithFormat(name string, list config.List, format OutputFormat, reqOpts RequestOptions) (string, error) {
	entries, err := g.collectEntries(name, list, reqOpts)
	if err != nil {
		return "", err
	}

	// Generate output based on format
	switch format {
	case FormatPlain:
		return g.generatePlainOutput(entries), nil
	case FormatJSON:
		return g.generateJSONOutput(name, entries)
	case FormatNftables:
		return g.generateNftablesOutput(name, entries)
	case FormatMikrotik:
		fallthrough
	default:
		return g.generateMikrotikOutput(name, entries)
	}
}

func (g *Generator) collectEntries(name string, list config.List, reqOpts RequestOptions) ([]Entry, error) {
	timeout, err := list.GetTimeout(g.cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("error getting timeout: %v", err)
	}

	commentPrefix := list.GetCommentPrefix(g.cfg.Config)
	entries := make([]Entry, 0)

	// Process URLs - with deduplication within source type
	urlAddresses := make([]string, 0)
	for _, url := range list.URLs {
		addresses, err := g.fetchAddresses(url)
		if err != nil {
			return nil, fmt.Errorf("error fetching addresses from %s: %v", url, err)
		}
		urlAddresses = append(urlAddresses, addresses...)
	}
	if reqOpts.Deduplicate {
		urlAddresses = netutil.DeduplicateStrings(urlAddresses)
	}
	if reqOpts.Aggregate && len(urlAddresses) > 0 {
		urlAddresses, err = netutil.ParseAndAggregate(urlAddresses)
		if err != nil {
			return nil, fmt.Errorf("error aggregating URL addresses: %v", err)
		}
	}
	for _, addr := range urlAddresses {
		entries = append(entries, Entry{
			Address: addr,
			Comment: fmt.Sprintf("%s/external", commentPrefix),
			Timeout: timeout.String(),
		})
	}

	// Process files - with deduplication within source type
	fileAddresses := make([]string, 0)
	for _, file := range list.Files {
		addresses, err := g.readAddresses(file)
		if err != nil {
			return nil, fmt.Errorf("error reading addresses from %s: %v", file, err)
		}
		fileAddresses = append(fileAddresses, addresses...)
	}
	if reqOpts.Deduplicate {
		fileAddresses = netutil.DeduplicateStrings(fileAddresses)
	}
	if reqOpts.Aggregate && len(fileAddresses) > 0 {
		fileAddresses, err = netutil.ParseAndAggregate(fileAddresses)
		if err != nil {
			return nil, fmt.Errorf("error aggregating file addresses: %v", err)
		}
	}
	for _, addr := range fileAddresses {
		entries = append(entries, Entry{
			Address: addr,
			Comment: fmt.Sprintf("%s/file", commentPrefix),
			Timeout: timeout.String(),
		})
	}

	// Process static addresses - with deduplication within source type
	staticAddresses := list.Addresses
	if reqOpts.Deduplicate {
		staticAddresses = netutil.DeduplicateStrings(staticAddresses)
	}
	if reqOpts.Aggregate && len(staticAddresses) > 0 {
		staticAddresses, err = netutil.ParseAndAggregate(staticAddresses)
		if err != nil {
			return nil, fmt.Errorf("error aggregating static addresses: %v", err)
		}
	}
	for _, addr := range staticAddresses {
		entries = append(entries, Entry{
			Address: addr,
			Comment: fmt.Sprintf("%s/static", commentPrefix),
			Timeout: timeout.String(),
		})
	}

	// Update stats cache with counts from this generation
	stat := ListStats{
		Name:          name,
		URLEntries:    len(urlAddresses),
		FileEntries:   len(fileAddresses),
		StaticEntries: len(staticAddresses),
		TotalEntries:  len(urlAddresses) + len(fileAddresses) + len(staticAddresses),
	}
	g.mu.Lock()
	g.statsCache[name] = stat
	g.mu.Unlock()

	return entries, nil
}

func (g *Generator) generatePlainOutput(entries []Entry) string {
	var result strings.Builder
	for _, entry := range entries {
		result.WriteString(entry.Address)
		result.WriteString("\n")
	}
	return result.String()
}

func (g *Generator) generateJSONOutput(name string, entries []Entry) (string, error) {
	output := JSONOutput{
		ListName:  name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Count:     len(entries),
		Entries:   entries,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %v", err)
	}

	return string(data), nil
}

func (g *Generator) generateNftablesOutput(name string, entries []Entry) (string, error) {
	tmpl, err := template.New("nftables").Parse(nftablesTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing nftables template: %v", err)
	}

	// Separate IPv4 and IPv6 entries
	var ipv4Entries, ipv6Entries []Entry
	for _, entry := range entries {
		ipnet, err := netutil.ParseIPOrCIDR(entry.Address)
		if err != nil {
			continue // Skip invalid addresses
		}
		if ipnet.IsIPv4() {
			ipv4Entries = append(ipv4Entries, entry)
		} else {
			ipv6Entries = append(ipv6Entries, entry)
		}
	}

	data := ScriptData{
		ListName:    name,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Entries:     entries,
		IPv4Entries: ipv4Entries,
		IPv6Entries: ipv6Entries,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing nftables template: %v", err)
	}

	return buf.String(), nil
}

func (g *Generator) generateMikrotikOutput(name string, entries []Entry) (string, error) {
	tmpl, err := template.New("script").Parse(scriptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	// Separate IPv4 and IPv6 entries
	var ipv4Entries, ipv6Entries []Entry
	for _, entry := range entries {
		ipnet, err := netutil.ParseIPOrCIDR(entry.Address)
		if err != nil {
			continue // Skip invalid addresses
		}
		if ipnet.IsIPv4() {
			ipv4Entries = append(ipv4Entries, entry)
		} else {
			ipv6Entries = append(ipv6Entries, entry)
		}
	}

	data := ScriptData{
		ListName:    name,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Entries:     entries,
		IPv4Entries: ipv4Entries,
		IPv6Entries: ipv6Entries,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return buf.String(), nil
}

func (g *Generator) fetchAddresses(url string) ([]string, error) {
	resp, err := g.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return readAddresses(resp.Body)
}

func (g *Generator) readAddresses(path string) ([]string, error) {
	cleanPath := filepath.Clean(path)
	file, err := os.Open(cleanPath) // #nosec G304 - path is cleaned with filepath.Clean
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readAddresses(file)
}

func readAddresses(r io.Reader) ([]string, error) {
	var addresses []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			// Remove inline comments
			if idx := strings.Index(line, "#"); idx != -1 {
				line = strings.TrimSpace(line[:idx])
			}
			if line != "" {
				addresses = append(addresses, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return addresses, nil
}

// GetListStats returns statistics about a list
type ListStats struct {
	Name          string `json:"name"`
	TotalEntries  int    `json:"total_entries"`
	URLEntries    int    `json:"url_entries"`
	FileEntries   int    `json:"file_entries"`
	StaticEntries int    `json:"static_entries"`
}

// GetStats returns statistics for all lists.
// Counts reflect the last successful generation for each list.
// Lists that have never been generated will show zero counts.
func (g *Generator) GetStats() map[string]ListStats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	stats := make(map[string]ListStats, len(g.cfg.Lists))
	for name := range g.cfg.Lists {
		if s, ok := g.statsCache[name]; ok {
			stats[name] = s
		} else {
			stats[name] = ListStats{Name: name}
		}
	}
	return stats
}
