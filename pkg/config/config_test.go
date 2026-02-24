package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
config:
  timeout: 4h
  commentPrefix: "Default comment"

lists:
  blocklist:
    timeout: 3h59m54s
    commentPrefix: "Combined blocklist entry"
    urls:
      - https://example.com/list1.txt
    files:
      - list1.txt
    addresses:
      - 172.16.1.0/24
      - 8.8.8.8
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify global config
	if cfg.Config.Timeout != "4h" {
		t.Errorf("Expected global timeout '4h', got '%s'", cfg.Config.Timeout)
	}
	if cfg.Config.CommentPrefix != "Default comment" {
		t.Errorf("Expected global comment prefix 'Default comment', got '%s'", cfg.Config.CommentPrefix)
	}

	// Verify list config
	list, exists := cfg.Lists["blocklist"]
	if !exists {
		t.Fatal("Expected 'blocklist' to exist")
	}

	if list.Timeout != "3h59m54s" {
		t.Errorf("Expected list timeout '3h59m54s', got '%s'", list.Timeout)
	}
	if list.CommentPrefix != "Combined blocklist entry" {
		t.Errorf("Expected list comment prefix 'Combined blocklist entry', got '%s'", list.CommentPrefix)
	}

	// Verify list sources
	if len(list.URLs) != 1 || list.URLs[0] != "https://example.com/list1.txt" {
		t.Errorf("Unexpected URLs: %v", list.URLs)
	}
	if len(list.Files) != 1 || list.Files[0] != "list1.txt" {
		t.Errorf("Unexpected Files: %v", list.Files)
	}
	if len(list.Addresses) != 2 || list.Addresses[0] != "172.16.1.0/24" || list.Addresses[1] != "8.8.8.8" {
		t.Errorf("Unexpected Addresses: %v", list.Addresses)
	}
}

func TestList_GetTimeout(t *testing.T) {
	tests := []struct {
		name         string
		listTimeout  string
		globalTimeout string
		want         time.Duration
		wantErr      bool
	}{
		{
			name:         "list timeout",
			listTimeout:  "2h",
			globalTimeout: "1h",
			want:         2 * time.Hour,
			wantErr:      false,
		},
		{
			name:         "global timeout",
			listTimeout:  "",
			globalTimeout: "1h",
			want:         time.Hour,
			wantErr:      false,
		},
		{
			name:         "default timeout",
			listTimeout:  "",
			globalTimeout: "",
			want:         time.Hour,
			wantErr:      false,
		},
		{
			name:         "invalid timeout",
			listTimeout:  "invalid",
			globalTimeout: "1h",
			want:         0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &List{Timeout: tt.listTimeout}
			got, err := l.GetTimeout(tt.globalTimeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("List.GetTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("List.GetTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestList_GetCommentPrefix(t *testing.T) {
	tests := []struct {
		name         string
		listPrefix   string
		globalPrefix string
		want         string
	}{
		{
			name:         "list prefix",
			listPrefix:   "list",
			globalPrefix: "global",
			want:         "list",
		},
		{
			name:         "global prefix",
			listPrefix:   "",
			globalPrefix: "global",
			want:         "global",
		},
		{
			name:         "empty prefixes",
			listPrefix:   "",
			globalPrefix: "",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &List{CommentPrefix: tt.listPrefix}
			if got := l.GetCommentPrefix(tt.globalPrefix); got != tt.want {
				t.Errorf("List.GetCommentPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Lists: map[string]List{
					"test": {
						Addresses: []string{"192.168.1.0/24"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple sources",
			config: &Config{
				Lists: map[string]List{
					"test": {
						Addresses: []string{"192.168.1.0/24"},
						URLs:     []string{"https://example.com/list.txt"},
						Files:    []string{"list.txt"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no lists",
			config: &Config{
				Lists: nil,
			},
			wantErr: true,
		},
		{
			name: "empty list sources",
			config: &Config{
				Lists: map[string]List{
					"test": {},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: &Config{
				Lists: map[string]List{
					"test": {
						Addresses: []string{"192.168.1.0/24"},
						Timeout:   "invalid",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateConfig(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}