package driver

import "testing"

func TestDriverNameForType(t *testing.T) {
	tests := []struct {
		name     string
		dt       DriverType
		expected string
	}{
		{"NAS driver", NASDriver, NASDriverName},
		{"Block driver", BlockDriver, BlockDriverName},
		{"Unknown driver", DriverType("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DriverNameForType(tt.dt)
			if got != tt.expected {
				t.Errorf("DriverNameForType(%q) = %q, want %q", tt.dt, got, tt.expected)
			}
		})
	}
}

func TestDriverConstants(t *testing.T) {
	if NASDriverName == "" {
		t.Error("NASDriverName must not be empty")
	}
	if BlockDriverName == "" {
		t.Error("BlockDriverName must not be empty")
	}
	if NASDriverName == BlockDriverName {
		t.Error("NAS and Block driver names must be different")
	}
	if DriverVersion == "" {
		t.Error("DriverVersion must not be empty")
	}
}
