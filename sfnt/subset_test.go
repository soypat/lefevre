package sfnt_test

import (
	"os"
	"slices"
	"testing"

	"github.com/soypat/lefevre"
	"github.com/soypat/lefevre/sfnt"
)

func loadFont(t testing.TB, name string) (*lefevre.Font, []byte) {
	t.Helper()
	data, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		t.Skipf("test font not available: %v", err)
	}
	f, err := lefevre.FontFromMemory(data, 0)
	if err != nil {
		t.Fatalf("FontFromMemory(%s): %v", name, err)
	}
	return f, data
}

// subset builds a subset of name holding the glyphs for text, and re-parses it.
func subset(t *testing.T, name, text string, s *sfnt.Subsetter) (src, sub *lefevre.Font, program []byte) {
	t.Helper()
	src, _ = loadFont(t, name)
	var gids []uint16
	for _, r := range text {
		gids = append(gids, src.GlyphID(r))
	}
	if s == nil {
		s = &sfnt.Subsetter{}
	}
	program, err := s.AppendSubset(nil, src, gids)
	if err != nil {
		t.Fatalf("AppendSubset: %v", err)
	}
	sub, err = lefevre.FontFromMemory(program, 0)
	if err != nil {
		t.Fatalf("re-parsing the subset: %v", err)
	}
	return src, sub, program
}

func TestSubsetKeepsRequestedGlyphsAtTheSameID(t *testing.T) {
	const text = "Hello, world!"
	src, sub, _ := subset(t, "DejaVuSans.ttf", text, nil)

	if got, want := sub.NumGlyphs(), src.NumGlyphs(); got != want {
		t.Errorf("subset declares %d glyphs, want %d: ids must not shift", got, want)
	}
	var wantSegs, gotSegs []lefevre.Segment
	for _, r := range text {
		gid := src.GlyphID(r)
		wx0, wy0, wx1, wy1 := src.GlyphBounds(gid)
		gx0, gy0, gx1, gy1 := sub.GlyphBounds(gid)
		if wx0 != gx0 || wy0 != gy0 || wx1 != gx1 || wy1 != gy1 {
			t.Errorf("bounds of %q (gid %d) = %d,%d,%d,%d, want %d,%d,%d,%d",
				r, gid, gx0, gy0, gx1, gy1, wx0, wy0, wx1, wy1)
		}
		wantSegs = src.GlyphOutline(wantSegs[:0], gid)
		gotSegs = sub.GlyphOutline(gotSegs[:0], gid)
		if len(gotSegs) != len(wantSegs) {
			t.Fatalf("outline of %q (gid %d) has %d segments, want %d", r, gid, len(gotSegs), len(wantSegs))
		}
		for i := range wantSegs {
			if gotSegs[i] != wantSegs[i] {
				t.Fatalf("outline of %q segment %d = %+v, want %+v", r, i, gotSegs[i], wantSegs[i])
			}
		}
		// Advances come from hmtx, which the subset carries verbatim.
		if got, want := sub.GlyphAdvance(gid), src.GlyphAdvance(gid); got != want {
			t.Errorf("advance of %q (gid %d) = %d, want %d", r, gid, got, want)
		}
	}
}

func TestSubsetDropsUnusedOutlines(t *testing.T) {
	src, sub, program := subset(t, "DejaVuSans.ttf", "AB", nil)

	kept := map[uint16]bool{0: true, src.GlyphID('A'): true, src.GlyphID('B'): true}
	dropped := 0
	for gid := range src.NumGlyphs() {
		if kept[uint16(gid)] || len(src.GlyphData(uint16(gid))) == 0 {
			continue
		}
		if got := sub.GlyphData(uint16(gid)); got != nil {
			t.Fatalf("gid %d was not requested but kept %d bytes of outline", gid, len(got))
		}
		dropped++
	}
	if dropped == 0 {
		t.Fatal("the test font has nothing to drop")
	}
	if srcGlyf := len(src.Table("glyf")); len(sub.Table("glyf")) >= srcGlyf/2 {
		t.Errorf("subset glyf is %d bytes against the source's %d: nothing was saved",
			len(sub.Table("glyf")), srcGlyf)
	}
	if _, srcData := loadFont(t, "DejaVuSans.ttf"); len(program) >= len(srcData) {
		t.Errorf("subset program is %d bytes, source is %d", len(program), len(srcData))
	}
}

func TestSubsetKeepsCompositeComponents(t *testing.T) {
	src, _ := loadFont(t, "DejaVuSans.ttf")
	acute := src.GlyphID('é')
	if acute == 0 {
		t.Skip("test font has no 'é'")
	}
	comps := src.AppendGlyphComponents(nil, acute)
	if len(comps) == 0 {
		t.Skip("'é' is not a composite in the test font")
	}
	_, sub, _ := subset(t, "DejaVuSans.ttf", "é", nil)
	for _, c := range comps {
		if len(src.GlyphData(c)) == 0 {
			continue // an empty component has nothing to keep
		}
		if sub.GlyphData(c) == nil {
			t.Errorf("component gid %d of 'é' was dropped: the composite will not draw", c)
		}
	}
}

func TestSubsetKeepsNotdef(t *testing.T) {
	src, sub, _ := subset(t, "DejaVuSans.ttf", "A", nil)
	if len(src.GlyphData(0)) == 0 {
		t.Skip("test font has an empty .notdef")
	}
	if sub.GlyphData(0) == nil {
		t.Error(".notdef was dropped: an unmapped code has nothing to draw")
	}
}

func TestSubsetDropsCmapUnlessAsked(t *testing.T) {
	src, sub, _ := subset(t, "DejaVuSans.ttf", "A", nil)
	if sub.Table("cmap") != nil {
		t.Error("subset carries a cmap that was not asked for")
	}

	s := &sfnt.Subsetter{Keep: []string{"cmap"}}
	_, kept, _ := subset(t, "DejaVuSans.ttf", "A", s)
	if kept.Table("cmap") == nil {
		t.Fatal("Keep did not carry the cmap into the subset")
	}
	if got, want := kept.GlyphID('A'), src.GlyphID('A'); got != want {
		t.Errorf("GlyphID('A') on the kept-cmap subset = %d, want %d", got, want)
	}
}

func TestSubsetHasNoLeftoverTables(t *testing.T) {
	_, sub, _ := subset(t, "DejaVuSans.ttf", "A", nil)
	// A PDF supplies the encoding and the widths itself, so nothing that only
	// describes the source font belongs in the program it embeds.
	for _, tag := range []string{"cmap", "name", "post", "OS/2", "GSUB", "GPOS", "kern"} {
		if sub.Table(tag) != nil {
			t.Errorf("subset carries a %q table it was not asked for", tag)
		}
	}
	// Hinting travels with the outlines it hints.
	for _, tag := range []string{"glyf", "loca", "head", "maxp", "hhea", "hmtx"} {
		if sub.Table(tag) == nil {
			t.Errorf("subset is missing the required %q table", tag)
		}
	}
}

func TestSubsetChecksumsValidate(t *testing.T) {
	_, _, program := subset(t, "DejaVuSans.ttf", "Hello", nil)
	// head.checkSumAdjustment is chosen to make the whole file sum to this.
	if got := sfnt.Checksum(program); got != 0xB1B0AFBA {
		t.Errorf("whole-file checksum = %#08x, want 0xb1b0afba", got)
	}
	if len(program)%4 != 0 {
		t.Errorf("program is %d bytes, want a multiple of 4", len(program))
	}
}

func TestSubsetOfOpenSans(t *testing.T) {
	// A second face, to catch anything specific to DejaVu's table layout.
	src, sub, _ := subset(t, "OpenSans.ttf", "Quick brown fox", nil)
	for _, r := range "Quick brown fox" {
		gid := src.GlyphID(r)
		if len(src.GlyphData(gid)) == 0 {
			continue
		}
		if sub.GlyphData(gid) == nil {
			t.Errorf("gid %d for %q was dropped", gid, r)
		}
	}
}

func TestSubsetRejectsFontsWithoutGlyfOutlines(t *testing.T) {
	src, _ := loadFont(t, "DejaVuSans.ttf")
	if got := src.OutlineFormat(); got != lefevre.OutlineGlyf {
		t.Fatalf("OutlineFormat = %v, want OutlineGlyf", got)
	}
	var s sfnt.Subsetter
	// Nothing to copy glyph records out of: a subsetter must say so rather
	// than emit a program with an empty glyf.
	for _, f := range []*lefevre.Font{nil, {}} {
		if _, err := s.AppendSubset(nil, f, []uint16{1}); err == nil {
			t.Errorf("AppendSubset(%v) = nil error, want a failure", f)
		}
	}
}

func TestAppendClosureIsTransitive(t *testing.T) {
	src, _ := loadFont(t, "DejaVuSans.ttf")
	acute := src.GlyphID('é')
	if acute == 0 {
		t.Skip("test font has no 'é'")
	}
	var s sfnt.Subsetter
	// The same id twice: the closure is a set, not a list.
	got := s.AppendClosure(nil, src, []uint16{acute, acute, uint16(src.NumGlyphs())})
	if !slices.IsSorted(got) {
		t.Errorf("AppendClosure = %v, want ascending glyph ids", got)
	}
	want := append(src.AppendGlyphComponents(nil, acute), acute)
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("AppendClosure = %v, want it to include %d", got, w)
		}
	}
	for _, g := range got {
		if int(g) >= src.NumGlyphs() {
			t.Errorf("AppendClosure = %v, want no id past the font's %d glyphs", got, src.NumGlyphs())
		}
	}
	if n := len(slices.Compact(slices.Clone(got))); n != len(got) {
		t.Errorf("AppendClosure = %v, want each id once", got)
	}
}

func TestSubsetterReuseAllocatesNothing(t *testing.T) {
	src, _ := loadFont(t, "DejaVuSans.ttf")
	gids := make([]uint16, 0, 16)
	for _, r := range "The quick brown" {
		gids = append(gids, src.GlyphID(r))
	}
	var s sfnt.Subsetter
	dst, err := s.AppendSubset(nil, src, gids)
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(20, func() {
		dst, err = s.AppendSubset(dst[:0], src, gids)
	})
	if err != nil {
		t.Fatal(err)
	}
	if allocs != 0 {
		t.Errorf("a reused Subsetter allocates %v times per subset, want 0", allocs)
	}
}

func TestAppendRequiresHead(t *testing.T) {
	_, err := sfnt.Append(nil, []sfnt.Table{{Tag: "maxp", Data: make([]byte, 6)}})
	if err == nil {
		t.Error("Append without a head table = nil error, want a failure")
	}
}

func TestAppendSortsTheDirectory(t *testing.T) {
	head := make([]byte, 54)
	out, err := sfnt.Append(nil, []sfnt.Table{
		{Tag: "maxp", Data: make([]byte, 6)},
		{Tag: "head", Data: head},
		{Tag: "cvt ", Data: make([]byte, 4)},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The directory must be ascending by tag: "cvt " < "head" < "maxp".
	if n := int(out[4])<<8 | int(out[5]); n != 3 {
		t.Fatalf("directory declares %d tables, want 3", n)
	}
	prev := ""
	for i := range 3 {
		tag := string(out[12+16*i : 12+16*i+4])
		if tag <= prev {
			t.Errorf("table %d has tag %q, which does not follow %q", i, tag, prev)
		}
		prev = tag
	}
	if got := sfnt.Checksum(out); got != 0xB1B0AFBA {
		t.Errorf("whole-file checksum = %#08x, want 0xb1b0afba", got)
	}
}

func TestAppendPreservesDst(t *testing.T) {
	prefix := []byte{1, 2, 3}
	out, err := sfnt.Append(prefix, []sfnt.Table{
		{Tag: "head", Data: make([]byte, 54)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) <= 3 || out[0] != 1 || out[1] != 2 || out[2] != 3 {
		t.Errorf("Append overwrote dst: got % x", out[:min(len(out), 8)])
	}
}
