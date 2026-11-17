package raster

import (
	"testing"
)

func TestPack_FitsASCII(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	scale := float32(16) / float32(upem)
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
	upem := f.UnitsPerEm()
	scale := float32(32) / float32(upem)
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
	upem := f.UnitsPerEm()
	scale := float32(16) / float32(upem)
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
	upem := f.UnitsPerEm()
	scale := float32(16) / float32(upem)
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

func TestPack_PopulatesGlyphMetrics(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	scale := float32(32) / float32(upem)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	glyphs := []uint16{f.GlyphID('A')}
	placements, err := cfg.Pack(nil, glyphs, 256, 256)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}
	p := placements[0]
	if p.GlyphID != glyphs[0] {
		t.Errorf("GlyphID = %d, want %d", p.GlyphID, glyphs[0])
	}
	if p.W <= 0 || p.H <= 0 {
		t.Errorf("W/H = %dx%d, want positive for 'A'", p.W, p.H)
	}
	// 'A' sits above baseline, so Yoff must be negative (bitmap top above pen).
	if p.Yoff >= 0 {
		t.Errorf("Yoff = %d, want < 0 for 'A'", p.Yoff)
	}
}

func TestBakeAtlas_CoverageStaysInsideRect(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	scale := float32(24) / float32(upem)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 2}

	glyphs := []uint16{f.GlyphID('A'), f.GlyphID('g'), f.GlyphID('M')}
	const atlasW, atlasH = 128, 128
	atlas := make([]byte, atlasW*atlasH)
	placements := make([]PackedGlyph, len(glyphs))

	var r ScanlineRasterizer
	if err := cfg.BakeAtlas(&r, glyphs, atlas, atlasW, atlasH, placements); err != nil {
		t.Fatalf("BakeAtlas: %v", err)
	}

	inside := func(x, y int) bool {
		for _, p := range placements {
			if x >= p.X && x < p.X+p.W && y >= p.Y && y < p.Y+p.H {
				return true
			}
		}
		return false
	}
	for y := 0; y < atlasH; y++ {
		for x := 0; x < atlasW; x++ {
			if atlas[y*atlasW+x] != 0 && !inside(x, y) {
				t.Fatalf("coverage at (%d,%d) outside every placement rect", x, y)
			}
		}
	}
}

func TestPack_AppendsToExisting(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	scale := float32(16) / float32(upem)
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

// bottomRowMax returns the highest coverage in the bottom row of p's atlas cell.
func bottomRowMax(atlas []byte, atlasW int, p PackedGlyph) byte {
	var m byte
	for _, a := range atlas[(p.Y+p.H-1)*atlasW+p.X:][:p.W] {
		m = max(m, a)
	}
	return m
}

func TestBakeAtlas_Baseline(t *testing.T) {
	// Glyphs resting on the baseline end exactly at the bottom of their bitmap, whatever the
	// fractional part of their height: drawn at the pen plus Yoff, they share one baseline.
	f := loadTestFont(t)
	cfg := PackConfig{Font: f, Scale: float32(32) / float32(f.UnitsPerEm()), Padding: 1}
	const flat = "HIx"
	glyphs := make([]uint16, len(flat))
	for i, r := range flat {
		glyphs[i] = f.GlyphID(r)
		if _, yMin, _, _ := f.GlyphBounds(glyphs[i]); yMin != 0 {
			t.Fatalf("%q does not rest on the baseline: yMin=%d", r, yMin)
		}
	}
	placements := make([]PackedGlyph, len(glyphs))
	atlas := make([]byte, 128*128)
	var rast ScanlineRasterizer
	if err := cfg.BakeAtlas(&rast, glyphs, atlas, 128, 128, placements); err != nil {
		t.Fatal(err)
	}
	for i, p := range placements {
		if p.Yoff+p.H != 0 {
			t.Errorf("%q: bitmap bottom %d pixels from the baseline", flat[i], p.Yoff+p.H)
		}
		if m := bottomRowMax(atlas, 128, p); m < 250 {
			t.Errorf("%q: bottom row coverage %d, want full coverage at the baseline", flat[i], m)
		}
	}
}

func TestBakeAtlas_BottomEdge(t *testing.T) {
	// Glyphs placed against the atlas's bottom edge are drawn, wherever they are in the row.
	f := loadTestFont(t)
	cfg := PackConfig{Font: f, Scale: float32(32) / float32(f.UnitsPerEm())}
	glyphs := []uint16{f.GlyphID('H'), f.GlyphID('I')}
	_, h, _, _ := GlyphBox(f, glyphs[0], cfg.Scale)
	const atlasW = 64
	atlas := make([]byte, atlasW*h)
	placements := make([]PackedGlyph, len(glyphs))
	var rast ScanlineRasterizer
	if err := cfg.BakeAtlas(&rast, glyphs, atlas, atlasW, h, placements); err != nil {
		t.Fatal(err)
	}
	for i, p := range placements {
		if p.Y+p.H != h {
			t.Fatalf("glyph %d does not touch the atlas bottom: Y=%d H=%d", i, p.Y, p.H)
		}
		if bottomRowMax(atlas, atlasW, p) == 0 {
			t.Errorf("glyph %d at X=%d has no ink in its bottom row", i, p.X)
		}
	}
}
