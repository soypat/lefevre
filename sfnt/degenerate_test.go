package sfnt_test

import (
	"testing"

	"github.com/soypat/lefevre"
	"github.com/soypat/lefevre/sfnt"
)

// buildFont assembles the four tables an outline walk needs around a list of
// glyph records, so a test can hand the parser a shape no shipped font has.
// Glyph ids are the indices of glyphs; an empty record is an empty glyph.
func buildFont(t *testing.T, glyphs [][]byte, extra ...sfnt.Table) *lefevre.Font {
	t.Helper()
	var glyf, loca []byte
	for _, g := range glyphs {
		loca = appendBE32(loca, uint32(len(glyf)))
		glyf = append(glyf, g...)
		for len(glyf)%4 != 0 {
			glyf = append(glyf, 0)
		}
	}
	loca = appendBE32(loca, uint32(len(glyf)))

	head := make([]byte, 54)
	putBE32(head[0:], 0x00010000)  // version
	putBE32(head[12:], 0x5F0F3CF5) // magicNumber
	putBE16(head[18:], 1000)       // unitsPerEm
	putBE16(head[50:], 1)          // indexToLocFormat: long loca
	maxp := make([]byte, 6)
	putBE32(maxp[0:], 0x00010000)          // version
	putBE16(maxp[4:], uint16(len(glyphs))) // numGlyphs

	tables := append([]sfnt.Table{
		{Tag: "head", Data: head},
		{Tag: "maxp", Data: maxp},
		{Tag: "loca", Data: loca},
		{Tag: "glyf", Data: glyf},
	}, extra...)
	program, err := sfnt.Append(nil, tables)
	if err != nil {
		t.Fatalf("assembling the test font: %v", err)
	}
	f, err := lefevre.FontFromMemory(program, 0)
	if err != nil {
		t.Fatalf("parsing the test font: %v", err)
	}
	return f
}

func appendBE16(dst []byte, v uint16) []byte {
	return append(dst, byte(v>>8), byte(v))
}

func appendBE32(dst []byte, v uint32) []byte {
	return append(dst, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func putBE16(b []byte, v uint16) { b[0], b[1] = byte(v>>8), byte(v) }

func putBE32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v>>24), byte(v>>16), byte(v>>8), byte(v)
}

// simpleGlyph encodes a drawable simple glyph: nContours contours of two
// on-curve points each, which is the least the outline walker turns into
// segments at all — it drops any contour of under two points.
func simpleGlyph(nContours int) []byte {
	const pointsPerContour = 2
	total := nContours * pointsPerContour
	g := appendBE16(nil, uint16(int16(nContours)))
	for _, v := range []int16{0, 0, 100, 100} { // xMin, yMin, xMax, yMax
		g = appendBE16(g, uint16(v))
	}
	for c := range nContours {
		g = appendBE16(g, uint16((c+1)*pointsPerContour-1)) // endPtsOfContours
	}
	g = appendBE16(g, 0) // instructionLength
	for range total {
		// on-curve, x and y as positive one-byte deltas.
		g = append(g, 0x37)
	}
	for range total * 2 { // the x deltas, then the y deltas
		g = append(g, 10)
	}
	return g
}

// truncatedComposite encodes a composite declaring one component then stopping
// with MORE_COMPONENTS set. The record is 18 bytes: 2 of padding follow it.
func truncatedComposite(component uint16) []byte {
	const (
		argsAreWords   = 1 << 0
		argsAreXY      = 1 << 1
		moreComponents = 1 << 5
	)
	const composite = ^uint16(0) // numberOfContours: -1, meaning composite
	g := appendBE16(nil, composite)
	for _, v := range []int16{0, 0, 100, 100} {
		g = appendBE16(g, uint16(v))
	}
	g = appendBE16(g, argsAreWords|argsAreXY|moreComponents)
	g = appendBE16(g, component)
	g = appendBE16(g, 0) // argument1: x offset
	g = appendBE16(g, 0) // argument2: y offset
	return g
}

func sameSegments(a, b []lefevre.Segment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A record that ends mid-list ends the list; reading on decodes the next glyph's
// bytes as components. Here they decode as gid 3, which gid 1 never mentions.
func TestCompositeComponentsStopAtTheRecordEnd(t *testing.T) {
	// gid2 has three contours, so its record opens with 0x0003 — the bytes a
	// walker running past gid1's record reads as a component's glyph id.
	f := buildFont(t, [][]byte{
		nil,                   // gid0: .notdef, empty
		truncatedComposite(2), // gid1: declares gid2 and nothing more
		simpleGlyph(3),        // gid2: the declared component
		simpleGlyph(1),        // gid3: mentioned by nobody
	})

	if got := f.AppendGlyphComponents(nil, 1); len(got) != 1 || got[0] != 2 {
		t.Fatalf("AppendGlyphComponents(1) = %v, want [2]: the record declares one component", got)
	}
	// gid1 places gid2 at 0,0 and stops, so it draws exactly what gid2 draws.
	want := f.GlyphOutline(nil, 2)
	got := f.GlyphOutline(nil, 1)
	if len(want) == 0 {
		t.Fatal("gid2 draws nothing: the test font cannot show anything")
	}
	if !sameSegments(got, want) {
		t.Errorf("gid1 draws %d segments, want gid2's %d: the walk read past the record end",
			len(got), len(want))
	}
}

// The consequence: a subset keeps only the closure, so a glyph the renderer
// reaches but the closure never saw is dropped and a kept glyph draws differently.
func TestSubsetKeepsEveryGlyphAKeptGlyphDraws(t *testing.T) {
	src := buildFont(t, [][]byte{
		nil,
		truncatedComposite(2),
		simpleGlyph(3),
		simpleGlyph(1),
	})
	var s sfnt.Subsetter
	program, err := s.AppendSubset(nil, src, []uint16{1})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := lefevre.FontFromMemory(program, 0)
	if err != nil {
		t.Fatalf("re-parsing the subset: %v", err)
	}
	want, got := src.GlyphOutline(nil, 1), sub.GlyphOutline(nil, 1)
	if !sameSegments(got, want) {
		t.Errorf("gid1 draws %d segments in the subset against %d in the source: "+
			"the subset dropped a glyph the source draws", len(got), len(want))
	}
}

// Subsetting down to glyphs that are all empty leaves nothing to put in glyf.
// The program that comes out still has to be the kind of font it went in as:
// a reader that asks what outlines it carries must not be told "none".
func TestSubsetOfEmptyGlyphsIsStillAGlyfFont(t *testing.T) {
	src := buildFont(t, [][]byte{
		nil,            // gid0: .notdef, empty
		nil,            // gid1: empty, the one asked for
		simpleGlyph(1), // gid2: keeps the source's own glyf non-empty
	})
	if got := src.OutlineFormat(); got != lefevre.OutlineGlyf {
		t.Fatalf("source OutlineFormat = %v, want OutlineGlyf", got)
	}
	var s sfnt.Subsetter
	program, err := s.AppendSubset(nil, src, []uint16{1})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := lefevre.FontFromMemory(program, 0)
	if err != nil {
		t.Fatalf("re-parsing the subset: %v", err)
	}
	if got := sub.OutlineFormat(); got != lefevre.OutlineGlyf {
		t.Errorf("subset OutlineFormat = %v, want OutlineGlyf: a subset of a glyf font is a glyf font", got)
	}
	// And it must still be subsettable a second time, which is what the format
	// report is for.
	if _, err := s.AppendSubset(nil, sub, []uint16{1}); err != nil {
		t.Errorf("subsetting the subset: %v", err)
	}
}

// A table can be present and empty. Carrying a zero-length CFF table is not
// carrying CFF outlines, and a font with neither those nor glyf has none.
func TestOutlineFormatIgnoresAnEmptyOutlineTable(t *testing.T) {
	head := make([]byte, 54)
	appendBE32(head[12:12], 0x5F0F3CF5)
	program, err := sfnt.Append(nil, []sfnt.Table{
		{Tag: "head", Data: head},
		{Tag: "maxp", Data: make([]byte, 6)},
		{Tag: "CFF ", Data: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := lefevre.FontFromMemory(program, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.OutlineFormat(); got != lefevre.OutlineNone {
		t.Errorf("OutlineFormat = %v, want OutlineNone: an empty CFF table holds no charstrings", got)
	}
}
