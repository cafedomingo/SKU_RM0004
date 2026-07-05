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

func (f *fakeI2C) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	f.writes = append(f.writes, cp)
	return len(p), nil
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
	var fb Framebuffer
	d.SendRegion(Region{X: 0, Y: 10, W: Width, H: 2}, &fb)

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

	// Two rows × 160 pixels × 2 bytes = 640 bytes → exactly 4 burst chunks.
	wantBurstCount := 4
	if got, want := len(fake.writes), len(want)+wantBurstCount+2; got != want {
		t.Fatalf("got %d writes, want %d", got, want)
	}

	// Pixel bursts: every chunk must respect the hardware limit and the
	// total must equal the region's byte size.
	burst := fake.writes[len(want) : len(want)+wantBurstCount]
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
	wantTail := [][]byte{
		{regBurstWrite, 0x00, 0x00},
		{regSync, 0x00, 0x01},
	}
	last := fake.writes[len(want)+wantBurstCount:]
	for i, w := range wantTail {
		if got := last[i]; len(got) != len(w) || got[0] != w[0] || got[1] != w[1] || got[2] != w[2] {
			t.Errorf("trailing write[%d] = %v, want %v", i, got, w)
		}
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

func TestRegionToBytes(t *testing.T) {
	var fb Framebuffer
	fb.SetPixel(3, 5, 0xF800)
	fb.SetPixel(4, 5, 0x07E0)
	fb.SetPixel(3, 6, 0xFFFF)
	fb.SetPixel(4, 6, 0x0000)
	// Pixel outside the region on the same rows: must not be serialized.
	fb.SetPixel(5, 5, 0x1234)

	// A 2x2 region not touching the framebuffer edge exercises the row
	// striding: source rows are 160 pixels apart, output is contiguous.
	got := regionToBytes(Region{X: 3, Y: 5, W: 2, H: 2}, &fb)
	want := []byte{
		0xF8, 0x00, 0x07, 0xE0, // row 5: big-endian RGB565
		0xFF, 0xFF, 0x00, 0x00, // row 6
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("byte[%d] = 0x%02X, want 0x%02X", i, got[i], want[i])
		}
	}
}
