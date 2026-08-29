package config

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
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

	// "0" means no timeout (permanent entry)
	if s == "0" {
		return 0, nil
	}

	// Regular expression to match duration components
	re := regexp.MustCompile(`^(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$`)
	matches := re.FindStringSubmatch(s)

	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	var seconds int64
	for _, component := range []struct {
		value       string
		unitSeconds int64
	}{
		{matches[1], 24 * 60 * 60},
		{matches[2], 60 * 60},
		{matches[3], 60},
		{matches[4], 1},
	} {
		var err error
		seconds, err = addDurationComponent(seconds, component.value, component.unitSeconds)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %s: %v", s, err)
		}
	}

	duration := time.Duration(seconds) * time.Second

	// Check if any valid duration component was found
	if duration == 0 {
		return 0, fmt.Errorf("invalid duration: %s (zero duration)", s)
	}

	return duration, nil
}

// maxDurationSeconds is the largest whole number of seconds representable as a
// time.Duration.
const maxDurationSeconds = int64(math.MaxInt64) / int64(time.Second)

// addDurationComponent adds one parsed duration component to a running total of
// seconds, rejecting values that do not fit instead of silently wrapping around
// into a negative duration.
func addDurationComponent(total int64, value string, unitSeconds int64) (int64, error) {
	if value == "" {
		return total, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("component %q is out of range", value)
	}
	if parsed > (maxDurationSeconds-total)/unitSeconds {
		return 0, fmt.Errorf("value exceeds the maximum of %s", time.Duration(math.MaxInt64))
	}

	return total + parsed*unitSeconds, nil
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

	if err := validateCommentPrefix(cfg.Config.CommentPrefix); err != nil {
		return fmt.Errorf("invalid global commentPrefix: %v", err)
	}

	for name, list := range cfg.Lists {
		if !listNamePattern.MatchString(name) {
			return fmt.Errorf("invalid list name %q: names must match %s "+
				"(the name is substituted into RouterOS commands)", name, listNamePattern)
		}

		if err := validateCommentPrefix(list.CommentPrefix); err != nil {
			return fmt.Errorf("invalid commentPrefix in list %s: %v", name, err)
		}

		// Resolve the timeout the same way generation does, so that a config
		// accepted here cannot fail at request time.
		if _, err := list.GetTimeout(cfg.Config); err != nil {
			return fmt.Errorf("invalid timeout in list %s: %v", name, err)
		}

		// Check if at least one source is defined
		if len(list.URLs) == 0 && len(list.Files) == 0 && len(list.Addresses) == 0 {
			return fmt.Errorf("list %s has no sources defined (urls, files, or addresses)", name)
		}
	}

	return nil
}

// listNamePattern restricts list names to characters that are safe to embed in
// the generated RouterOS script, where the name becomes both a quoted argument
// and a global variable name.
var listNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// unsafeCommentChars are characters that would break out of the quoted comment
// argument in the generated RouterOS script.
const unsafeCommentChars = "\"$;{}\\\r\n"

func validateCommentPrefix(prefix string) error {
	if i := strings.IndexAny(prefix, unsafeCommentChars); i >= 0 {
		return fmt.Errorf("character %q is not allowed (it would terminate the RouterOS comment argument)", prefix[i])
	}
	return nil
}
