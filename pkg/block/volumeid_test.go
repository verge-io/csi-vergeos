package block

import "testing"

func TestEncodeVolumeID(t *testing.T) {
	id := encodeVolumeID(12345)
	if id != "12345" {
		t.Errorf("got %q, want %q", id, "12345")
	}
}

func TestDecodeVolumeID(t *testing.T) {
	driveID, err := decodeVolumeID("12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if driveID != 12345 {
		t.Errorf("driveID = %d, want 12345", driveID)
	}
}

func TestDecodeVolumeID_Invalid(t *testing.T) {
	_, err := decodeVolumeID("not-a-number")
	if err == nil {
		t.Error("expected error for non-numeric volume ID")
	}
}

func TestDeriveSerial(t *testing.T) {
	serial := deriveSerial("test-pvc-uid-12345")
	if serial == "" {
		t.Error("serial must not be empty")
	}
	if len(serial) > 20 {
		t.Errorf("serial too long: %d chars (max 20)", len(serial))
	}

	// Must be deterministic.
	serial2 := deriveSerial("test-pvc-uid-12345")
	if serial != serial2 {
		t.Error("serial must be deterministic")
	}

	// Different inputs produce different serials.
	serial3 := deriveSerial("different-pvc")
	if serial == serial3 {
		t.Error("different inputs should produce different serials")
	}
}
