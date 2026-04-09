package glyphraster

import (
	"github.com/soypat/lefevre"
)

func ExampleMakeGlyphBitmap() {
	var ttfData []byte
	font, _ := LoadFont(ttfData)
	scale := font.ScaleForPixelHeight(16)

	glyphID := uint16(65) // 'A'
	w, h, xoff, yoff := font.GlyphBitmapBox(glyphID, scale)
	buf := make([]byte, w*h) // caller‑managed, could be reused
	font.MakeGlyphBitmap(glyphID, scale, buf, w)
	_, _ = xoff, yoff
	// draw buf at (penX + xoff, penY + yoff) using your own blitter

}

func ExampleAtlasGlyph() {
	var font Font
	glyphs := []uint16{65, 66, 67} // 'A','B','C'
	scale := font.ScaleForPixelHeight(16)

	// estimate atlas size (e.g., 256x256)
	const atlasW, atlasH = 256, 256
	atlas := make([]byte, atlasW*atlasH)
	glyphRecs := make([]AtlasGlyph, len(glyphs))

	err := font.BakeAtlas(glyphs, scale, atlas, atlasW, atlasH, glyphRecs)
	if err != nil { /* handle */
	}

	// upload atlas to GPU texture (atlas is 8‑bit alpha, you can convert to RGBA if needed)
	// build a map for fast lookup:
	atlasMap := make(map[uint16]AtlasGlyph)
	for i, gid := range glyphs {
		atlasMap[gid] = glyphRecs[i]
	}
}

func Example() {
	// Assume lefevre gives you []lefevre.Glyph – convert to our GlyphRun slice
	var runs []GlyphRun
	var lefevreGlyphs []lefevre.Glyph
	for _, lg := range lefevreGlyphs {
		runs = append(runs, GlyphRun{
			GlyphID: lg.ID,
			Advance: lg.AdvanceX, // already in font units
			// XOffset, YOffset from lefevre if any
		})
	}

	// Layout and draw using the atlas map from above
	// font.Layout(runs, scale, startX, startY, func(gid uint16, x, y int) {
	// rec := atlasMap[gid]
	// blit rectangle (x+rec.Xoff, y+rec.Yoff) of size rec.W x rec.H
	// from atlas at (rec.X, rec.Y)
	// })
}
