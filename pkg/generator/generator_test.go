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
		{
			name:  "ipv6 addresses",
			input: "2001:db8::1\n2001:db8:1::/48\nfe80::1",
			want: []string{
				"2001:db8::1",
				"2001:db8:1::/48",
				"fe80::1",
			},
			wantErr: false,
		},
		{
			name:  "mixed ipv4 and ipv6",
			input: "192.168.1.1\n2001:db8::1\n10.0.0.0/24\n2001:db8:1::/48",
			want: []string{
				"192.168.1.1",
				"2001:db8::1",
				"10.0.0.0/24",
				"2001:db8:1::/48",
			},
			wantErr: false,
		},
		{
			name:  "ipv6 with comments",
			input: "2001:db8::1 # IPv6 address\n# Comment\nfe80::1",
			want: []string{
				"2001:db8::1",
				"fe80::1",
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

func TestGenerator_GenerateList_MixedIPv4IPv6(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"mixed": {
				Addresses: []string{
					"192.168.1.1",
					"2001:db8::1",
					"10.0.0.0/24",
					"2001:db8:1::/48",
				},
			},
		},
	}

	g := NewGenerator(cfg)

	script, err := g.GenerateList("mixed", cfg.Lists["mixed"])
	if err != nil {
		t.Fatalf("GenerateList() error = %v", err)
	}

	// Must contain IPv4 section
	if !strings.Contains(script, `/ip/firewall/address-list/remove [ find where list="mixed" ]`) {
		t.Error("script does not contain IPv4 address-list remove command")
	}
	if !strings.Contains(script, `$mixedAddIP "192.168.1.1"`) {
		t.Error("script does not contain IPv4 address entry")
	}
	if !strings.Contains(script, `$mixedAddIP "10.0.0.0/24"`) {
		t.Error("script does not contain IPv4 CIDR entry")
	}

	// Must contain IPv6 section
	if !strings.Contains(script, `/ipv6/firewall/address-list/remove [ find where list="mixed" ]`) {
		t.Error("script does not contain IPv6 address-list remove command")
	}
	if !strings.Contains(script, `$mixedAddIPv6 "2001:db8::1"`) {
		t.Error("script does not contain IPv6 address entry")
	}
	if !strings.Contains(script, `$mixedAddIPv6 "2001:db8:1::/48"`) {
		t.Error("script does not contain IPv6 CIDR entry")
	}
}

func TestGenerator_GenerateList_IPv6Only(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"v6only": {
				Addresses: []string{
					"2001:db8::1",
					"fe80::1",
				},
			},
		},
	}

	g := NewGenerator(cfg)

	script, err := g.GenerateList("v6only", cfg.Lists["v6only"])
	if err != nil {
		t.Fatalf("GenerateList() error = %v", err)
	}

	// Must contain IPv6 section
	if !strings.Contains(script, `/ipv6/firewall/address-list/remove [ find where list="v6only" ]`) {
		t.Error("script does not contain IPv6 address-list remove command")
	}
	if !strings.Contains(script, `$v6onlyAddIPv6 "2001:db8::1"`) {
		t.Error("script does not contain IPv6 entry")
	}

	// Must NOT contain IPv4 section
	if strings.Contains(script, `/ip/firewall/address-list`) {
		t.Error("script should not contain IPv4 address-list commands for IPv6-only config")
	}
}

func TestGenerator_GenerateList_IPv4Only(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"v4only": {
				Addresses: []string{
					"192.168.1.1",
					"10.0.0.0/24",
				},
			},
		},
	}

	g := NewGenerator(cfg)

	script, err := g.GenerateList("v4only", cfg.Lists["v4only"])
	if err != nil {
		t.Fatalf("GenerateList() error = %v", err)
	}

	// Must contain IPv4 section
	if !strings.Contains(script, `/ip/firewall/address-list/remove [ find where list="v4only" ]`) {
		t.Error("script does not contain IPv4 address-list remove command")
	}
	if !strings.Contains(script, `$v4onlyAddIP "192.168.1.1"`) {
		t.Error("script does not contain IPv4 entry")
	}

	// Must NOT contain IPv6 section
	if strings.Contains(script, `/ipv6/firewall/address-list`) {
		t.Error("script should not contain IPv6 address-list commands for IPv4-only config")
	}
}

func TestGenerator_GenerateListWithFormat_Nftables(t *testing.T) {
	cfg := &config.Config{
		Config: config.ConfigDefaults{
			Timeout:       "1d",
			CommentPrefix: "test",
		},
		Lists: map[string]config.List{
			"nft": {
				Addresses: []string{
					"192.168.1.0/24",
					"10.0.0.1",
					"2001:db8::1",
					"2001:db8:1::/48",
				},
			},
		},
	}

	g := NewGenerator(cfg)

	script, err := g.GenerateListWithFormat("nft", cfg.Lists["nft"], FormatNftables)
	if err != nil {
		t.Fatalf("GenerateListWithFormat() error = %v", err)
	}

	// Check IPv4 set
	if !strings.Contains(script, "define nft_v4 = {") {
		t.Error("script does not contain IPv4 nftables set definition")
	}
	if !strings.Contains(script, "192.168.1.0/24") {
		t.Error("script does not contain IPv4 CIDR in v4 set")
	}
	if !strings.Contains(script, "10.0.0.1") {
		t.Error("script does not contain IPv4 address in v4 set")
	}

	// Check IPv6 set
	if !strings.Contains(script, "define nft_v6 = {") {
		t.Error("script does not contain IPv6 nftables set definition")
	}
	if !strings.Contains(script, "2001:db8::1") {
		t.Error("script does not contain IPv6 address in v6 set")
	}
	if !strings.Contains(script, "2001:db8:1::/48") {
		t.Error("script does not contain IPv6 CIDR in v6 set")
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
