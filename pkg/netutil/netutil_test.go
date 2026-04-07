package netutil

import (
	"reflect"
	"sort"
	"testing"
)

func TestParseIPOrCIDR(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "IPv4 single address",
			input:   "192.168.1.1",
			want:    "192.168.1.1/32",
			wantErr: false,
		},
		{
			name:    "IPv4 CIDR",
			input:   "192.168.1.0/24",
			want:    "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "IPv6 single address",
			input:   "2001:db8::1",
			want:    "2001:db8::1/128",
			wantErr: false,
		},
		{
			name:    "IPv6 CIDR",
			input:   "2001:db8::/32",
			want:    "2001:db8::/32",
			wantErr: false,
		},
		{
			name:    "invalid address",
			input:   "invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIPOrCIDR(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIPOrCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("ParseIPOrCIDR() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no duplicates",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "with duplicates",
			input: []string{"a", "b", "a", "c", "b"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "all same",
			input: []string{"a", "a", "a"},
			want:  []string{"a"},
		},
		{
			name:  "empty",
			input: []string{},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeduplicateStrings(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeduplicateStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAggregate(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "two adjacent /24 networks",
			input: []string{"192.168.0.0/24", "192.168.1.0/24"},
			want:  []string{"192.168.0.0/23"},
		},
		{
			name:  "contained network removed",
			input: []string{"10.0.0.0/8", "10.1.0.0/16"},
			want:  []string{"10.0.0.0/8"},
		},
		{
			name:  "non-adjacent networks unchanged",
			input: []string{"192.168.0.0/24", "192.168.2.0/24"},
			want:  []string{"192.168.0.0/24", "192.168.2.0/24"},
		},
		{
			name:  "multiple aggregations",
			input: []string{"192.168.0.0/24", "192.168.1.0/24", "192.168.2.0/24", "192.168.3.0/24"},
			want:  []string{"192.168.0.0/22"},
		},
		{
			name:  "single IPs to aggregate",
			input: []string{"192.168.1.0", "192.168.1.1"},
			want:  []string{"192.168.1.0/31"},
		},
		{
			name:  "duplicate networks",
			input: []string{"192.168.1.0/24", "192.168.1.0/24"},
			want:  []string{"192.168.1.0/24"},
		},
		{
			name:  "two adjacent IPv6 /64 networks",
			input: []string{"2001:db8::/64", "2001:db8:0:1::/64"},
			want:  []string{"2001:db8::/63"},
		},
		{
			name:  "contained IPv6 network removed",
			input: []string{"2001:db8::/32", "2001:db8:1::/48"},
			want:  []string{"2001:db8::/32"},
		},
		{
			name:  "mixed IPv4 and IPv6 aggregated independently",
			input: []string{"192.168.0.0/24", "192.168.1.0/24", "2001:db8::/64", "2001:db8:0:1::/64"},
			want:  []string{"192.168.0.0/23", "2001:db8::/63"},
		},
		{
			name:  "non-adjacent IPv6 networks unchanged",
			input: []string{"2001:db8::/48", "2001:db8:2::/48"},
			want:  []string{"2001:db8::/48", "2001:db8:2::/48"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAndAggregate(tt.input)
			if err != nil {
				t.Errorf("ParseAndAggregate() error = %v", err)
				return
			}

			// Sort both for comparison
			sort.Strings(got)
			sort.Strings(tt.want)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAndAggregate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPNet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		parent   string
		child    string
		contains bool
	}{
		{
			name:     "parent contains child",
			parent:   "10.0.0.0/8",
			child:    "10.1.0.0/16",
			contains: true,
		},
		{
			name:     "same network",
			parent:   "10.0.0.0/8",
			child:    "10.0.0.0/8",
			contains: true,
		},
		{
			name:     "child larger than parent",
			parent:   "10.1.0.0/16",
			child:    "10.0.0.0/8",
			contains: false,
		},
		{
			name:     "different networks",
			parent:   "192.168.0.0/24",
			child:    "10.0.0.0/24",
			contains: false,
		},
		{
			name:     "IPv6 parent contains child",
			parent:   "2001:db8::/32",
			child:    "2001:db8:1::/48",
			contains: true,
		},
		{
			name:     "IPv6 same network",
			parent:   "2001:db8::/32",
			child:    "2001:db8::/32",
			contains: true,
		},
		{
			name:     "IPv6 child larger than parent",
			parent:   "2001:db8:1::/48",
			child:    "2001:db8::/32",
			contains: false,
		},
		{
			name:     "IPv6 different networks",
			parent:   "2001:db8::/32",
			child:    "2001:db9::/32",
			contains: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, _ := ParseIPOrCIDR(tt.parent)
			child, _ := ParseIPOrCIDR(tt.child)

			if got := parent.Contains(child); got != tt.contains {
				t.Errorf("Contains() = %v, want %v", got, tt.contains)
			}
		})
	}
}
