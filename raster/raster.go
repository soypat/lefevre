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
	// buf must be at least stride*height bytes. Segments are in font units;
	// the rasterizer applies scale and offset internally.
	Rasterize(buf []byte, width, height, stride int, segments []lefevre.Segment, scale, xoff, yoff float32)
}

// GlyphBox computes the pixel bounding box for a glyph at the given scale.
// Returns width and height in pixels and the offset from pen position to
// the bitmap's top-left corner.
func GlyphBox(font *lefevre.Font, glyphID uint16, scale float32) (width, height, xoff, yoff int) {
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

// PackedGlyph describes the placement of one glyph in a packed atlas.
type PackedGlyph struct {
	GlyphID     uint16
	X, Y, W, H int // Position and size in atlas (pixels)
	Xoff, Yoff  int // Offset from pen position to bitmap top-left (pixels)
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
