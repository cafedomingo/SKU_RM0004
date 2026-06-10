package st7735

import (
	"log/slog"
	"testing"
)

// fakeI2C records every write so tests can check protocol framing.
type fakeI2C struct {
	writes [][]byte
	closed bool
}

func (f *fakeI2C) write(p []byte) error {
	cp := make([]byte, len(p))
	copy(cp, p)
	f.writes = append(f.writes, cp)
	return nil
}

func (f *fakeI2C) Close() error {
	f.closed = true
	return nil
}

func TestSendRegionFraming(t *testing.T) {
	fake := &fakeI2C{}
	d := &display{dev: fake, logger: slog.Default()}

	// Two rows of 160 pixels starting at y=10: 640 bytes of pixel data,
	// which must be split into 160-byte burst chunks.
	pixels := make([]uint16, 2*Width)
	d.SendRegion(0, 10, Width, 2, pixels)

	want := [][]byte{
		{regXCoord, 0, Width - 1},               // column window
		{regYCoord, 10 + yOffset, 11 + yOffset}, // row window with panel offset
		{regCharData, 0x00, 0x00},
		{regSync, 0x00, 0x01},
		{regBurstWrite, 0x00, 0x01}, // burst on
	}
	if len(fake.writes) < len(want) {
		t.Fatalf("got %d writes, want at least %d", len(fake.writes), len(want))
	}
	for i, w := range want {
		got := fake.writes[i]
		if len(got) != len(w) {
			t.Fatalf("write[%d] = %v, want %v", i, got, w)
		}
		for j := range w {
			if got[j] != w[j] {
				t.Errorf("write[%d][%d] = 0x%02X, want 0x%02X", i, j, got[j], w[j])
			}
		}
	}

	// Pixel bursts: every chunk must respect the hardware limit and the
	// total must equal the region's byte size.
	burst := fake.writes[len(want) : len(fake.writes)-2]
	total := 0
	for i, chunk := range burst {
		if len(chunk) > burstMaxLen {
			t.Errorf("burst chunk %d is %d bytes, exceeds hardware limit %d", i, len(chunk), burstMaxLen)
		}
		total += len(chunk)
	}
	if total != 2*Width*2 {
		t.Errorf("burst total = %d bytes, want %d", total, 2*Width*2)
	}

	// Trailing writes: burst off, then sync.
	last := fake.writes[len(fake.writes)-2:]
	if last[0][0] != regBurstWrite || last[0][2] != 0x00 {
		t.Errorf("expected burst-off command, got %v", last[0])
	}
	if last[1][0] != regSync {
		t.Errorf("expected sync command, got %v", last[1])
	}
}

func TestDisplayClose(t *testing.T) {
	fake := &fakeI2C{}
	d := &display{dev: fake, logger: slog.Default()}
	if err := d.Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if !fake.closed {
		t.Error("Close() did not close the underlying bus")
	}
}

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
