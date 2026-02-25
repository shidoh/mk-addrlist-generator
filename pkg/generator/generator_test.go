package generator

import (
	"mk-addrlist-generator/pkg/config"
	"strings"
	"testing"
)

func TestGenerator_GenerateList(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"test": {
				Addresses: []string{
					"192.168.1.1",
					"10.0.0.0/24",
				},
			},
		},
	}

	g := NewGenerator(cfg)

	script, err := g.GenerateList("test", cfg.Lists["test"])
	if err != nil {
		t.Fatalf("GenerateList() error = %v", err)
	}

	// Check script content
	expectedLines := []string{
		`/ip/firewall/address-list/remove [ find where list="test" ];`,
		`:global testAddIP;`,
		`:set testAddIP do={`,
		`:do { /ip/firewall/address-list/add list=test address=$1 comment="$2" timeout=$3; } on-error={ }`,
		`}`,
		`$testAddIP "192.168.1.1" "test/static" "24h0m0s"`,
		`$testAddIP "10.0.0.0/24" "test/static" "24h0m0s"`,
		`:set testAddIP;`,
	}

	for _, line := range expectedLines {
		if !strings.Contains(script, line) {
			t.Errorf("GenerateList() script does not contain expected line: %s", line)
		}
	}
}

func TestGenerator_GenerateAll(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"list1": {
				Addresses: []string{
					"192.168.1.1",
				},
			},
			"list2": {
				Addresses: []string{
					"10.0.0.0/24",
				},
			},
		},
	}

	g := NewGenerator(cfg)

	script, err := g.GenerateAll()
	if err != nil {
		t.Fatalf("GenerateAll() error = %v", err)
	}

	// Check script content
	for name := range cfg.Lists {
		if !strings.Contains(script, name) {
			t.Errorf("GenerateAll() script does not contain list: %s", name)
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
			name:  "simple addresses",
			input: "192.168.1.1\n10.0.0.0/24",
			want: []string{
				"192.168.1.1",
				"10.0.0.0/24",
			},
			wantErr: false,
		},
		{
			name:  "with comments",
			input: "192.168.1.1 # First address\n# Comment line\n10.0.0.0/24",
			want: []string{
				"192.168.1.1",
				"10.0.0.0/24",
			},
			wantErr: false,
		},
		{
			name:  "empty lines",
			input: "\n192.168.1.1\n\n10.0.0.0/24\n",
			want: []string{
				"192.168.1.1",
				"10.0.0.0/24",
			},
			wantErr: false,
		},
		{
			name:  "whitespace",
			input: "  192.168.1.1  \n  10.0.0.0/24  ",
			want: []string{
				"192.168.1.1",
				"10.0.0.0/24",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			got, err := readAddresses(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !stringSliceEqual(got, tt.want) {
				t.Errorf("readAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func stringSliceEqual(a, b []string) bool {
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
