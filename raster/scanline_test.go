package raster

import (
	"testing"

	"github.com/soypat/lefevre"
)

func TestScanlineRasterizer_Triangle(t *testing.T) {
	// A triangle outline in "font units" that covers a known area.
	// Triangle: (2,2) -> (14,2) -> (8,14) -> close.
	segs := []lefevre.Segment{
		{Op: lefevre.SegmentMoveTo, X: 2, Y: 2},
		{Op: lefevre.SegmentLineTo, X: 14, Y: 2},
		{Op: lefevre.SegmentLineTo, X: 8, Y: 14},
		{Op: lefevre.SegmentClose},
	}
	const w, h = 16, 16
	buf := make([]byte, w*h)
	var r ScanlineRasterizer
	r.Rasterize(buf, w, h, w, segs, 1, 0, 0)

	// Check that some pixels inside the triangle have coverage.
	nonZero := 0
	for _, b := range buf {
		if b > 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("triangle rasterization produced all-zero buffer")
	}
}

func TestScanlineRasterizer_EmptyOutline(t *testing.T) {
	const w, h = 8, 8
	buf := make([]byte, w*h)
	// Set to non-zero to verify rasterizer clears.
	for i := range buf {
		buf[i] = 0xFF
	}
	var r ScanlineRasterizer
	r.Rasterize(buf, w, h, w, nil, 1, 0, 0)
	for i, b := range buf {
		if b != 0 {
			t.Errorf("buf[%d] = %d, want 0 for empty outline", i, b)
			break
		}
	}
}

func TestScanlineRasterizer_GlyphA(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	gid := f.GlyphID('A')
	w, h, _, _ := GlyphBox(f, gid, scale)
	if w == 0 || h == 0 {
		t.Fatal("GlyphBox returned 0 size for 'A'")
	}
	segs := f.GlyphOutline(nil, gid)
	if len(segs) == 0 {
		t.Fatal("no outline segments for 'A'")
	}

	buf := make([]byte, w*h)
	xMin, _, _, yMax := f.GlyphBounds(gid)
	xoff := -float32(xMin) * scale
	yoff := float32(yMax) * scale // flip Y: yMax in font units maps to y=0 in bitmap
	var r ScanlineRasterizer
	r.Rasterize(buf, w, h, w, segs, scale, xoff, yoff)

	nonZero := 0
	for _, b := range buf {
		if b > 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("glyph 'A' rasterization produced all-zero buffer")
	}
	// 'A' should have a mix of filled and empty pixels (not all filled).
	total := w * h
	if nonZero == total {
		t.Error("glyph 'A' filled entire buffer — likely a rasterization error")
	}
}

func TestScanlineRasterizer_ScratchReuse(t *testing.T) {
	// Rasterize twice with the same rasterizer to verify scratch buffer reuse.
	segs := []lefevre.Segment{
		{Op: lefevre.SegmentMoveTo, X: 1, Y: 1},
		{Op: lefevre.SegmentLineTo, X: 7, Y: 1},
		{Op: lefevre.SegmentLineTo, X: 7, Y: 7},
		{Op: lefevre.SegmentLineTo, X: 1, Y: 7},
		{Op: lefevre.SegmentClose},
	}
	const w, h = 8, 8
	buf1 := make([]byte, w*h)
	buf2 := make([]byte, w*h)
	var r ScanlineRasterizer
	r.Rasterize(buf1, w, h, w, segs, 1, 0, 0)
	r.Rasterize(buf2, w, h, w, segs, 1, 0, 0)

	for i := range buf1 {
		if buf1[i] != buf2[i] {
			t.Errorf("pixel %d differs between calls: %d vs %d", i, buf1[i], buf2[i])
			break
		}
	}
}

func TestScanlineRasterizer_QuadBezier(t *testing.T) {
	// A simple curved shape using QuadTo.
	segs := []lefevre.Segment{
		{Op: lefevre.SegmentMoveTo, X: 2, Y: 8},
		{Op: lefevre.SegmentQuadTo, X: 14, Y: 8, Cx: 8, Cy: 0},
		{Op: lefevre.SegmentClose},
	}
	const w, h = 16, 16
	buf := make([]byte, w*h)
	var r ScanlineRasterizer
	r.Rasterize(buf, w, h, w, segs, 1, 0, 0)

	nonZero := 0
	for _, b := range buf {
		if b > 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("quad bezier rasterization produced all-zero buffer")
	}
}
