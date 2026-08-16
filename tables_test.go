package lefevre

import (
	"testing"
)

// directoryOf re-reads src's table directory independently of Font, so the
// tests check the accessors against the file rather than against themselves.
func directoryOf(t *testing.T, src []byte) map[string][]byte {
	t.Helper()
	if len(src) < 12 {
		t.Fatal("font too short")
	}
	n := int(readU16BE(src, 4))
	out := make(map[string][]byte, n)
	for i := range n {
		rec := 12 + i*16
		tag := string(src[rec : rec+4])
		off, length := int(readU32BE(src, rec+8)), int(readU32BE(src, rec+12))
		if off+length > len(src) {
			continue
		}
		out[tag] = src[off : off+length]
	}
	return out
}

func TestTableReturnsTheDirectoryContents(t *testing.T) {
	data := loadTestFontData(t)
	f, err := FontFromMemory(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := directoryOf(t, data)
	if len(want) == 0 {
		t.Fatal("test font has no tables")
	}
	for tag, wantData := range want {
		got := f.Table(tag)
		if len(got) != len(wantData) {
			t.Errorf("Table(%q) length = %d, want %d", tag, len(got), len(wantData))
			continue
		}
		if len(got) > 0 && &got[0] != &wantData[0] {
			t.Errorf("Table(%q) does not alias the font data", tag)
		}
	}
	// A table the font does not carry, and a tag that is not a tag at all.
	for _, tag := range []string{"ZZZZ", "", "glyf!"} {
		if got := f.Table(tag); got != nil {
			t.Errorf("Table(%q) = %d bytes, want nil", tag, len(got))
		}
	}
}

// recordOffsetOf returns the position of the named table's offset field within
// the raw font, so a test can point a record somewhere the record may not go.
func recordOffsetOf(t *testing.T, src []byte, tag string) int {
	t.Helper()
	n := int(readU16BE(src, 4))
	for i := range n {
		rec := 12 + i*16
		if string(src[rec:rec+4]) == tag {
			return rec + 8
		}
	}
	t.Fatalf("font carries no %q table", tag)
	return 0
}

// A table's bytes live after the directory describing them, so such a record
// names no table — kb rejects these, and Table would hand out the directory.
func TestTableRejectsRecordsOverlappingTheDirectory(t *testing.T) {
	data := append([]byte(nil), loadTestFontData(t)...)
	// post is not needed to parse the rest, so the font stays loadable.
	rec := recordOffsetOf(t, data, "post")
	for _, off := range []uint32{0, 12, uint32(12 + 16*int(readU16BE(data, 4)) - 1)} {
		data[rec], data[rec+1] = byte(off>>24), byte(off>>16)
		data[rec+2], data[rec+3] = byte(off>>8), byte(off)
		f, err := FontFromMemory(data, 0)
		if err != nil {
			continue // refusing the whole font is an acceptable answer too
		}
		if got := f.Table("post"); got != nil {
			t.Errorf("post pointed at offset %d: Table returned %d bytes of the directory, want nil",
				off, len(got))
		}
	}
}

func TestTableReachesTablesTheParserDoesNotIndex(t *testing.T) {
	f := loadTestFont(t)
	// post is not one of the tables Font parses for shaping, and is exactly
	// the kind an embedder needs. cvt /fpgm/prep likewise.
	if len(f.Table("post")) < 32 {
		t.Errorf("Table(%q) = %d bytes, want a full post header", "post", len(f.Table("post")))
	}
}

func TestNumGlyphsMatchesMaxp(t *testing.T) {
	f := loadTestFont(t)
	maxp := f.Table("maxp")
	if len(maxp) < 6 {
		t.Fatal("font has no maxp")
	}
	want := int(readU16BE(maxp, 4))
	if got := f.NumGlyphs(); got != want {
		t.Errorf("NumGlyphs = %d, want %d", got, want)
	}
	if want == 0 {
		t.Fatal("font declares no glyphs")
	}
}

func TestNumGlyphsOnInvalidFont(t *testing.T) {
	var f *Font
	if got := f.NumGlyphs(); got != 0 {
		t.Errorf("NumGlyphs on nil font = %d, want 0", got)
	}
}

func TestGlyphDataIsTheOutlineRecord(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	if gid == 0 {
		t.Fatal("font has no 'A'")
	}
	g := f.GlyphData(gid)
	if len(g) < 10 {
		t.Fatalf("GlyphData(%d) = %d bytes, want an outline record", gid, len(g))
	}
	// The record's own header repeats the bounding box GlyphBounds reports.
	xMin, yMin, xMax, yMax := f.GlyphBounds(gid)
	if readS16BE(g, 2) != xMin || readS16BE(g, 4) != yMin ||
		readS16BE(g, 6) != xMax || readS16BE(g, 8) != yMax {
		t.Errorf("GlyphData header bbox = %d,%d,%d,%d, want %d,%d,%d,%d",
			readS16BE(g, 2), readS16BE(g, 4), readS16BE(g, 6), readS16BE(g, 8),
			xMin, yMin, xMax, yMax)
	}
	// A space has no outline, and neither does a glyph past the end.
	if got := f.GlyphData(f.GlyphID(' ')); got != nil {
		t.Errorf("GlyphData(space) = %d bytes, want nil", len(got))
	}
	if got := f.GlyphData(uint16(f.NumGlyphs() - 1 + 1)); got != nil {
		t.Errorf("GlyphData(out of range) = %d bytes, want nil", len(got))
	}
}

// A composite glyph draws its components: 'é' is 'e' plus an accent, so its
// outline carries every contour 'e' has and one more.
func TestGlyphOutlineFlattensComposite(t *testing.T) {
	f := loadTestFont(t)
	gid, e := f.GlyphID('é'), f.GlyphID('e')
	if gid == 0 || e == 0 {
		t.Skip("font has no 'é' or no 'e'")
	}
	g := f.GlyphData(gid)
	if len(g) < 2 || readS16BE(g, 0) >= 0 {
		t.Skip("'é' is not a composite in this font")
	}
	acute, base := contourCount(f.GlyphOutline(nil, gid)), contourCount(f.GlyphOutline(nil, e))
	if base == 0 {
		t.Fatal("GlyphOutline('e') has no contours")
	}
	if acute <= base {
		t.Errorf("GlyphOutline('é') has %d contours, 'e' has %d: the accent was not drawn", acute, base)
	}
}

func contourCount(segs []Segment) int {
	n := 0
	for _, s := range segs {
		if s.Op == SegmentMoveTo {
			n++
		}
	}
	return n
}

func TestOutlineFormat(t *testing.T) {
	f := loadTestFont(t)
	if got := f.OutlineFormat(); got != OutlineGlyf {
		t.Errorf("OutlineFormat = %v, want OutlineGlyf", got)
	}
	var nilFont *Font
	if got := nilFont.OutlineFormat(); got != OutlineNone {
		t.Errorf("OutlineFormat on nil font = %v, want OutlineNone", got)
	}
}

func TestFontInfoReadsPostTable(t *testing.T) {
	f := loadTestFont(t)
	post := f.Table("post")
	if len(post) < 32 {
		t.Skip("font has no post table")
	}
	var info FontInfo
	f.ReadInfo(&info)
	wantAngle := float64(int32(readU32BE(post, 4))) / 65536
	if info.ItalicAngle != wantAngle {
		t.Errorf("ItalicAngle = %v, want %v", info.ItalicAngle, wantAngle)
	}
	if want := readS16BE(post, 8); info.UnderlinePosition != want {
		t.Errorf("UnderlinePosition = %d, want %d", info.UnderlinePosition, want)
	}
	if want := readS16BE(post, 10); info.UnderlineThickness != want {
		t.Errorf("UnderlineThickness = %d, want %d", info.UnderlineThickness, want)
	}
	if want := readU32BE(post, 12) != 0; info.IsFixedPitch != want {
		t.Errorf("IsFixedPitch = %v, want %v", info.IsFixedPitch, want)
	}
	// DejaVu Sans is upright and proportional; the values above must agree.
	if info.ItalicAngle != 0 {
		t.Errorf("ItalicAngle = %v, want 0 for an upright face", info.ItalicAngle)
	}
	if info.IsFixedPitch {
		t.Error("IsFixedPitch = true, want false for a proportional face")
	}
}

func TestTableLookupDoesNotAllocate(t *testing.T) {
	f := loadTestFont(t)
	allocs := testing.AllocsPerRun(100, func() {
		_ = f.Table("glyf")
		_ = f.Table("post")
		_ = f.Table("ZZZZ")
		_ = f.NumGlyphs()
		_ = f.GlyphData(5)
	})
	if allocs != 0 {
		t.Errorf("table accessors allocate %v times per run, want 0", allocs)
	}
}
