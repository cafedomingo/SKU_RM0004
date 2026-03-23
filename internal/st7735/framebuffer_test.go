package st7735

import (
	"testing"

	"github.com/cafedomingo/SKU_RM0004/internal/font"
)

func TestFill(t *testing.T) {
	var fb Framebuffer
	const color uint16 = 0xF800 // red in RGB565
	fb.Fill(color)
	for i, px := range fb.Pixels {
		if px != color {
			t.Fatalf("pixel %d: got 0x%04x, want 0x%04x", i, px, color)
		}
	}
}

func TestSetPixel(t *testing.T) {
	tests := []struct {
		name  string
		x, y  int
		color uint16
	}{
		{"origin", 0, 0, 0xFFFF},
		{"bottom-right", 159, 79, 0x001F},
		{"center", 80, 40, 0x07E0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fb Framebuffer
			fb.SetPixel(tt.x, tt.y, tt.color)
			idx := tt.y*Width + tt.x
			if fb.Pixels[idx] != tt.color {
				t.Errorf("pixel at (%d,%d): got 0x%04x, want 0x%04x", tt.x, tt.y, fb.Pixels[idx], tt.color)
			}
		})
	}
}

func TestSetPixelOutOfBounds(t *testing.T) {
	tests := []struct {
		name string
		x, y int
	}{
		{"neg-x", -1, 0},
		{"x-at-width", 160, 0},
		{"y-at-height", 0, 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fb Framebuffer
			// Fill with known value so we can detect any change
			fb.Fill(0x1234)
			// Must not panic, must not change any pixel
			fb.SetPixel(tt.x, tt.y, 0xFFFF)
			for i, px := range fb.Pixels {
				if px != 0x1234 {
					t.Fatalf("pixel %d changed after out-of-bounds SetPixel(%d,%d)", i, tt.x, tt.y)
				}
			}
		})
	}
}

func TestRect(t *testing.T) {
	var fb Framebuffer
	const color uint16 = 0xF800
	const bg uint16 = 0x0000
	// rect at (20,30) size 10x5
	fb.Rect(20, 30, 10, 5, color)

	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			px := fb.Pixels[y*Width+x]
			inRect := x >= 20 && x < 30 && y >= 30 && y < 35
			if inRect {
				if px != color {
					t.Errorf("(%d,%d) inside rect: got 0x%04x, want 0x%04x", x, y, px, color)
				}
			} else {
				if px != bg {
					t.Errorf("(%d,%d) outside rect: got 0x%04x, want 0x%04x", x, y, px, bg)
				}
			}
		}
	}
}

// 'A' glyph from Spleen8x16:
// row 0: 0x00
// row 1: 0x00
// row 2: 0x7c = 0111 1100 — pixels 1-6 set
// row 3: 0xc6 = 1100 0110 — pixels 0,1,5,6 set
// row 4: 0xc6
// row 5: 0xc6
// row 6: 0xfe = 1111 1110 — pixels 0-6 set
// row 7: 0xc6
// ...
func TestChar(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xFFFF
	fb.Char(0, 0, 'A', font.Spleen8x16, fg)

	// Row 2, col 1: bit 6 of 0x7c (0111 1100) is set
	if fb.Pixels[2*Width+1] != fg {
		t.Errorf("row2,col1 should be fg")
	}
	// Row 2, col 0: bit 7 of 0x7c is 0 — should be unset (still 0)
	if fb.Pixels[2*Width+0] != 0 {
		t.Errorf("row2,col0 should be background (0), got 0x%04x", fb.Pixels[2*Width+0])
	}
	// Row 6 (0xfe = 1111 1110): cols 0-6 set, col 7 not set
	if fb.Pixels[6*Width+0] != fg {
		t.Errorf("row6,col0 should be fg")
	}
	if fb.Pixels[6*Width+7] != 0 {
		t.Errorf("row6,col7 should be background (0), got 0x%04x", fb.Pixels[6*Width+7])
	}
	// Row 0 (0x00): all cols should be untouched (0)
	for col := 0; col < 8; col++ {
		if fb.Pixels[0*Width+col] != 0 {
			t.Errorf("row0,col%d should be 0, got 0x%04x", col, fb.Pixels[0*Width+col])
		}
	}
}

func TestCharUnknown(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xFFFF

	// rune(0) is not in the font, should fall back to '?'
	fb.Char(0, 0, rune(0), font.Spleen8x16, fg)

	// '?' glyph row 2: 0x7c = 0111 1100 — col 1 should be set
	if fb.Pixels[2*Width+1] != fg {
		t.Errorf("unknown rune fallback to '?': row2,col1 should be fg")
	}
}

func TestString(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xFFFF

	// "Hi" — 'H' at x=0, 'i' at x=8
	fb.String(0, 0, "Hi", font.Spleen8x16, fg)

	// 'H' row 2: 0xc6 = 1100 0110 — col 0 set
	if fb.Pixels[2*Width+0] != fg {
		t.Errorf("'H' at x=0, row2 col0 should be fg")
	}
	// 'i' row 2: 0x18 = 0001 1000 — col 3 of 'i' (x=8+3=11) set
	if fb.Pixels[2*Width+11] != fg {
		t.Errorf("'i' at x=8, row2 col3 should be fg")
	}
	// col 8 in row 0 should not be set by 'i' (row 0 of 'i' is 0x00)
	if fb.Pixels[0*Width+8] != 0 {
		t.Errorf("'i' at x=8, row0 col0 should be 0")
	}
}

func TestStringClipping(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xFFFF

	// A string starting at x=156 with 8-wide font:
	// first char at 156 fits (156+8=164 > 160, but starts before 160 — per spec clip means
	// stop rendering chars that would start past Width)
	// x=156 < 160 so first char renders (clipped by SetPixel bounds)
	// x=164 >= 160 so second char does not render
	fb.String(156, 0, "AB", font.Spleen8x16, fg)

	// 'A' row 2: 0x7c — cols 1-5 at x+1..x+5 = 157..161; only 157,158,159 in bounds
	// pixel at (157, 2) should be set (col 1 of 'A' from x=156)
	if fb.Pixels[2*Width+157] != fg {
		t.Errorf("clipped first char: pixel (157,2) should be fg")
	}
	// 'B' should not render at all (starts at x=164 >= 160)
	// Verify columns 0..155 in row 2 are all zero (no 'B' bled in)
	for x := 0; x < 156; x++ {
		if fb.Pixels[2*Width+x] != 0 {
			t.Errorf("unexpected pixel at (%d,2) = 0x%04x", x, fb.Pixels[2*Width+x])
		}
	}
}

func TestBar(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xF800
	const bg uint16 = 0x001F
	// 50% bar: x=10,y=20,w=60,h=5
	fb.Bar(10, 20, 60, 5, 50, fg, bg)

	// first 30px (x=10..39) should be fg
	for x := 10; x < 40; x++ {
		for y := 20; y < 25; y++ {
			if fb.Pixels[y*Width+x] != fg {
				t.Errorf("(%d,%d) should be fg, got 0x%04x", x, y, fb.Pixels[y*Width+x])
			}
		}
	}
	// last 30px (x=40..69) should be bg
	for x := 40; x < 70; x++ {
		for y := 20; y < 25; y++ {
			if fb.Pixels[y*Width+x] != bg {
				t.Errorf("(%d,%d) should be bg, got 0x%04x", x, y, fb.Pixels[y*Width+x])
			}
		}
	}
}

func TestBar0Percent(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xF800
	const bg uint16 = 0x001F
	fb.Bar(0, 0, 60, 5, 0, fg, bg)

	for x := 0; x < 60; x++ {
		for y := 0; y < 5; y++ {
			if fb.Pixels[y*Width+x] != bg {
				t.Errorf("0%% bar: (%d,%d) should be bg, got 0x%04x", x, y, fb.Pixels[y*Width+x])
			}
		}
	}
}

func TestBar100Percent(t *testing.T) {
	var fb Framebuffer
	const fg uint16 = 0xF800
	const bg uint16 = 0x001F
	fb.Bar(0, 0, 60, 5, 100, fg, bg)

	for x := 0; x < 60; x++ {
		for y := 0; y < 5; y++ {
			if fb.Pixels[y*Width+x] != fg {
				t.Errorf("100%% bar: (%d,%d) should be fg, got 0x%04x", x, y, fb.Pixels[y*Width+x])
			}
		}
	}
}
