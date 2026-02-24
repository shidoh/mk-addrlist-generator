package config

import "time"

// Config represents the root configuration structure
type Config struct {
	Config GlobalConfig     `yaml:"config"`
	Lists  map[string]List `yaml:"lists"`
}

// GlobalConfig represents global configuration parameters
type GlobalConfig struct {
	Timeout       string `yaml:"timeout"`
	CommentPrefix string `yaml:"commentPrefix"`
}

// List represents a single address list configuration
type List struct {
	Timeout       string   `yaml:"timeout"`
	CommentPrefix string   `yaml:"commentPrefix"`
	URLs          []string `yaml:"urls,omitempty"`
	Files         []string `yaml:"files,omitempty"`
	Addresses     []string `yaml:"addresses,omitempty"`
}

// GetTimeout returns the effective timeout for a list
func (l *List) GetTimeout(globalTimeout string) (time.Duration, error) {
	if l.Timeout != "" {
		return time.ParseDuration(l.Timeout)
	}
	if globalTimeout != "" {
		return time.ParseDuration(globalTimeout)
	}
	return time.Hour, nil // default 1 hour timeout
}

// GetCommentPrefix returns the effective comment prefix for a list
func (l *List) GetCommentPrefix(globalPrefix string) string {
	if l.CommentPrefix != "" {
		return l.CommentPrefix
	}
	return globalPrefix
}

// GetType returns the type of the list based on its configuration
func (l *List) GetType() string {
	switch {
	case len(l.URLs) > 0:
		return "external"
	case len(l.Files) > 0:
		return "files"
	case len(l.Addresses) > 0:
		return "static"
	default:
		return "unknown"
	}
}