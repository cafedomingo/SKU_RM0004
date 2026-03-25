package font

// Custom glyph runes (Private Use Area to avoid collisions).
const (
	ArrowUp   rune = '\uE000'
	ArrowDown rune = '\uE001'
	Diamond   rune = '\uE002'
)

// AddCustomGlyphs adds arrow and diamond glyphs sized for this font.
// Only supports 6-wide fonts (6x12).
func (f *Font) AddCustomGlyphs() {
	if f.Width != 6 || f.Height != 12 {
		return
	}
	// Up arrow (5x6, centered in 6x12 cell with 3px top/bottom padding)
	// Bit patterns are MSB-aligned in 8 bits (6 pixels used):
	//   ..X...  = 0x20
	//   .XXX..  = 0x70
	//   XXXXX.  = 0xF8
	//   ..X...  = 0x20
	//   ..X...  = 0x20
	//   ..X...  = 0x20
	f.Glyphs[ArrowUp] = []byte{
		0x00, 0x00, 0x00,
		0x20, 0x70, 0xF8, 0x20, 0x20, 0x20,
		0x00, 0x00, 0x00,
	}
	// Down arrow (reversed)
	f.Glyphs[ArrowDown] = []byte{
		0x00, 0x00, 0x00,
		0x20, 0x20, 0x20, 0xF8, 0x70, 0x20,
		0x00, 0x00, 0x00,
	}
	// Diamond (6x6, centered in 6x12 cell with 3px top/bottom padding)
	// Bit patterns:
	//   ..XX..  = 0x30
	//   .XXXX.  = 0x78
	//   XXXXXX  = 0xFC
	//   XXXXXX  = 0xFC
	//   .XXXX.  = 0x78
	//   ..XX..  = 0x30
	f.Glyphs[Diamond] = []byte{
		0x00, 0x00, 0x00,
		0x30, 0x78, 0xFC, 0xFC, 0x78, 0x30,
		0x00, 0x00, 0x00,
	}
}
