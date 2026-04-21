package raster

import (
	"errors"
	"slices"

	"github.com/soypat/lefevre"
)

// PackConfig configures packing and baking glyphs into a single alpha atlas.
// Scale maps font units to pixels (typically pxSize/UnitsPerEm). Padding is
// the per-glyph margin reserved in the atlas so neighbouring glyphs do not
// bleed into each other under linear sampling.
type PackConfig struct {
	Font    *lefevre.Font
	Scale   float32
	Padding int
}

// Pack computes atlas placements for glyphs and appends them to dst.
// Placement order in the returned slice matches the input glyph order.
// Returns an error if the atlas cannot fit all glyphs.
func (cfg *PackConfig) Pack(dst []PackedGlyph, glyphs []uint16, atlasW, atlasH int) ([]PackedGlyph, error) {
	start := len(dst)
	for range glyphs {
		dst = append(dst, PackedGlyph{})
	}
	if err := cfg.packInto(glyphs, atlasW, atlasH, dst[start:]); err != nil {
		return dst[:start], err
	}
	return dst, nil
}

// BakeAtlas packs glyphs, rasterizes their outlines into atlas, and writes
// placements in the same order as glyphs. len(placements) must equal
// len(glyphs). atlas must have length >= atlasW*atlasH.
func (cfg *PackConfig) BakeAtlas(rast Rasterizer, glyphs []uint16, atlas []byte, atlasW, atlasH int, placements []PackedGlyph) error {
	if len(placements) != len(glyphs) {
		return errors.New("raster: len(placements) != len(glyphs)")
	}
	if len(atlas) < atlasW*atlasH {
		return errors.New("raster: atlas buffer too small")
	}
	if err := cfg.packInto(glyphs, atlasW, atlasH, placements); err != nil {
		return err
	}
	var segBuf, flipBuf []lefevre.Segment
	for i, gID := range glyphs {
		p := placements[i]
		if p.W == 0 || p.H == 0 {
			continue
		}
		segBuf = cfg.Font.GlyphOutline(segBuf[:0], gID)
		if len(segBuf) == 0 {
			continue
		}
		flipBuf = flipBuf[:0]
		for _, s := range segBuf {
			flipBuf = append(flipBuf, lefevre.Segment{
				Op: s.Op,
				X:  s.X, Y: -s.Y,
				Cx: s.Cx, Cy: -s.Cy,
			})
		}
		xMin, _, _, yMax := cfg.Font.GlyphBounds(gID)
		// After Y-flip, map font point (fx, -fy) through rasterizer:
		//   sx = fx*scale + xoff -> pixel fx*scale - xMin*scale
		//   sy = -fy*scale + yoff -> pixel yMax*scale - fy*scale
		xoff := -float32(xMin) * cfg.Scale
		yoff := float32(yMax) * cfg.Scale
		dst := atlas[p.Y*atlasW+p.X:]
		rast.Rasterize(dst, p.W, p.H, atlasW, flipBuf, cfg.Scale, xoff, yoff)
	}
	return nil
}

// packInto sizes each glyph via GlyphBox and places them with a shelf packer.
// Glyphs are placed in descending-height order for density, but out is
// written in the original input order.
func (cfg *PackConfig) packInto(glyphs []uint16, atlasW, atlasH int, out []PackedGlyph) error {
	pad := cfg.Padding
	if pad < 0 {
		pad = 0
	}
	if atlasW <= 2*pad || atlasH <= 2*pad {
		return errors.New("raster: atlas smaller than padding")
	}

	// Measure all glyphs first.
	for i, gID := range glyphs {
		w, h, xoff, yoff := GlyphBox(cfg.Font, gID, cfg.Scale)
		out[i] = PackedGlyph{GlyphID: gID, W: w, H: h, Xoff: xoff, Yoff: yoff}
	}

	// Sort indices by height descending for shelf density.
	order := make([]int, len(glyphs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return out[b].H - out[a].H
	})

	// Shelf pack.
	shelfY := pad
	shelfH := 0
	cursorX := pad
	for _, idx := range order {
		w, h := out[idx].W, out[idx].H
		if w == 0 || h == 0 {
			continue
		}
		if w+2*pad > atlasW {
			return errors.New("raster: glyph wider than atlas")
		}
		if cursorX+w+pad > atlasW {
			shelfY += shelfH + pad
			shelfH = 0
			cursorX = pad
		}
		if shelfY+h+pad > atlasH {
			return errors.New("raster: atlas too small for glyph set")
		}
		out[idx].X = cursorX
		out[idx].Y = shelfY
		cursorX += w + pad
		if h > shelfH {
			shelfH = h
		}
	}
	return nil
}
