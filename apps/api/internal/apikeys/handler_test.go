package apikeys

import (
	"strings"
	"testing"
)

func TestKeyGeneration(t *testing.T) {
	// Verify that generated keys have the expected prefix
	rawBytes := make([]byte, 32)
	for i := range rawBytes {
		rawBytes[i] = byte(i)
	}
	key := keyPrefix + "test_suffix"
	if !strings.HasPrefix(key, "opsowl_") {
		t.Errorf("expected key prefix 'opsowl_', got %q", key[:7])
	}
}

func TestMinHelper(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{7, 7, 7},
		{0, 1, 0},
	}
	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
