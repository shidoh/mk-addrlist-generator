package config

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Config struct {
	Config ConfigDefaults  `yaml:"config"`
	Lists  map[string]List `yaml:"lists"`
}

type ConfigDefaults struct {
	Timeout       string `yaml:"timeout"`
	CommentPrefix string `yaml:"commentPrefix"`
}

type List struct {
	Timeout       string   `yaml:"timeout"`
	CommentPrefix string   `yaml:"commentPrefix"`
	URLs          []string `yaml:"urls,omitempty"`
	Files         []string `yaml:"files,omitempty"`
	Addresses     []string `yaml:"addresses,omitempty"`
}

func (l *List) GetTimeout(defaults ConfigDefaults) (time.Duration, error) {
	timeoutStr := l.Timeout
	if timeoutStr == "" {
		timeoutStr = defaults.Timeout
	}
	if timeoutStr == "" {
		return 0, fmt.Errorf("timeout not specified in list or defaults")
	}

	return parseDuration(timeoutStr)
}

func (l *List) GetCommentPrefix(defaults ConfigDefaults) string {
	if l.CommentPrefix != "" {
		return l.CommentPrefix
	}
	return defaults.CommentPrefix
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Regular expression to match duration components
	re := regexp.MustCompile(`^(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$`)
	matches := re.FindStringSubmatch(s)

	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	var duration time.Duration

	// Parse days
	if matches[1] != "" {
		days, _ := strconv.Atoi(matches[1])
		duration += time.Duration(days) * 24 * time.Hour
	}

	// Parse hours
	if matches[2] != "" {
		hours, _ := strconv.Atoi(matches[2])
		duration += time.Duration(hours) * time.Hour
	}

	// Parse minutes
	if matches[3] != "" {
		minutes, _ := strconv.Atoi(matches[3])
		duration += time.Duration(minutes) * time.Minute
	}

	// Parse seconds
	if matches[4] != "" {
		seconds, _ := strconv.Atoi(matches[4])
		duration += time.Duration(seconds) * time.Second
	}

	// Check if any valid duration component was found
	if duration == 0 {
		return 0, fmt.Errorf("invalid duration: %s (zero duration)", s)
	}

	return duration, nil
}

func ValidateConfig(cfg *Config) error {
	if len(cfg.Lists) == 0 {
		return fmt.Errorf("no lists defined in configuration")
	}

	// Validate global timeout if specified
	if cfg.Config.Timeout != "" {
		if _, err := parseDuration(cfg.Config.Timeout); err != nil {
			return fmt.Errorf("invalid global timeout: %v", err)
		}
	}

	for name, list := range cfg.Lists {
		// Validate list timeout if specified
		if list.Timeout != "" {
			if _, err := parseDuration(list.Timeout); err != nil {
				return fmt.Errorf("invalid timeout in list %s: %v", name, err)
			}
		}

		// Check if at least one source is defined
		if len(list.URLs) == 0 && len(list.Files) == 0 && len(list.Addresses) == 0 {
			return fmt.Errorf("list %s has no sources defined (urls, files, or addresses)", name)
		}
	}

	return nil
}
