package config

import (
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid config",
			path:    "../../config.example.yaml",
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    "nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestList_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		list     List
		defaults ConfigDefaults
		want     time.Duration
		wantErr  bool
	}{
		{
			name: "days",
			list: List{
				Timeout: "2d",
			},
			defaults: ConfigDefaults{},
			want:     48 * time.Hour,
			wantErr:  false,
		},
		{
			name: "hours and minutes",
			list: List{
				Timeout: "12h30m",
			},
			defaults: ConfigDefaults{},
			want:     12*time.Hour + 30*time.Minute,
			wantErr:  false,
		},
		{
			name: "minutes and seconds",
			list: List{
				Timeout: "45m30s",
			},
			defaults: ConfigDefaults{},
			want:     45*time.Minute + 30*time.Second,
			wantErr:  false,
		},
		{
			name: "complex duration",
			list: List{
				Timeout: "2d3h45m30s",
			},
			defaults: ConfigDefaults{},
			want:     51*time.Hour + 45*time.Minute + 30*time.Second,
			wantErr:  false,
		},
		{
			name: "global timeout",
			list: List{},
			defaults: ConfigDefaults{
				Timeout: "1d",
			},
			want:    24 * time.Hour,
			wantErr: false,
		},
		{
			name:     "default timeout",
			list:     List{},
			defaults: ConfigDefaults{},
			want:     0,
			wantErr:  true,
		},
		{
			name: "zero timeout (no timeout)",
			list: List{
				Timeout: "0",
			},
			defaults: ConfigDefaults{},
			want:     0,
			wantErr:  false,
		},
		{
			name: "invalid timeout",
			list: List{
				Timeout: "invalid",
			},
			defaults: ConfigDefaults{},
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.list.GetTimeout(tt.defaults)
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
		name     string
		list     List
		defaults ConfigDefaults
		want     string
	}{
		{
			name: "list prefix",
			list: List{
				CommentPrefix: "list-prefix",
			},
			defaults: ConfigDefaults{
				CommentPrefix: "global-prefix",
			},
			want: "list-prefix",
		},
		{
			name: "global prefix",
			list: List{},
			defaults: ConfigDefaults{
				CommentPrefix: "global-prefix",
			},
			want: "global-prefix",
		},
		{
			name:     "empty prefixes",
			list:     List{},
			defaults: ConfigDefaults{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.list.GetCommentPrefix(tt.defaults)
			if got != tt.want {
				t.Errorf("List.GetCommentPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Config: ConfigDefaults{Timeout: "1d"},
				Lists: map[string]List{
					"test": {
						URLs: []string{"https://example.com"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple sources",
			cfg: &Config{
				Config: ConfigDefaults{Timeout: "1d"},
				Lists: map[string]List{
					"test": {
						URLs:      []string{"https://example.com"},
						Files:     []string{"/path/to/file"},
						Addresses: []string{"192.168.1.1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no lists",
			cfg: &Config{
				Lists: map[string]List{},
			},
			wantErr: true,
		},
		{
			name: "empty list sources",
			cfg: &Config{
				Lists: map[string]List{
					"test": {},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			cfg: &Config{
				Config: ConfigDefaults{
					Timeout: "invalid",
				},
				Lists: map[string]List{
					"test": {
						URLs: []string{"https://example.com"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid timeouts",
			cfg: &Config{
				Config: ConfigDefaults{
					Timeout: "1d",
				},
				Lists: map[string]List{
					"test": {
						Timeout: "12h30m",
						URLs:    []string{"https://example.com"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig_RejectsUnresolvableTimeout(t *testing.T) {
	cfg := &Config{
		Lists: map[string]List{
			"blocklist": {Addresses: []string{"1.2.3.4"}},
		},
	}

	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("ValidateConfig() = nil, want error: no timeout in list or defaults")
	}
}

func TestValidateConfig_RejectsUnsafeListName(t *testing.T) {
	cfg := &Config{
		Config: ConfigDefaults{Timeout: "1d"},
		Lists: map[string]List{
			`evil"; :log info "pwn`: {Addresses: []string{"1.2.3.4"}},
		},
	}

	if err := ValidateConfig(cfg); err == nil {
		t.Fatal("ValidateConfig() = nil, want error for list name with RouterOS metacharacters")
	}
}

func TestValidateConfig_RejectsUnsafeCommentPrefix(t *testing.T) {
	tests := map[string]string{
		"quote":   `bad" ; /system reboot; :local x "`,
		"newline": "bad\nprefix",
		"dollar":  "bad$prefix",
	}

	for name, prefix := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{
				Config: ConfigDefaults{Timeout: "1d"},
				Lists: map[string]List{
					"blocklist": {CommentPrefix: prefix, Addresses: []string{"1.2.3.4"}},
				},
			}

			if err := ValidateConfig(cfg); err == nil {
				t.Fatalf("ValidateConfig() = nil, want error for commentPrefix %q", prefix)
			}
		})
	}
}

func TestValidateConfig_AcceptsRealWorldNamesAndPrefixes(t *testing.T) {
	cfg := &Config{
		Config: ConfigDefaults{Timeout: "4h", CommentPrefix: "Default comment"},
		Lists: map[string]List{
			"rknBlocklist":   {CommentPrefix: "crowdsecurity/external", Addresses: []string{"1.2.3.4"}},
			"ipv6_only_list": {CommentPrefix: "IPv6 addresses", Addresses: []string{"2001:db8::1"}},
			"allow-list":     {Addresses: []string{"10.0.0.0/8"}},
		},
	}

	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig() = %v, want nil", err)
	}
}
