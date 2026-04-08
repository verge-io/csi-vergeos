package nas

import "testing"

func TestEncodeVolumeID(t *testing.T) {
	id := encodeVolumeID(42, "abc123", "def456")
	if id != "42:abc123:def456" {
		t.Errorf("got %q, want %q", id, "42:abc123:def456")
	}
}

func TestDecodeVolumeID(t *testing.T) {
	nasID, volID, shareID, err := decodeVolumeID("42:abc123:def456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nasID != 42 {
		t.Errorf("nasServiceID = %d, want 42", nasID)
	}
	if volID != "abc123" {
		t.Errorf("volumeID = %q, want %q", volID, "abc123")
	}
	if shareID != "def456" {
		t.Errorf("nfsShareID = %q, want %q", shareID, "def456")
	}
}

func TestDecodeVolumeID_Invalid(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"one part", "abc"},
		{"two parts", "42:abc"},
		{"non-integer NAS ID", "abc:def:ghi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := decodeVolumeID(tt.id)
			if err == nil {
				t.Error("expected error for invalid volume ID")
			}
		})
	}
}
