package generator

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mk-addrlist-generator/pkg/config"
	"net/http"
	"os"
	"strings"
	"text/template"
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

type Generator struct {
	cfg *config.Config
}

type ScriptData struct {
	ListName string
	Entries  []Entry
}

type Entry struct {
	Address string
	Comment string
	Timeout string
}

func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{cfg: cfg}
}

func (g *Generator) GenerateAll() (string, error) {
	var result strings.Builder

	for name, list := range g.cfg.Lists {
		script, err := g.GenerateList(name, list)
		if err != nil {
			return "", fmt.Errorf("error generating list %s: %v", name, err)
		}
		result.WriteString(script)
		result.WriteString("\n")
	}

	return result.String(), nil
}

func (g *Generator) GenerateList(name string, list config.List) (string, error) {
	timeout, err := list.GetTimeout(g.cfg.Config)
	if err != nil {
		return "", fmt.Errorf("error getting timeout: %v", err)
	}

	commentPrefix := list.GetCommentPrefix(g.cfg.Config)
	entries := make([]Entry, 0)

	// Process URLs
	for _, url := range list.URLs {
		addresses, err := g.fetchAddresses(url)
		if err != nil {
			return "", fmt.Errorf("error fetching addresses from %s: %v", url, err)
		}
		for _, addr := range addresses {
			entries = append(entries, Entry{
				Address: addr,
				Comment: fmt.Sprintf("%s/external", commentPrefix),
				Timeout: timeout.String(),
			})
		}
	}

	// Process files
	for _, file := range list.Files {
		addresses, err := g.readAddresses(file)
		if err != nil {
			return "", fmt.Errorf("error reading addresses from %s: %v", file, err)
		}
		for _, addr := range addresses {
			entries = append(entries, Entry{
				Address: addr,
				Comment: fmt.Sprintf("%s/file", commentPrefix),
				Timeout: timeout.String(),
			})
		}
	}

	// Process static addresses
	for _, addr := range list.Addresses {
		entries = append(entries, Entry{
			Address: addr,
			Comment: fmt.Sprintf("%s/static", commentPrefix),
			Timeout: timeout.String(),
		})
	}

	// Generate script
	tmpl, err := template.New("script").Parse(scriptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %v", err)
	}

	data := ScriptData{
		ListName: name,
		Entries:  entries,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return buf.String(), nil
}

func (g *Generator) fetchAddresses(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return readAddresses(resp.Body)
}

func (g *Generator) readAddresses(path string) ([]string, error) {
	file, err := os.Open(path)
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
