package glyphraster

// Font holds read-only font data and precomputed metrics.
// All methods are safe for concurrent use.
type Font struct {
	// private: unitsPerEm, glyphCount, hmtx, loca, glyf, etc.
}

// LoadFont parses a TrueType/OpenType font from data.
// The data slice must stay valid for the lifetime of the Font.
// This performs heap allocations (parsing tables), but after this no further allocations occur.
func LoadFont(data []byte) (*Font, error)

// Basic metrics -------------------------------------------------

func (f *Font) UnitsPerEm() int32

// GlyphAdvance returns advance width in font units.
func (f *Font) GlyphAdvance(glyphID uint16) int32

// GlyphLeftBearing returns left side bearing in font units.
func (f *Font) GlyphLeftBearing(glyphID uint16) int32

// GlyphBBox returns bounding box in font units relative to origin (0,0 baseline).
func (f *Font) GlyphBBox(glyphID uint16) (xmin, ymin, xmax, ymax int32)

// Scale conversion ----------------------------------------------

// ScaleForPixelHeight returns scale = pixelHeight / UnitsPerEm().
func (f *Font) ScaleForPixelHeight(pixelHeight float32) float32

// Per‑glyph rasterization (no atlas) ----------------------------

// GlyphBitmapBox computes the pixel size and offset of the glyph's bitmap.
// Returns width, height, and the offset from the pen position to the bitmap's top‑left.
func (f *Font) GlyphBitmapBox(glyphID uint16, scale float32) (width, height, xoff, yoff int)

// MakeGlyphBitmap rasterizes a single glyph into a caller‑supplied alpha buffer.
// buf must be at least width*height bytes (or stride*height). stride is bytes per row (>= width).
// The bitmap is 8‑bit alpha, 0 = transparent, 255 = opaque.
// The caller must have obtained width/height via GlyphBitmapBox.
func (f *Font) MakeGlyphBitmap(glyphID uint16, scale float32, buf []byte, stride int)

// Atlas baking --------------------------------------------------

// AtlasGlyph describes the placement of one glyph in a packed atlas.
type AtlasGlyph struct {
	X, Y, W, H int   // rectangle in atlas (pixels)
	Xoff, Yoff int   // offset from pen to bitmap top‑left (pixels)
	Advance    int32 // advance width in font units (scale‑independent)
}

// BakeAtlas packs the given glyphs into a single atlas image.
// atlas buffer is provided by the caller and must be atlasW*atlasH bytes.
// The function writes 8‑bit alpha values row‑major, with 1‑pixel padding between glyphs.
// glyphRecs must be a slice of length >= len(glyphs) and will be filled with placement info.
// Returns an error if the atlas is too small to fit all glyphs.
func (f *Font) BakeAtlas(glyphs []uint16, scale float32, atlas []byte, atlasW, atlasH int, glyphRecs []AtlasGlyph) error

// Layout and measurement ----------------------------------------

// GlyphRun represents a single shaped glyph, typically from lefevre.
type GlyphRun struct {
	GlyphID          uint16
	Advance          int32 // in font units
	XOffset, YOffset int32 // extra displacement in font units (for kerning, marks)
}

// Layout computes pixel positions for a sequence of glyph runs.
// It starts at (penX, penY) in pixels (baseline position) and calls drawGlyph for each glyph
// with its top‑left pixel coordinate. No heap allocations are performed during layout.
func (f *Font) Layout(runs []GlyphRun, scale float32, penX, penY int, drawGlyph func(glyphID uint16, x, y int))

// Measure returns the total advance width of the runs in pixels.
func (f *Font) Measure(runs []GlyphRun, scale float32) int
