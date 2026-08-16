package sfnt

import (
	"github.com/soypat/lefevre/internal"
)

// Source is a font that will hand over its sfnt tables, which is all a subset
// needs. *lefevre.Font satisfies it, and so does anything else that can read a
// table directory: subsetting asks a font for its bytes, not for its help.
//
// Each of the three is something only the font can answer. How a composite
// glyph is assembled is not among them: that is written in the glyph record, so
// a source that hands over the record has already said it.
type Source interface {
	// Table returns the bytes of a four-character named table, nil if absent.
	Table(tag string) []byte
	// NumGlyphs is the glyph count the font declares in maxp.
	NumGlyphs() int
	// GlyphData returns a glyph's raw glyf record, nil if empty or out of range.
	GlyphData(glyph uint16) []byte
}

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
func (s *Subsetter) AppendSubset(dst []byte, f Source, gids []uint16) ([]byte, error) {
	// Outlines living in a glyf/loca pair is what makes a font subsettable by
	// copying glyph records; a CFF font's charstrings are not reachable this way.
	glyf, head, maxp := f.Table("glyf"), f.Table("head"), f.Table("maxp")
	if len(glyf) == 0 || len(f.Table("loca")) == 0 {
		return dst, ErrNoOutlines
	}
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
func (s *Subsetter) AppendClosure(dst []uint16, f Source, gids []uint16) []uint16 {
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
func (s *Subsetter) mark(f Source, numGlyphs int, gids []uint16, keepNotdef bool) {
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
		s.stack = appendComponents(s.stack, f.GlyphData(gid))
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

// Component record flags, from the glyf table's composite glyph description.
// Only the ones that change a record's length matter here.
const (
	compArgsAreWords    = 1 << 0
	compHaveScale       = 1 << 3
	compMoreComponents  = 1 << 5
	compHaveXYScale     = 1 << 6
	compHaveTwoByTwo    = 1 << 7
	compRecordHeaderLen = 4  // flags and glyph id
	compGlyphHeaderLen  = 10 // contour count and bounding box
)

// appendComponents appends the glyph ids the composite glyph record g is
// assembled from, without recursing: a component may itself be a composite. A
// simple or empty record appends nothing.
//
// Component records are variable length — the flags say whether the placement
// arguments are bytes or words and how much transform follows — so the walk
// stops at the first record the record's own bytes cannot hold.
func appendComponents(dst []uint16, g []byte) []uint16 {
	if len(g) < compGlyphHeaderLen || int16(be16(g, 0)) >= 0 {
		return dst // Empty, or a simple glyph with a non-negative contour count.
	}
	for p := compGlyphHeaderLen; p+compRecordHeaderLen <= len(g); {
		flags, comp := be16(g, p), be16(g, p+2)
		next := p + compRecordHeaderLen
		if flags&compArgsAreWords != 0 {
			next += 4
		} else {
			next += 2
		}
		if next > len(g) {
			break // The arguments run past the record: it names no component.
		}
		dst = append(dst, comp)
		switch {
		case flags&compHaveTwoByTwo != 0:
			next += 8
		case flags&compHaveXYScale != 0:
			next += 4
		case flags&compHaveScale != 0:
			next += 2
		}
		if flags&compMoreComponents == 0 {
			break
		}
		p = next
	}
	return dst
}

func be16(b []byte, i int) uint16 { return uint16(b[i])<<8 | uint16(b[i+1]) }

// hasTag reports whether tables already carries the named table.
func hasTag(tables []Table, tag string) bool {
	for _, t := range tables {
		if t.Tag == tag {
			return true
		}
	}
	return false
}
