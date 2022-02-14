package main

import "testing"

func TestByteCounter_String(t *testing.T) {
	tests := []struct {
		name string
		b    ByteCounter
		want string
	}{
		{"0B", 0, "0.0B"},
		{"500B", 500, "0.5KB"},
		{"1K", 1000, "1.0KB"},
		{"2MB", 2000 * 1000, "2.0MB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.b.String(); got != tt.want {
				t.Errorf("ByteCounter.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
