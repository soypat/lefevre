package raster

import (
	"os"
	"testing"

	"github.com/soypat/lefevre"
)

func loadTestFont(t *testing.T) *lefevre.Font {
	t.Helper()
	data, err := os.ReadFile("../testdata/DejaVuSans.ttf")
	if err != nil {
		t.Skipf("test font not available: %v", err)
	}
	f, err := lefevre.FontFromMemory(data, 0)
	if err != nil {
		t.Fatalf("FontFromMemory: %v", err)
	}
	return f
}

// GlyphBox tests

func TestGlyphBox_LatinA(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	scale := float32(32) / float32(upem)
	gid := f.GlyphID('A')
	w, h, xoff, yoff := GlyphBox(f, gid, scale)
	if w <= 0 || h <= 0 {
		t.Errorf("GlyphBox('A') size = %dx%d, want positive", w, h)
	}
	// xoff should be >= 0 for 'A' (no left overhang).
	if xoff < 0 {
		t.Errorf("GlyphBox('A') xoff = %d, want >= 0", xoff)
	}
	// yoff should be negative (bitmap starts above baseline).
	if yoff >= 0 {
		t.Errorf("GlyphBox('A') yoff = %d, want < 0", yoff)
	}
}

func TestGlyphBox_Space(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	scale := float32(32) / float32(upem)
	gid := f.GlyphID(' ')
	w, h, _, _ := GlyphBox(f, gid, scale)
	if w != 0 || h != 0 {
		t.Errorf("GlyphBox(' ') = %dx%d, want 0x0", w, h)
	}
}

func TestGlyphBox_ScaleProportional(t *testing.T) {
	f := loadTestFont(t)
	upem := f.UnitsPerEm()
	gid := f.GlyphID('A')
	scale1 := float32(16) / float32(upem)
	scale2 := float32(32) / float32(upem)
	w1, h1, _, _ := GlyphBox(f, gid, scale1)
	w2, h2, _, _ := GlyphBox(f, gid, scale2)
	// At double scale, dimensions should be roughly double (within rounding).
	if w2 < w1 || h2 < h1 {
		t.Errorf("double scale should produce larger box: %dx%d vs %dx%d", w1, h1, w2, h2)
	}
}

// FindPackedGlyph tests

func TestFindPackedGlyph_Found(t *testing.T) {
	placements := []PackedGlyph{
		{GlyphID: 10, X: 0, Y: 0, W: 8, H: 12},
		{GlyphID: 20, X: 9, Y: 0, W: 7, H: 11},
		{GlyphID: 30, X: 17, Y: 0, W: 9, H: 13},
	}
	pg, ok := FindPackedGlyph(placements, 20)
	if !ok {
		t.Fatal("FindPackedGlyph(20) not found")
	}
	if pg.GlyphID != 20 || pg.W != 7 {
		t.Errorf("FindPackedGlyph(20) = %+v, want GlyphID=20, W=7", pg)
	}
}

func TestFindPackedGlyph_NotFound(t *testing.T) {
	placements := []PackedGlyph{
		{GlyphID: 10},
		{GlyphID: 20},
		{GlyphID: 30},
	}
	_, ok := FindPackedGlyph(placements, 15)
	if ok {
		t.Error("FindPackedGlyph(15) should not be found")
	}
}

func TestFindPackedGlyph_Empty(t *testing.T) {
	_, ok := FindPackedGlyph(nil, 10)
	if ok {
		t.Error("FindPackedGlyph on nil slice should not be found")
	}
}

func TestRasterizeGlyph(t *testing.T) {
	f := loadTestFont(t)
	scale := float32(32) / float32(f.UnitsPerEm())
	gid := f.GlyphID('H')
	w, h, _, _ := GlyphBox(f, gid, scale)
	var r ScanlineRasterizer
	buf := make([]byte, w*h)
	segs := RasterizeGlyph(&r, buf, w, f, gid, scale, nil)

	// 'H' rests on the baseline, which is the bitmap's bottom edge: its stems fully cover the bottom row.
	var bottom byte
	for _, a := range buf[(h-1)*w:] {
		bottom = max(bottom, a)
	}
	if bottom < 250 {
		t.Errorf("bottom row coverage %d, want full", bottom)
	}

	// Drawn into a wider bitmap, the glyph's rows are the same.
	const stride, x0 = 100, 7
	wide := make([]byte, (h-1)*stride+x0+w)
	RasterizeGlyph(&r, wide[x0:], stride, f, gid, scale, segs)
	for y := range h {
		if got, want := wide[x0+y*stride:][:w], buf[y*w:][:w]; string(got) != string(want) {
			t.Fatalf("row %d differs with stride %d", y, stride)
		}
	}

	// An empty glyph draws nothing.
	space := f.GlyphID(' ')
	if w, h, _, _ := GlyphBox(f, space, scale); w != 0 || h != 0 {
		t.Fatalf("space has a %dx%d box", w, h)
	}
	RasterizeGlyph(&r, nil, 0, f, space, scale, segs)
}
