package randutil

import (
	"testing"
)

func TestNumeric(t *testing.T) {
	s, err := Numeric(12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 12 {
		t.Errorf("expected length 12, got %d (%s)", len(s), s)
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Errorf("expected digit character, got %c", c)
		}
	}

	// Zero length
	s0, err0 := Numeric(0)
	if err0 != nil {
		t.Fatalf("unexpected error for 0 length: %v", err0)
	}
	if s0 != "" {
		t.Errorf("expected empty string for length 0, got %s", s0)
	}
}
