package st7735

import (
	"bytes"
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
	if total != 2*Width*bytesPerPixel {
		t.Errorf("burst total = %d bytes, want %d", total, 2*Width*bytesPerPixel)
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

// A partial rectangle (non-zero X, non-full W) through the real send path:
// window bytes, payload size, short final chunk, and row-strided order.
func TestSendRegionPartialFraming(t *testing.T) {
	fake := &fakeI2C{}
	d := &display{dev: fake, logger: slog.Default()}

	var fb Framebuffer
	fb.SetPixel(82, 37, 0xAAAA)  // region's first pixel
	fb.SetPixel(159, 37, 0xBBBB) // end of first region row
	fb.SetPixel(82, 38, 0xCCCC)  // start of second region row
	fb.SetPixel(81, 37, 0xDDDD)  // left of region, must not appear

	r := Region{X: 82, Y: 37, W: 78, H: 18}
	d.SendRegion(r, &fb)

	if got, want := fake.writes[0], []byte{regXCoord, 82, 159}; !bytes.Equal(got, want) {
		t.Errorf("column window = %v, want %v", got, want)
	}
	if got, want := fake.writes[1], []byte{regYCoord, 37 + yOffset, 54 + yOffset}; !bytes.Equal(got, want) {
		t.Errorf("row window = %v, want %v", got, want)
	}

	// 78*18*2 = 2808 payload bytes: 17 full chunks and an 88-byte final chunk.
	var payload []byte
	for _, w := range fake.writes[5 : len(fake.writes)-2] {
		if len(w) > burstMaxLen {
			t.Errorf("chunk of %d bytes exceeds burstMaxLen", len(w))
		}
		payload = append(payload, w...)
	}
	if len(payload) != r.W*r.H*bytesPerPixel {
		t.Fatalf("payload = %d bytes, want %d", len(payload), r.W*r.H*bytesPerPixel)
	}
	if got := len(fake.writes[len(fake.writes)-3]); got != 88 {
		t.Errorf("final chunk = %d bytes, want 88", got)
	}

	// Row striding: byte offsets within the payload are region-relative.
	checks := []struct {
		off  int
		want [2]byte
	}{
		{0, [2]byte{0xAA, 0xAA}},                    // (82,37)
		{77 * bytesPerPixel, [2]byte{0xBB, 0xBB}},   // (159,37)
		{r.W * bytesPerPixel, [2]byte{0xCC, 0xCC}},  // (82,38)
	}
	for _, c := range checks {
		if payload[c.off] != c.want[0] || payload[c.off+1] != c.want[1] {
			t.Errorf("payload[%d:%d] = %02X%02X, want %02X%02X",
				c.off, c.off+2, payload[c.off], payload[c.off+1], c.want[0], c.want[1])
		}
	}
	for i := 0; i < len(payload); i += 2 {
		if payload[i] == 0xDD {
			t.Errorf("pixel left of region leaked into payload at offset %d", i)
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
	fb.SetPixel(5, 5, 0x1234) // outside the region, must not be serialized

	// A 2x2 region away from the framebuffer edge exercises row striding.
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
