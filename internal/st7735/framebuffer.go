package st7735

import "github.com/cafedomingo/SKU_RM0004/internal/font"

const (
	Width  = 160
	Height = 80
)

// Framebuffer holds a 160x80 RGB565 pixel buffer.
type Framebuffer struct {
	Pixels [Width * Height]uint16
}

// Fill sets every pixel to color.
func (fb *Framebuffer) Fill(color uint16) {
	for i := range fb.Pixels {
		fb.Pixels[i] = color
	}
}

// SetPixel sets the pixel at (x, y) to color. Out-of-bounds coordinates are silently ignored.
func (fb *Framebuffer) SetPixel(x, y int, color uint16) {
	if x < 0 || x >= Width || y < 0 || y >= Height {
		return
	}
	fb.Pixels[y*Width+x] = color
}

// Rect fills the rectangle at (x, y) with size (w, h) in color, clipped to display bounds.
func (fb *Framebuffer) Rect(x, y, w, h int, color uint16) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			fb.SetPixel(col, row, color)
		}
	}
}

// Char renders rune ch using font f at position (x, y) in color.
// Only foreground pixels (set bits) are drawn; background pixels are left untouched.
func (fb *Framebuffer) Char(x, y int, ch rune, f *font.Font, color uint16) {
	glyph := f.Glyph(ch)
	if glyph == nil {
		return
	}
	for row := 0; row < f.Height; row++ {
		bits := glyph[row]
		for col := 0; col < f.Width; col++ {
			if (bits>>(7-col))&1 == 1 {
				fb.SetPixel(x+col, y+row, color)
			}
		}
	}
}

// String renders string s using font f at position (x, y) in color.
// Characters are placed side by side; characters that would start at or past Width are not rendered.
func (fb *Framebuffer) String(x, y int, s string, f *font.Font, color uint16) {
	cx := x
	for _, ch := range s {
		if cx >= Width {
			break
		}
		fb.Char(cx, y, ch, f, color)
		cx += f.Width
	}
}

// Bar draws a horizontal progress bar at (x, y) with size (w, h).
// pct is the fill percentage (0–100). Filled portion uses fg color, remainder uses bg color.
func (fb *Framebuffer) Bar(x, y, w, h int, pct int, fg, bg uint16) {
	filled := w * pct / 100
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			if col-x < filled {
				fb.SetPixel(col, row, fg)
			} else {
				fb.SetPixel(col, row, bg)
			}
		}
	}
}
