package generator

import (
	"mk-addrlist-generator/pkg/config"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGenerator_GenerateList(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "addresses-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "192.168.1.0/24\n10.0.0.0/8\n"
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("172.16.0.0/12\n172.17.0.0/16\n"))
	}))
	defer ts.Close()

	cfg := &config.Config{
		Config: config.GlobalConfig{
			Timeout:       "1h",
			CommentPrefix: "Global comment",
		},
		Lists: map[string]config.List{
			"test": {
				Timeout:       "2h",
				CommentPrefix: "Test comment",
				URLs:         []string{ts.URL},
				Files:        []string{tmpfile.Name()},
				Addresses:    []string{"8.8.8.8", "1.1.1.1"},
			},
		},
	}

	g := NewGenerator(cfg)

	list := cfg.Lists["test"]
	script, err := g.GenerateList("test", &list)
	if err != nil {
		t.Fatalf("GenerateList() error = %v", err)
	}

	// Verify script content
	expectedAddresses := []string{
		"192.168.1.0/24",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"172.17.0.0/16",
		"8.8.8.8",
		"1.1.1.1",
	}

	for _, addr := range expectedAddresses {
		if !strings.Contains(script, addr) {
			t.Errorf("Generated script missing address %q", addr)
		}
	}

	// Verify script structure
	expectedParts := []string{
		"/ip/firewall/address-list/remove [ find where list=\"test\" ];",
		":global testAddIP;",
		":set testAddIP do={",
		"} on-error={ }",
		":set testAddIP;",
	}

	for _, part := range expectedParts {
		if !strings.Contains(script, part) {
			t.Errorf("Generated script missing part %q", part)
		}
	}
}

func TestGenerator_GenerateAll(t *testing.T) {
	cfg := &config.Config{
		Config: config.GlobalConfig{
			Timeout:       "1h",
			CommentPrefix: "Global comment",
		},
		Lists: map[string]config.List{
			"list1": {
				Addresses: []string{"192.168.1.0/24"},
			},
			"list2": {
				Addresses: []string{"10.0.0.0/8"},
			},
		},
	}

	g := NewGenerator(cfg)

	scripts, err := g.GenerateAll()
	if err != nil {
		t.Fatalf("GenerateAll() error = %v", err)
	}

	if len(scripts) != 2 {
		t.Errorf("Expected 2 scripts, got %d", len(scripts))
	}

	for name, script := range scripts {
		if !strings.Contains(script, name) {
			t.Errorf("Script for %q does not contain its name", name)
		}
	}
}

func TestReadAddresses(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "simple addresses",
			input:   "192.168.1.0/24\n10.0.0.0/8\n",
			want:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			wantErr: false,
		},
		{
			name:    "with comments",
			input:   "192.168.1.0/24 # Comment\n# Full comment line\n10.0.0.0/8\n",
			want:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			wantErr: false,
		},
		{
			name:    "empty lines",
			input:   "\n192.168.1.0/24\n\n10.0.0.0/8\n\n",
			want:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			wantErr: false,
		},
		{
			name:    "whitespace",
			input:   "  192.168.1.0/24  \n\t10.0.0.0/8\t\n",
			want:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			got, err := readAddresses(reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("readAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equalSlices(got, tt.want) {
				t.Errorf("readAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}