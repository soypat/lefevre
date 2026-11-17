package raster

import (
	"math"
	"slices"

	"github.com/soypat/lefevre"
)

// Rasterizer converts glyph outlines into alpha coverage bitmaps.
// Implementations must be safe for sequential use.
type Rasterizer interface {
	// Rasterize fills buf with 8-bit alpha coverage for the given outline segments.
	// buf holds height rows of stride bytes; the last row only needs width bytes, so buf
	// may be a sub-slice of an atlas starting at a glyph's cell. Segments are in font
	// units; the rasterizer applies scale and offset internally.
	Rasterize(buf []byte, width, height, stride int, segments []lefevre.Segment, scale, xoff, yoff float32)
}

// Outliner is the glyph geometry rasterizing needs: bounds and outlines in font
// units with Y up. *lefevre.Font implements it.
type Outliner interface {
	GlyphBounds(glyphID uint16) (xMin, yMin, xMax, yMax int16)
	GlyphOutline(dst []lefevre.Segment, glyphID uint16) []lefevre.Segment
}

var _ Outliner = (*lefevre.Font)(nil)

// GlyphBox computes the pixel bounding box for a glyph at the given scale.
// Returns width and height in pixels and the offset from pen position to
// the bitmap's top-left corner.
func GlyphBox(font Outliner, glyphID uint16, scale float32) (width, height, xoff, yoff int) {
	xMin, yMin, xMax, yMax := font.GlyphBounds(glyphID)
	if xMin == 0 && yMin == 0 && xMax == 0 && yMax == 0 {
		return 0, 0, 0, 0
	}
	// Scale bounds to pixels. Floor mins, ceil maxes for tight pixel coverage.
	x0 := int(math.Floor(float64(float32(xMin) * scale)))
	y0 := int(math.Floor(float64(float32(yMin) * scale)))
	x1 := int(math.Ceil(float64(float32(xMax) * scale)))
	y1 := int(math.Ceil(float64(float32(yMax) * scale)))
	width = x1 - x0
	height = y1 - y0
	xoff = x0
	yoff = -y1 // Y-up to Y-down: top of glyph is -yMax in screen coords
	return width, height, xoff, yoff
}

// RasterizeGlyph rasterizes glyphID at scale into dst, a bitmap of stride bytes
// per row sized by [GlyphBox]. The outline is offset by the box's whole pixel
// corner, as stb_truetype does, so the bitmap drawn at the pen position plus
// GlyphBox's offsets places the glyph exactly at the pen: glyphs resting on the
// baseline share it. segs is scratch for the outline and is returned for reuse.
func RasterizeGlyph(rast Rasterizer, dst []byte, stride int, font Outliner, glyphID uint16, scale float32, segs []lefevre.Segment) []lefevre.Segment {
	w, h, xoff, yoff := GlyphBox(font, glyphID, scale)
	if w == 0 || h == 0 {
		return segs
	}
	segs = font.GlyphOutline(segs[:0], glyphID)
	for i := range segs {
		// Y-up to Y-down: a font point (fx, fy) lands on pixel (fx*scale-xoff, -fy*scale-yoff).
		segs[i].Y, segs[i].Cy = -segs[i].Y, -segs[i].Cy
	}
	rast.Rasterize(dst, w, h, stride, segs, scale, float32(-xoff), float32(-yoff))
	return segs
}

// PackedGlyph describes the placement of one glyph in a packed atlas.
type PackedGlyph struct {
	GlyphID    uint16
	X, Y, W, H int // Position and size in atlas (pixels)
	Xoff, Yoff int // Offset from pen position to bitmap top-left (pixels)
}

// PositionedGlyph is a glyph positioned at a specific pixel coordinate for drawing.
type PositionedGlyph struct {
	GlyphID uint16
	X, Y    int // Pixel position of bitmap top-left
}

// DrawQuad describes a textured quad for drawing a glyph from an atlas.
type DrawQuad struct {
	DstX, DstY, DstW, DstH int // Screen-space destination rectangle
	SrcX, SrcY, SrcW, SrcH int // Atlas-space source rectangle
}

// FindPackedGlyph finds a PackedGlyph by glyph ID in a sorted slice using binary search.
// The slice must be sorted by GlyphID. Returns the PackedGlyph and true if found.
func FindPackedGlyph(placements []PackedGlyph, glyphID uint16) (PackedGlyph, bool) {
	i, found := slices.BinarySearchFunc(placements, glyphID, func(pg PackedGlyph, target uint16) int {
		if pg.GlyphID < target {
			return -1
		}
		if pg.GlyphID > target {
			return 1
		}
		return 0
	})
	if found {
		return placements[i], true
	}
	return PackedGlyph{}, false
}
