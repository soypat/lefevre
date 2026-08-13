package sfnt

import (
	"github.com/soypat/lefevre"
	"github.com/soypat/lefevre/internal"
)

// Subsetter produces font programs carrying a chosen set of glyphs; the zero
// value is ready, reuse recycles its buffers, and glyph ids never shift.
type Subsetter struct {
	// Keep names tables copied verbatim beyond the required set. A PDF embedder
	// needs none; a consumer that maps codepoints itself wants "cmap".
	Keep []string

	keep  []bool   // per-glyph retention, indexed by glyph id
	stack []uint16 // composite traversal
	glyf  []byte   // assembled outlines
	loca  []byte   // assembled offsets, always long-form
	head  []byte   // the source head with its two edited fields
	out   []Table  // the directory handed to Append
}

// required lists the tables a subset copies verbatim for its outlines to draw:
// the metrics that place them and the hinting that tunes them at small sizes.
var required = [...]string{"cvt ", "fpgm", "hhea", "hmtx", "maxp", "prep"}

// AppendSubset appends to dst a font program holding only gids and the glyphs
// they reference, at unchanged ids. .notdef is kept; ids past the end are not.
func (s *Subsetter) AppendSubset(dst []byte, f *lefevre.Font, gids []uint16) ([]byte, error) {
	if f.OutlineFormat() != lefevre.OutlineGlyf {
		return dst, ErrNoOutlines
	}
	glyf, head, maxp := f.Table("glyf"), f.Table("head"), f.Table("maxp")
	if len(head) < 54 || len(maxp) < 6 {
		return dst, ErrTruncated
	}
	numGlyphs := f.NumGlyphs()
	if numGlyphs == 0 {
		return dst, ErrNoOutlines
	}

	// .notdef must survive: it is what an unmapped code draws.
	s.mark(f, numGlyphs, gids, true)

	// Lay the kept outlines down back to back, recording where each landed.
	// A dropped glyph gets a zero-length range, which is how loca says "this
	// glyph is empty" — the same thing it already says for a space.
	internal.SliceReuse(&s.glyf, len(glyf)/8)
	internal.SliceReuse(&s.loca, 4*(numGlyphs+1))
	s.loca = s.loca[:4*(numGlyphs+1)]
	for gid := range numGlyphs {
		putBE32(s.loca[4*gid:], uint32(len(s.glyf)))
		if !s.keep[gid] {
			continue
		}
		s.glyf = append(s.glyf, f.GlyphData(uint16(gid))...)
		for len(s.glyf)%4 != 0 {
			s.glyf = append(s.glyf, 0)
		}
	}
	putBE32(s.loca[4*numGlyphs:], uint32(len(s.glyf)))
	if len(s.glyf) == 0 {
		// Every kept glyph was empty, but a glyf font subsets to a glyf font.
		// loca gives every glyph a zero-length range, so nothing points here.
		s.glyf = append(s.glyf, 0, 0, 0, 0)
	}

	// head is the one copied table that changes: loca is now unconditionally
	// long, and the whole-font checksum is filled in once the file is
	// assembled, so it must go in reading zero.
	internal.SliceReuse(&s.head, len(head))
	s.head = append(s.head, head...)
	putBE32(s.head[8:], 0)
	putBE16(s.head[50:], 1)

	s.out = append(s.out[:0],
		Table{Tag: "glyf", Data: s.glyf},
		Table{Tag: "head", Data: s.head},
		Table{Tag: "loca", Data: s.loca},
	)
	for _, tag := range required {
		if d := f.Table(tag); d != nil {
			s.out = append(s.out, Table{Tag: tag, Data: d})
		}
	}
	for _, tag := range s.Keep {
		if d := f.Table(tag); d != nil && !hasTag(s.out, tag) {
			s.out = append(s.out, Table{Tag: tag, Data: d})
		}
	}
	return Append(dst, s.out)
}

// AppendClosure appends gids plus their component glyphs, transitively, ascending
// and deduplicated. It is what [Subsetter.AppendSubset] keeps, minus .notdef.
func (s *Subsetter) AppendClosure(dst []uint16, f *lefevre.Font, gids []uint16) []uint16 {
	numGlyphs := f.NumGlyphs()
	if numGlyphs == 0 {
		return dst
	}
	s.mark(f, numGlyphs, gids, false)
	for gid := range numGlyphs {
		if s.keep[gid] {
			dst = append(dst, uint16(gid))
		}
	}
	return dst
}

// mark fills s.keep with the transitive closure of gids over composite
// references, adding .notdef when keepNotdef.
func (s *Subsetter) mark(f *lefevre.Font, numGlyphs int, gids []uint16, keepNotdef bool) {
	internal.SliceReuse(&s.keep, numGlyphs)
	s.keep = s.keep[:numGlyphs]
	clear(s.keep)

	internal.SliceReuse(&s.stack, len(gids)+8)
	if keepNotdef {
		s.keep[0] = true
		s.stack = append(s.stack, 0)
	}
	for _, gid := range gids {
		if int(gid) < numGlyphs && !s.keep[gid] {
			s.keep[gid] = true
			s.stack = append(s.stack, gid)
		}
	}
	// A component may itself be a composite, so this walks rather than scans.
	for len(s.stack) > 0 {
		gid := s.stack[len(s.stack)-1]
		s.stack = s.stack[:len(s.stack)-1]
		n := len(s.stack)
		s.stack = f.AppendGlyphComponents(s.stack, gid)
		// Filter the components in place: those already kept are not work.
		w := n
		for _, comp := range s.stack[n:] {
			if int(comp) < numGlyphs && !s.keep[comp] {
				s.keep[comp] = true
				s.stack[w] = comp
				w++
			}
		}
		s.stack = s.stack[:w]
	}
}

// hasTag reports whether tables already carries the named table.
func hasTag(tables []Table, tag string) bool {
	for _, t := range tables {
		if t.Tag == tag {
			return true
		}
	}
	return false
}
