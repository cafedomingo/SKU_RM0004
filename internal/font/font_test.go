package font

import "testing"

func TestGlyphLookupASCII(t *testing.T) {
	for r := rune(32); r <= 126; r++ {
		g := Spleen8x16.Glyph(r)
		if g == nil {
			t.Errorf("Glyph(%q) = nil, want non-nil", r)
		}
	}
}

func TestGlyphLookupUnicode(t *testing.T) {
	runes := []rune{'°', '◆', '▲', '▼', '█'}
	for _, r := range runes {
		g := Spleen8x16.Glyph(r)
		if g == nil {
			t.Errorf("Glyph(%q) = nil, want non-nil", r)
		}
	}
}

func TestGlyphLookupMissing(t *testing.T) {
	g := Spleen8x16.Glyph(rune(999999))
	q := Spleen8x16.Glyph('?')
	if g == nil {
		t.Fatal("missing rune returned nil, want '?' replacement")
	}
	if len(g) != len(q) {
		t.Fatalf("missing rune glyph len=%d, '?' glyph len=%d", len(g), len(q))
	}
	for i := range g {
		if g[i] != q[i] {
			t.Errorf("byte %d: got %#02x, want %#02x", i, g[i], q[i])
		}
	}
}

func TestFontDimensions(t *testing.T) {
	if Spleen8x16.Width != 8 {
		t.Errorf("Width = %d, want 8", Spleen8x16.Width)
	}
	if Spleen8x16.Height != 16 {
		t.Errorf("Height = %d, want 16", Spleen8x16.Height)
	}
}
