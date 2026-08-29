package config

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "days only",
			input:   "2d",
			want:    48 * time.Hour,
			wantErr: false,
		},
		{
			name:    "days and hours",
			input:   "1d12h",
			want:    36 * time.Hour,
			wantErr: false,
		},
		{
			name:    "hours and minutes",
			input:   "12h30m",
			want:    12*time.Hour + 30*time.Minute,
			wantErr: false,
		},
		{
			name:    "minutes and seconds",
			input:   "45m30s",
			want:    45*time.Minute + 30*time.Second,
			wantErr: false,
		},
		{
			name:    "complex duration",
			input:   "2d3h45m30s",
			want:    51*time.Hour + 45*time.Minute + 30*time.Second,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "missing values",
			input:   "dhs",
			want:    0,
			wantErr: true,
		},
		{
			name:    "zero duration",
			input:   "0d0h0m0s",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDuration_RejectsOutOfRangeComponents(t *testing.T) {
	tests := []string{
		"1000000000d",
		"99999999999999999999d",
		"9223372036854775807s",
	}

	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			got, err := parseDuration(s)
			if err == nil {
				t.Fatalf("parseDuration(%q) = %v, want error", s, got)
			}
		})
	}
}

func TestParseDuration_NeverReturnsNegative(t *testing.T) {
	for _, s := range []string{"1000000000d", "9223372036854775807s", "1d", "12h30m"} {
		got, err := parseDuration(s)
		if err == nil && got < 0 {
			t.Errorf("parseDuration(%q) = %v, want non-negative duration", s, got)
		}
	}
}
