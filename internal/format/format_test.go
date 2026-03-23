package format_test

import (
	"testing"
	"time"

	"github.com/cafedomingo/SKU_RM0004/internal/format"
)

func TestRate(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0B"},
		{1, "1B"},
		{500, "500B"},
		{1023, "1023B"},
		{1024, "1.0K"},
		{5120, "5.0K"},
		{10240, "10K"},
		{1048575, "1023K"},
		{1048576, "1.0M"},
		{10485760, "10M"},
		{104857600, "100M"},
	}
	for _, tt := range tests {
		got := format.Rate(tt.input)
		if got != tt.want {
			t.Errorf("Rate(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFreq(t *testing.T) {
	tests := []struct {
		input uint16
		want  string
	}{
		{0, "0MHz"},
		{600, "600MHz"},
		{999, "999MHz"},
		{1000, "1.0GHz"},
		{1800, "1.8GHz"},
		{2400, "2.4GHz"},
	}
	for _, tt := range tests {
		got := format.Freq(tt.input)
		if got != tt.want {
			t.Errorf("Freq(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUptime(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{0 * time.Second, "0m"},
		{59 * time.Second, "0m"},
		{60 * time.Second, "1m"},
		{120 * time.Second, "2m"},
		{3600 * time.Second, "1h 0m"},
		{3700 * time.Second, "1h 1m"},
		{86400 * time.Second, "1d 0h"},
		{90061 * time.Second, "1d 1h"},
	}
	for _, tt := range tests {
		got := format.Uptime(tt.input)
		if got != tt.want {
			t.Errorf("Uptime(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTemp(t *testing.T) {
	tests := []struct {
		celsius float64
		unit    string
		want    string
	}{
		{52, "C", "52°C"},
		{0, "C", " 0°C"},
		{100, "C", "100°C"},
		{52, "F", "125°F"},
	}
	for _, tt := range tests {
		got := format.Temp(tt.celsius, tt.unit)
		if got != tt.want {
			t.Errorf("Temp(%v, %q) = %q, want %q", tt.celsius, tt.unit, got, tt.want)
		}
	}
}

func TestCelsiusToF(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{0, 32},
		{100, 212},
		{50, 122},
	}
	for _, tt := range tests {
		got := format.CelsiusToF(tt.input)
		if got != tt.want {
			t.Errorf("CelsiusToF(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestAPTBadge(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, ""},
		{-1, ""},
		{1, "^1"},
		{3, "^3"},
		{99, "^99"},
		{100, "^99"},
	}
	for _, tt := range tests {
		got := format.APTBadge(tt.input)
		if got != tt.want {
			t.Errorf("APTBadge(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
