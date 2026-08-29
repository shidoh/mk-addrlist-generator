// Package netutil provides utilities for IP address handling,
// including deduplication and CIDR aggregation.
package netutil

import (
	"net"
	"sort"
)

// IPNet wraps net.IPNet with additional methods for comparison and sorting.
type IPNet struct {
	*net.IPNet
}

// ParseIPOrCIDR parses a string as either an IP address or CIDR notation.
// Single IPs are converted to /32 (IPv4) or /128 (IPv6) networks.
func ParseIPOrCIDR(s string) (*IPNet, error) {
	// Try parsing as CIDR first
	_, ipnet, err := net.ParseCIDR(s)
	if err == nil {
		return &IPNet{IPNet: ipnet}, nil
	}

	// Try parsing as single IP
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, &net.ParseError{Type: "IP address or CIDR", Text: s}
	}

	// Convert single IP to CIDR
	if ip4 := ip.To4(); ip4 != nil {
		return &IPNet{IPNet: &net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)}}, nil
	}
	return &IPNet{IPNet: &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}}, nil
}

// IsIPv4 returns true if the network is IPv4.
func (n *IPNet) IsIPv4() bool {
	return n.IP.To4() != nil
}

// Compare compares two IPNets for sorting.
// Returns -1 if n < other, 0 if equal, 1 if n > other.
func (n *IPNet) Compare(other *IPNet) int {
	// Compare IPs first
	for i := 0; i < len(n.IP) && i < len(other.IP); i++ {
		if n.IP[i] < other.IP[i] {
			return -1
		}
		if n.IP[i] > other.IP[i] {
			return 1
		}
	}

	// If IPs are equal, compare mask lengths (smaller mask = larger network)
	ones1, _ := n.Mask.Size()
	ones2, _ := other.Mask.Size()
	if ones1 < ones2 {
		return -1
	}
	if ones1 > ones2 {
		return 1
	}

	return 0
}

// Contains returns true if n fully contains other.
func (n *IPNet) Contains(other *IPNet) bool {
	// n must contain all IPs in other
	ones1, bits1 := n.Mask.Size()
	ones2, bits2 := other.Mask.Size()

	// Different address families
	if bits1 != bits2 {
		return false
	}

	// n's mask must be smaller or equal (larger network)
	if ones1 > ones2 {
		return false
	}

	return n.IPNet.Contains(other.IP)
}

// Deduplicate removes duplicate networks from a slice.
// Networks are considered duplicates if they have the same string representation.
func Deduplicate(networks []*IPNet) []*IPNet {
	seen := make(map[string]struct{})
	result := make([]*IPNet, 0, len(networks))

	for _, n := range networks {
		key := n.String()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, n)
		}
	}

	return result
}

// DeduplicateStrings removes duplicate strings from a slice while preserving order.
func DeduplicateStrings(items []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(items))

	for _, item := range items {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

// Aggregate merges overlapping and adjacent CIDR networks.
// This implements CIDR aggregation (also known as supernetting).
func Aggregate(networks []*IPNet) []*IPNet {
	if len(networks) == 0 {
		return networks
	}

	// Separate IPv4 and IPv6 networks
	var ipv4, ipv6 []*IPNet
	for _, n := range networks {
		if n.IsIPv4() {
			// Normalize to 4-byte representation
			n.IP = n.IP.To4()
			ipv4 = append(ipv4, n)
		} else {
			ipv6 = append(ipv6, n)
		}
	}

	// Aggregate each family separately
	result := aggregateFamily(ipv4)
	result = append(result, aggregateFamily(ipv6)...)

	return result
}

// aggregateFamily aggregates networks of the same IP family.
func aggregateFamily(networks []*IPNet) []*IPNet {
	if len(networks) == 0 {
		return networks
	}

	// Sort networks by IP and mask
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Compare(networks[j]) < 0
	})

	// Remove networks that are contained within others
	networks = removeContained(networks)

	// Merge adjacent networks
	for {
		merged := mergeAdjacent(networks)
		if len(merged) == len(networks) {
			break
		}
		networks = merged
	}

	return networks
}

// removeContained removes networks that are fully contained within other networks.
func removeContained(networks []*IPNet) []*IPNet {
	if len(networks) <= 1 {
		return networks
	}

	result := make([]*IPNet, 0, len(networks))

	for i, n := range networks {
		contained := false
		for j, other := range networks {
			if i == j || !other.Contains(n) {
				continue
			}
			// Identical networks contain each other. Without this check both
			// copies would consider themselves contained and both would be
			// dropped, silently removing the address from the list.
			if j > i && n.Contains(other) {
				continue
			}
			contained = true
			break
		}
		if !contained {
			result = append(result, n)
		}
	}

	return result
}

// mergeAdjacent merges adjacent networks that can be combined into a larger one.
func mergeAdjacent(networks []*IPNet) []*IPNet {
	if len(networks) <= 1 {
		return networks
	}

	result := make([]*IPNet, 0, len(networks))
	used := make([]bool, len(networks))

	for i := 0; i < len(networks); i++ {
		if used[i] {
			continue
		}

		merged := false
		for j := i + 1; j < len(networks); j++ {
			if used[j] {
				continue
			}

			combined := tryCombine(networks[i], networks[j])
			if combined != nil {
				result = append(result, combined)
				used[i] = true
				used[j] = true
				merged = true
				break
			}
		}

		if !merged {
			result = append(result, networks[i])
		}
	}

	return result
}

// tryCombine attempts to combine two adjacent networks into one.
// Returns nil if they cannot be combined.
func tryCombine(a, b *IPNet) *IPNet {
	// Networks must have the same mask length
	ones1, bits1 := a.Mask.Size()
	ones2, bits2 := b.Mask.Size()

	if ones1 != ones2 || bits1 != bits2 {
		return nil
	}

	// Cannot combine /0 networks
	if ones1 == 0 {
		return nil
	}

	// Create the potential parent network (one bit less specific)
	parentMask := net.CIDRMask(ones1-1, bits1)
	parentA := net.IPNet{IP: a.IP.Mask(parentMask), Mask: parentMask}
	parentB := net.IPNet{IP: b.IP.Mask(parentMask), Mask: parentMask}

	// If both networks have the same parent, they can be combined
	if parentA.IP.Equal(parentB.IP) {
		return &IPNet{IPNet: &parentA}
	}

	return nil
}

// ParseAndAggregate parses a list of IP/CIDR strings, deduplicates and aggregates them.
func ParseAndAggregate(addresses []string) ([]string, error) {
	networks := make([]*IPNet, 0, len(addresses))

	for _, addr := range addresses {
		n, err := ParseIPOrCIDR(addr)
		if err != nil {
			return nil, err
		}
		networks = append(networks, n)
	}

	// Deduplicate first
	networks = Deduplicate(networks)

	// Then aggregate
	networks = Aggregate(networks)

	// Convert back to strings
	result := make([]string, len(networks))
	for i, n := range networks {
		result[i] = n.String()
	}

	return result, nil
}
