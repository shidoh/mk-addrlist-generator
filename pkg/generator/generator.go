package generator

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"mk-addrlist-generator/pkg/config"
)

const scriptTemplate = `
/ip/firewall/address-list/remove [ find where list="{{.ListName}}" ];
:global {{.ListName}}AddIP;
:set {{.ListName}}AddIP do={
:do { /ip/firewall/address-list/add list={{.ListName}} address=$1 comment="$2" timeout=$3; } on-error={ }
}
{{range .Entries}}
${{$.ListName}}AddIP "{{.Address}}" "{{.Comment}}" "{{.Timeout}}"{{end}}

:set {{.ListName}}AddIP;
`

type ScriptData struct {
	ListName string
	Entries  []Entry
}

type Entry struct {
	Address string
	Comment string
	Timeout string
}

type Generator struct {
	config *config.Config
}

func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{config: cfg}
}

// GenerateAll generates scripts for all configured lists
func (g *Generator) GenerateAll() (map[string]string, error) {
	result := make(map[string]string)
	for name, list := range g.config.Lists {
		script, err := g.GenerateList(name, &list)
		if err != nil {
			return nil, fmt.Errorf("error generating list %q: %w", name, err)
		}
		result[name] = script
	}
	return result, nil
}

// GenerateList generates a script for a single list
func (g *Generator) GenerateList(name string, list *config.List) (string, error) {
	timeout, err := list.GetTimeout(g.config.Config.Timeout)
	if err != nil {
		return "", fmt.Errorf("invalid timeout: %w", err)
	}

	commentPrefix := list.GetCommentPrefix(g.config.Config.CommentPrefix)
	entries, err := g.getEntries(list, commentPrefix, timeout.String())
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("script").Parse(scriptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	data := ScriptData{
		ListName: name,
		Entries:  entries,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

// getEntries retrieves entries from all configured sources
func (g *Generator) getEntries(list *config.List, commentPrefix, timeout string) ([]Entry, error) {
	var allAddresses []string

	// Collect addresses from external URLs
	if len(list.URLs) > 0 {
		addresses, err := g.getExternalAddresses(list.URLs)
		if err != nil {
			return nil, fmt.Errorf("error getting external addresses: %w", err)
		}
		allAddresses = append(allAddresses, addresses...)
	}

	// Collect addresses from files
	if len(list.Files) > 0 {
		addresses, err := g.getFileAddresses(list.Files)
		if err != nil {
			return nil, fmt.Errorf("error getting file addresses: %w", err)
		}
		allAddresses = append(allAddresses, addresses...)
	}

	// Add static addresses
	allAddresses = append(allAddresses, list.Addresses...)

	// Create entries from all collected addresses
	entries := make([]Entry, 0, len(allAddresses))
	seen := make(map[string]bool)

	for _, addr := range allAddresses {
		if addr = strings.TrimSpace(addr); addr == "" {
			continue
		}
		// Skip duplicates
		if seen[addr] {
			continue
		}
		seen[addr] = true
		entries = append(entries, Entry{
			Address: addr,
			Comment: commentPrefix,
			Timeout: timeout,
		})
	}

	return entries, nil
}

// getExternalAddresses retrieves addresses from external URLs
func (g *Generator) getExternalAddresses(urls []string) ([]string, error) {
	var addresses []string
	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("error fetching URL %q: %w", url, err)
		}
		defer resp.Body.Close()

		addrs, err := readAddresses(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading addresses from URL %q: %w", url, err)
		}
		addresses = append(addresses, addrs...)
	}
	return addresses, nil
}

// getFileAddresses retrieves addresses from files
func (g *Generator) getFileAddresses(paths []string) ([]string, error) {
	var addresses []string
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("error opening file %q: %w", path, err)
		}
		defer file.Close()

		addrs, err := readAddresses(file)
		if err != nil {
			return nil, fmt.Errorf("error reading addresses from file %q: %w", path, err)
		}
		addresses = append(addresses, addrs...)
	}
	return addresses, nil
}

// readAddresses reads addresses line by line from a reader
func readAddresses(r io.Reader) ([]string, error) {
	var addresses []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove inline comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			addresses = append(addresses, line)
		}
	}
	return addresses, scanner.Err()
}