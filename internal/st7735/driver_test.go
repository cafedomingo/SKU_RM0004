package st7735

import (
	"testing"
)

func TestPixelsToBytes(t *testing.T) {
	tests := []struct {
		name   string
		pixels []uint16
		want   []byte
	}{
		{"empty", nil, []byte{}},
		{"single pixel", []uint16{0xF800}, []byte{0xF8, 0x00}},
		{"two pixels", []uint16{0xF800, 0x07E0}, []byte{0xF8, 0x00, 0x07, 0xE0}},
		{"zero pixel", []uint16{0x0000}, []byte{0x00, 0x00}},
		{"max pixel", []uint16{0xFFFF}, []byte{0xFF, 0xFF}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pixelsToBytes(tt.pixels)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("byte[%d] = 0x%02X, want 0x%02X", i, got[i], tt.want[i])
				}
			}
		})
	}
}
