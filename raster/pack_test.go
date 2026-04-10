package raster

import (
	"testing"
)

func TestPack_FitsASCII(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(16) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	// Printable ASCII: 32..126
	glyphs := make([]uint16, 95)
	for i := range glyphs {
		glyphs[i] = f.GlyphID(rune(32 + i))
	}

	placements, err := cfg.Pack(nil, glyphs, 256, 256)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	if len(placements) != len(glyphs) {
		t.Errorf("got %d placements, want %d", len(placements), len(glyphs))
	}
}

func TestPack_TooSmall(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	glyphs := make([]uint16, 95)
	for i := range glyphs {
		glyphs[i] = f.GlyphID(rune(32 + i))
	}

	_, err := cfg.Pack(nil, glyphs, 8, 8)
	if err == nil {
		t.Error("Pack should fail with 8x8 atlas for 95 glyphs at 32px")
	}
}

func TestPack_NoPadOverlap(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(16) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	glyphs := make([]uint16, 26)
	for i := range glyphs {
		glyphs[i] = f.GlyphID(rune('A' + i))
	}

	placements, err := cfg.Pack(nil, glyphs, 256, 256)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Check no two rects overlap (including padding).
	for i := 0; i < len(placements); i++ {
		a := placements[i]
		if a.W == 0 || a.H == 0 {
			continue
		}
		for j := i + 1; j < len(placements); j++ {
			b := placements[j]
			if b.W == 0 || b.H == 0 {
				continue
			}
			// Rects overlap if they intersect in both X and Y.
			if a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y {
				t.Errorf("placements %d and %d overlap: (%d,%d,%d,%d) vs (%d,%d,%d,%d)",
					i, j, a.X, a.Y, a.W, a.H, b.X, b.Y, b.W, b.H)
			}
		}
	}
}

func TestBakeAtlas_NonZeroCoverage(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(16) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	glyphs := []uint16{f.GlyphID('A'), f.GlyphID('B'), f.GlyphID('C')}
	const atlasW, atlasH = 128, 128
	atlas := make([]byte, atlasW*atlasH)
	placements := make([]PackedGlyph, len(glyphs))

	var r ScanlineRasterizer
	err := cfg.BakeAtlas(&r, glyphs, atlas, atlasW, atlasH, placements)
	if err != nil {
		t.Fatalf("BakeAtlas failed: %v", err)
	}

	nonZero := 0
	for _, b := range atlas {
		if b > 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("BakeAtlas produced all-zero atlas")
	}
}

func TestPack_AppendsToExisting(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(16) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	sentinel := PackedGlyph{GlyphID: 9999}
	glyphs := []uint16{f.GlyphID('X')}
	placements, err := cfg.Pack([]PackedGlyph{sentinel}, glyphs, 256, 256)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	if len(placements) < 2 {
		t.Fatal("expected at least 2 placements (sentinel + glyph)")
	}
	if placements[0] != sentinel {
		t.Error("Pack overwrote existing placement instead of appending")
	}
}
