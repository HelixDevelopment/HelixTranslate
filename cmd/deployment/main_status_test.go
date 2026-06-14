package main

import "testing"

// shortContainerID must never panic on container IDs shorter than 12 bytes.
// The original `status` action sliced instance.ContainerID[:12] unconditionally,
// which panics with "slice bounds out of range" for an empty or short ID (a
// short ID is possible: it can be a deployer return value or a short configured
// container name) and crashes the whole status report.
func TestShortContainerID_NoPanicOnShortID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"short", "abc", "abc"},
		{"exactly 11", "0123456789a", "0123456789a"},
		{"exactly 12", "0123456789ab", "0123456789ab"},
		{"long 13", "0123456789abc", "0123456789ab"},
		{"long docker id", "0123456789abcdef0123456789abcdef", "0123456789ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("shortContainerID(%q) panicked: %v", tt.in, r)
				}
			}()
			if got := shortContainerID(tt.in); got != tt.want {
				t.Fatalf("shortContainerID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
