package raster

import (
	"github.com/soypat/lefevre"
)

// BuildQuads appends one DrawQuad per non-empty shaped glyph to dst.
// origin is the pen start (Y-down pixels). scale must match the scale used
// to produce placements. placements must be sorted by GlyphID for lookup;
// glyphs without a matching placement or with zero-size placements are
// skipped but still advance the pen.
func BuildQuads(dst []DrawQuad, runs []lefevre.Run, placements []PackedGlyph, originX, originY int, scale float32) []DrawQuad {
	penX := float32(originX)
	penY := float32(originY)
	for _, run := range runs {
		for _, g := range run.Glyphs {
			p, ok := FindPackedGlyph(placements, g.ID)
			if ok && p.W > 0 && p.H > 0 {
				gx := penX + float32(g.OffsetX)*scale
				gy := penY - float32(g.OffsetY)*scale
				dst = append(dst, DrawQuad{
					DstX: int(gx) + p.Xoff,
					DstY: int(gy) + p.Yoff,
					DstW: p.W,
					DstH: p.H,
					SrcX: p.X,
					SrcY: p.Y,
					SrcW: p.W,
					SrcH: p.H,
				})
			}
			penX += float32(g.AdvanceX) * scale
			penY -= float32(g.AdvanceY) * scale
		}
	}
	return dst
}
