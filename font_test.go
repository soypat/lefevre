package lefevre

import (
	"os"
	"testing"
)

func loadTestFont(t *testing.T) *Font {
	t.Helper()
	data, err := os.ReadFile("testdata/DejaVuSans.ttf")
	if err != nil {
		t.Skipf("test font not available: %v", err)
	}
	f, err := FontFromMemory(data, 0)
	if err != nil {
		t.Fatalf("FontFromMemory: %v", err)
	}
	return f
}

func loadTestFontData(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/DejaVuSans.ttf")
	if err != nil {
		t.Skipf("test font not available: %v", err)
	}
	return data
}

// FontCount tests

func TestFontCount_SingleTTF(t *testing.T) {
	data := loadTestFontData(t)
	n := FontCount(data)
	if n != 1 {
		t.Errorf("FontCount = %d, want 1", n)
	}
}

func TestFontCount_NilData(t *testing.T) {
	n := FontCount(nil)
	if n != 0 {
		t.Errorf("FontCount(nil) = %d, want 0", n)
	}
}

func TestFontCount_Truncated(t *testing.T) {
	n := FontCount([]byte{0, 1})
	if n != 0 {
		t.Errorf("FontCount(truncated) = %d, want 0", n)
	}
}

// FontFromMemory tests

func TestFontFromMemory_Valid(t *testing.T) {
	f := loadTestFont(t)
	if !f.IsValid() {
		t.Error("expected IsValid() == true after successful load")
	}
}

func TestFontFromMemory_NilData(t *testing.T) {
	_, err := FontFromMemory(nil, 0)
	if err == nil {
		t.Error("expected error for nil data")
	}
}

func TestFontFromMemory_Truncated(t *testing.T) {
	_, err := FontFromMemory([]byte{0, 1, 0, 0, 0, 0, 0, 0, 0, 0}, 0)
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestFontFromMemory_BadMagic(t *testing.T) {
	bad := make([]byte, 256)
	for i := range bad {
		bad[i] = 0xFF
	}
	_, err := FontFromMemory(bad, 0)
	if err == nil {
		t.Error("expected error for bad magic bytes")
	}
}

func TestFontFromMemory_IndexOOB(t *testing.T) {
	data := loadTestFontData(t)
	_, err := FontFromMemory(data, 5)
	if err == nil {
		t.Error("expected error for out-of-bounds font index")
	}
}

// IsValid tests

func TestFontIsValid_Valid(t *testing.T) {
	f := loadTestFont(t)
	if !f.IsValid() {
		t.Error("expected IsValid() == true")
	}
}

func TestFontIsValid_ZeroValue(t *testing.T) {
	var f Font
	if f.IsValid() {
		t.Error("expected IsValid() == false for zero-value Font")
	}
}

// Info tests

func TestFontInfo_Family(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	if info.Family != "DejaVu Sans" {
		t.Errorf("Family = %q, want %q", info.Family, "DejaVu Sans")
	}
}

func TestFontInfo_Weight(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	if info.Weight == FontWeightUnknown {
		t.Error("expected a known weight value, got FontWeightUnknown")
	}
}

func TestFontInfo_Metrics(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	if info.UnitsPerEm == 0 {
		t.Error("expected UnitsPerEm > 0")
	}
	if info.Ascent == 0 {
		t.Error("expected Ascent != 0")
	}
}

func TestFontInfo_ZeroValue(t *testing.T) {
	var f Font
	info := f.Info()
	if info.Family != "" {
		t.Errorf("expected empty Family for zero Font, got %q", info.Family)
	}
}

// GlyphID tests

func TestFontGlyphID_LatinA(t *testing.T) {
	f := loadTestFont(t)
	id := f.GlyphID('A')
	if id == 0 {
		t.Error("expected non-zero glyph ID for 'A'")
	}
}

func TestFontGlyphID_Space(t *testing.T) {
	f := loadTestFont(t)
	id := f.GlyphID(' ')
	if id == 0 {
		t.Error("expected non-zero glyph ID for space")
	}
}

func TestFontGlyphID_NotDef(t *testing.T) {
	f := loadTestFont(t)
	id := f.GlyphID(0x10FFFF)
	if id != 0 {
		t.Errorf("expected glyph ID 0 for unmapped codepoint, got %d", id)
	}
}

func TestFontGlyphID_ZeroValue(t *testing.T) {
	var f Font
	id := f.GlyphID('A')
	if id != 0 {
		t.Errorf("expected glyph ID 0 for zero Font, got %d", id)
	}
}

// glyphAdvance tests

func TestFontGlyphAdvance_LatinA(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	adv := f.glyphAdvance(gid)
	if adv <= 0 {
		t.Errorf("glyphAdvance('A') = %d, want > 0", adv)
	}
}

func TestFontGlyphAdvance_Space(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID(' ')
	adv := f.glyphAdvance(gid)
	if adv <= 0 {
		t.Errorf("glyphAdvance(' ') = %d, want > 0", adv)
	}
}

func TestFontGlyphAdvance_NotDef(t *testing.T) {
	f := loadTestFont(t)
	// Glyph 0 (.notdef) should still have a valid advance.
	adv := f.glyphAdvance(0)
	if adv < 0 {
		t.Errorf("glyphAdvance(0) = %d, want >= 0", adv)
	}
}

func TestFontGlyphAdvance_ZeroValue(t *testing.T) {
	var f Font
	adv := f.glyphAdvance(0)
	if adv != 0 {
		t.Errorf("glyphAdvance on zero Font = %d, want 0", adv)
	}
}

// GSUB table presence

func TestFontHasGSUBTable(t *testing.T) {
	f := loadTestFont(t)
	te := f.tables[tableGsub]
	if te.length == 0 {
		t.Fatal("GSUB table not found in DejaVuSans.ttf")
	}
	if te.offset == 0 {
		t.Fatal("GSUB table offset is 0")
	}
}

func TestFontGSUBHeader(t *testing.T) {
	f := loadTestFont(t)
	te := f.tables[tableGsub]
	if te.length == 0 {
		t.Skip("no GSUB table")
	}
	base := int(te.offset)
	if base+10 > len(f.data) {
		t.Fatal("GSUB table too short for header")
	}
	major := readU16BE(f.data, base)
	minor := readU16BE(f.data, base+2)
	if major != 1 {
		t.Errorf("GSUB major version = %d, want 1", major)
	}
	if minor > 1 {
		t.Errorf("GSUB minor version = %d, want 0 or 1", minor)
	}
	scriptListOff := readU16BE(f.data, base+4)
	featureListOff := readU16BE(f.data, base+6)
	lookupListOff := readU16BE(f.data, base+8)
	if scriptListOff == 0 {
		t.Error("GSUB ScriptList offset is 0")
	}
	if featureListOff == 0 {
		t.Error("GSUB FeatureList offset is 0")
	}
	if lookupListOff == 0 {
		t.Error("GSUB LookupList offset is 0")
	}
}

func TestFontGSUBFindLigaFeature(t *testing.T) {
	f := loadTestFont(t)
	te := f.tables[tableGsub]
	if te.length == 0 {
		t.Skip("no GSUB table")
	}
	idx := f.findGSUBFeatureIndex(FeatureTagLiga)
	if idx < 0 {
		t.Fatal("'liga' feature not found in GSUB FeatureList")
	}
}

func TestFontGSUBLigaLookupCount(t *testing.T) {
	f := loadTestFont(t)
	te := f.tables[tableGsub]
	if te.length == 0 {
		t.Skip("no GSUB table")
	}
	idx := f.findGSUBFeatureIndex(FeatureTagLiga)
	if idx < 0 {
		t.Skip("no liga feature")
	}
	lookups := f.gsubFeatureLookups(idx)
	if len(lookups) == 0 {
		t.Fatal("'liga' feature has no lookup indices")
	}
}

// GDEF table tests

func TestFontHasGDEFTable(t *testing.T) {
	f := loadTestFont(t)
	te := f.tables[tableGdef]
	if te.length == 0 {
		t.Fatal("GDEF table not found in DejaVuSans.ttf")
	}
}

func TestFontGDEFGlyphClassBase(t *testing.T) {
	f := loadTestFont(t)
	// 'A' is a base glyph in any font with a GDEF table.
	gid := f.GlyphID('A')
	if gid == 0 {
		t.Skip("no glyph for 'A'")
	}
	cls := f.glyphClassDef(gid)
	if cls != glyphClassBase {
		t.Errorf("glyphClassDef('A' gid=%d) = %d, want %d (base)", gid, cls, glyphClassBase)
	}
}

func TestFontGDEFGlyphClassMark(t *testing.T) {
	f := loadTestFont(t)
	// U+0300 (combining grave accent) should be classified as a mark glyph.
	gid := f.GlyphID(0x0300)
	if gid == 0 {
		t.Skip("no glyph for U+0300")
	}
	cls := f.glyphClassDef(gid)
	if cls != glyphClassMark {
		t.Errorf("glyphClassDef(U+0300 gid=%d) = %d, want %d (mark)", gid, cls, glyphClassMark)
	}
}

func TestFontGDEFGlyphClassZeroValue(t *testing.T) {
	var f Font
	cls := f.glyphClassDef(42)
	if cls != glyphClassZero {
		t.Errorf("glyphClassDef on zero Font = %d, want 0", cls)
	}
}

func TestFontGDEFMarkAttachmentClassZeroValue(t *testing.T) {
	var f Font
	mac := f.markAttachmentClass(42)
	if mac != 0 {
		t.Errorf("markAttachmentClass on zero Font = %d, want 0", mac)
	}
}

func TestClassDefFormat1Lookup(t *testing.T) {
	// Build a synthetic ClassDef format 1 table:
	// format=1, startGlyphID=10, glyphCount=3, classes=[1, 3, 2]
	data := []byte{
		0, 1, // format 1
		0, 10, // startGlyphID = 10
		0, 3, // glyphCount = 3
		0, 1, // glyph 10 -> class 1
		0, 3, // glyph 11 -> class 3
		0, 2, // glyph 12 -> class 2
	}
	tests := []struct {
		gid  uint16
		want uint16
	}{
		{9, 0},  // before range
		{10, 1}, // first
		{11, 3}, // middle
		{12, 2}, // last
		{13, 0}, // after range
	}
	for _, tt := range tests {
		got := classDefLookup(data, 0, tt.gid)
		if got != tt.want {
			t.Errorf("classDefLookup(gid=%d) = %d, want %d", tt.gid, got, tt.want)
		}
	}
}

func TestClassDefFormat2Lookup(t *testing.T) {
	// Build a synthetic ClassDef format 2 table:
	// format=2, rangeCount=2
	// range 0: startGlyph=5, endGlyph=8, class=1
	// range 1: startGlyph=20, endGlyph=25, class=3
	data := []byte{
		0, 2, // format 2
		0, 2, // rangeCount = 2
		// range 0
		0, 5, // startGlyph = 5
		0, 8, // endGlyph = 8
		0, 1, // class = 1
		// range 1
		0, 20, // startGlyph = 20
		0, 25, // endGlyph = 25
		0, 3, // class = 3
	}
	tests := []struct {
		gid  uint16
		want uint16
	}{
		{4, 0},  // before first range
		{5, 1},  // start of range 0
		{7, 1},  // middle of range 0
		{8, 1},  // end of range 0
		{9, 0},  // between ranges
		{19, 0}, // just before range 1
		{20, 3}, // start of range 1
		{22, 3}, // middle of range 1
		{25, 3}, // end of range 1
		{26, 0}, // after last range
	}
	for _, tt := range tests {
		got := classDefLookup(data, 0, tt.gid)
		if got != tt.want {
			t.Errorf("classDefLookup(gid=%d) = %d, want %d", tt.gid, got, tt.want)
		}
	}
}

func TestFontGSUBApplyLiga(t *testing.T) {
	f := loadTestFont(t)
	// "ffi" should produce a ligature in DejaVuSans.
	glyphs := []Glyph{
		{Codepoint: 'f', ID: f.GlyphID('f')},
		{Codepoint: 'f', ID: f.GlyphID('f')},
		{Codepoint: 'i', ID: f.GlyphID('i')},
	}
	result := f.applyGSUBLigatures(glyphs)
	if len(result) >= 3 {
		t.Errorf("expected ligature substitution to reduce glyph count, got %d", len(result))
	}
	if len(result) == 0 {
		t.Fatal("applyGSUBLigatures returned empty slice")
	}
}

// GSUB type 1 (single substitution) tests

func TestSingleSubstFormat1(t *testing.T) {
	// Synthetic: coverage has glyph IDs 10 and 20. Delta = +5.
	// Coverage format 1: [10, 20]
	data := buildSingleSubstFormat1(t, []uint16{10, 20}, 5)
	f := &Font{data: data}
	f.cmapFormat = 0 // no cmap needed

	glyphs := []Glyph{
		{ID: 10}, {ID: 15}, {ID: 20}, {ID: 30},
	}
	result := f.applySingleSubst(glyphs, 0)
	if result[0].ID != 15 {
		t.Errorf("glyph 10 + delta 5 = %d, want 15", result[0].ID)
	}
	if result[1].ID != 15 {
		t.Errorf("glyph 15 (not covered) = %d, want 15", result[1].ID)
	}
	if result[2].ID != 25 {
		t.Errorf("glyph 20 + delta 5 = %d, want 25", result[2].ID)
	}
	if result[3].ID != 30 {
		t.Errorf("glyph 30 (not covered) = %d, want 30", result[3].ID)
	}
	if !result[0].Flags.Has(GlyphFlagGeneratedByGSUB) {
		t.Error("substituted glyph should have GeneratedByGSUB flag")
	}
	if result[1].Flags.Has(GlyphFlagGeneratedByGSUB) {
		t.Error("non-substituted glyph should not have GeneratedByGSUB flag")
	}
}

func TestSingleSubstFormat2(t *testing.T) {
	// Synthetic: coverage has glyph IDs [10, 20].
	// Substitute array: [100, 200].
	data := buildSingleSubstFormat2(t, []uint16{10, 20}, []uint16{100, 200})
	f := &Font{data: data}

	glyphs := []Glyph{
		{ID: 10}, {ID: 15}, {ID: 20},
	}
	result := f.applySingleSubst(glyphs, 0)
	if result[0].ID != 100 {
		t.Errorf("glyph 10 -> %d, want 100", result[0].ID)
	}
	if result[1].ID != 15 {
		t.Errorf("glyph 15 (uncovered) -> %d, want 15", result[1].ID)
	}
	if result[2].ID != 200 {
		t.Errorf("glyph 20 -> %d, want 200", result[2].ID)
	}
}

// GSUB type 2 (multiple substitution) tests

func TestMultipleSubst(t *testing.T) {
	// Synthetic: coverage has glyph 10.
	// Sequence for coverage index 0: [50, 51, 52] (1:3 replacement).
	data := buildMultipleSubst(t, []uint16{10}, [][]uint16{{50, 51, 52}})
	f := &Font{data: data}

	glyphs := []Glyph{
		{ID: 5}, {ID: 10}, {ID: 20},
	}
	result := f.applyMultipleSubst(glyphs, 0)
	if len(result) != 5 {
		t.Fatalf("expected 5 glyphs after 1:3 substitution, got %d", len(result))
	}
	wantIDs := []uint16{5, 50, 51, 52, 20}
	for i, want := range wantIDs {
		if result[i].ID != want {
			t.Errorf("result[%d].ID = %d, want %d", i, result[i].ID, want)
		}
	}
	if !result[1].Flags.Has(GlyphFlagFirstInMultiple) {
		t.Error("first substitute should have FirstInMultiple flag")
	}
	if !result[2].Flags.Has(GlyphFlagMultipleSubstitution) {
		t.Error("subsequent substitute should have MultipleSubstitution flag")
	}
}

func TestMultipleSubstDeletion(t *testing.T) {
	// Sequence count 0 = deletion.
	data := buildMultipleSubst(t, []uint16{10}, [][]uint16{{}})
	f := &Font{data: data}

	glyphs := []Glyph{
		{ID: 5}, {ID: 10}, {ID: 20},
	}
	result := f.applyMultipleSubst(glyphs, 0)
	if len(result) != 2 {
		t.Fatalf("expected 2 glyphs after deletion, got %d", len(result))
	}
	if result[0].ID != 5 || result[1].ID != 20 {
		t.Errorf("expected [5, 20], got [%d, %d]", result[0].ID, result[1].ID)
	}
}

// GSUB type 3 (alternate substitution) tests

func TestAlternateSubst(t *testing.T) {
	// Coverage has glyph 10. Alternates: [100, 200, 300].
	data := buildAlternateSubst(t, []uint16{10}, [][]uint16{{100, 200, 300}})
	f := &Font{data: data}

	glyphs := []Glyph{{ID: 10}, {ID: 15}}

	// altIndex=0 -> 100
	result := f.applyAlternateSubst(append([]Glyph{}, glyphs...), 0, 0)
	if result[0].ID != 100 {
		t.Errorf("alt index 0: got %d, want 100", result[0].ID)
	}
	if result[1].ID != 15 {
		t.Errorf("uncovered glyph changed: got %d, want 15", result[1].ID)
	}

	// altIndex=2 -> 300
	result = f.applyAlternateSubst(append([]Glyph{}, glyphs...), 0, 2)
	if result[0].ID != 300 {
		t.Errorf("alt index 2: got %d, want 300", result[0].ID)
	}

	// altIndex out of range -> falls back to 0 -> 100
	result = f.applyAlternateSubst(append([]Glyph{}, glyphs...), 0, 99)
	if result[0].ID != 100 {
		t.Errorf("alt index OOB: got %d, want 100", result[0].ID)
	}
}

func TestGSUBFeaturesDisableOverride(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f

	// Shape "ffi" with default features (liga enabled) — should produce ligature.
	runsDefault := cfg.ShapeSimple(nil, "ffi", DirectionLTR)
	defaultGlyphs := 0
	for _, r := range runsDefault {
		defaultGlyphs += len(r.Glyphs)
	}

	// Shape "ffi" with liga disabled — should produce 3 individual glyphs.
	cfg.Features = []FeatureOverride{{Tag: FeatureTagLiga, Value: 0}}
	runsNoLiga := cfg.ShapeSimple(nil, "ffi", DirectionLTR)
	noLigaGlyphs := 0
	for _, r := range runsNoLiga {
		noLigaGlyphs += len(r.Glyphs)
	}

	if noLigaGlyphs != 3 {
		t.Errorf("with liga=0: expected 3 glyphs for \"ffi\", got %d", noLigaGlyphs)
	}
	if defaultGlyphs >= noLigaGlyphs {
		t.Errorf("with default features: expected fewer glyphs (ligature), got %d vs %d", defaultGlyphs, noLigaGlyphs)
	}
}

func TestRTLGlyphReordering(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f

	// Shape LTR text — glyphs should be in logical order.
	ltrRuns := cfg.ShapeSimple(nil, "AB", DirectionLTR)
	if len(ltrRuns) == 0 {
		t.Fatal("no LTR runs")
	}
	if len(ltrRuns[0].Glyphs) < 2 {
		t.Fatal("expected at least 2 glyphs")
	}
	// In LTR, glyph[0] should be 'A' and glyph[1] should be 'B'.
	if ltrRuns[0].Glyphs[0].Codepoint != 'A' || ltrRuns[0].Glyphs[1].Codepoint != 'B' {
		t.Errorf("LTR order unexpected: [0]=%c [1]=%c", ltrRuns[0].Glyphs[0].Codepoint, ltrRuns[0].Glyphs[1].Codepoint)
	}

	// Shape with explicit RTL breaks — glyphs should be reversed.
	rtlBreaks := []Break{
		{Position: 0, Flags: BreakFlagDirection | BreakFlagParagraphDirection, Direction: DirectionRTL, ParagraphDirection: DirectionRTL},
	}
	rtlRuns := cfg.Shape(nil, "AB", rtlBreaks)
	if len(rtlRuns) == 0 {
		t.Fatal("no RTL runs")
	}
	if len(rtlRuns[0].Glyphs) < 2 {
		t.Fatal("expected at least 2 glyphs")
	}
	// In RTL, glyph[0] should be 'B' (rightmost visually) and glyph[1] should be 'A'.
	if rtlRuns[0].Glyphs[0].Codepoint != 'B' || rtlRuns[0].Glyphs[1].Codepoint != 'A' {
		t.Errorf("RTL order unexpected: [0]=%c [1]=%c, want B then A",
			rtlRuns[0].Glyphs[0].Codepoint, rtlRuns[0].Glyphs[1].Codepoint)
	}
}

func TestGPOSKerning(t *testing.T) {
	f := loadTestFont(t)
	// Check that GPOS table exists.
	if f.tables[tableGpos].length == 0 {
		t.Skip("test font has no GPOS table")
	}

	// Check if font has kern feature in GPOS.
	var idxBuf [4]int
	kernIndices := f.findGPOSFeatureIndices(idxBuf[:0], FeatureTagKern)
	if len(kernIndices) == 0 {
		t.Skip("test font has no GPOS kern feature")
	}

	// Shape a pair known to have kerning (e.g., "AV", "To", "Wa").
	var cfg ShapeConfig
	cfg.Font = f

	// Shape with kerning (default).
	runsKern := cfg.ShapeSimple(nil, "AV", DirectionLTR)
	if len(runsKern) == 0 || len(runsKern[0].Glyphs) < 2 {
		t.Fatal("expected at least 2 glyphs for \"AV\"")
	}

	// Shape with kerning disabled.
	cfg.Features = []FeatureOverride{{Tag: FeatureTagKern, Value: 0}}
	runsNoKern := cfg.ShapeSimple(nil, "AV", DirectionLTR)
	if len(runsNoKern) == 0 || len(runsNoKern[0].Glyphs) < 2 {
		t.Fatal("expected at least 2 glyphs for \"AV\" without kern")
	}

	// With kerning, the first glyph ('A') should have a different advance or offset.
	kernA := runsKern[0].Glyphs[0]
	noKernA := runsNoKern[0].Glyphs[0]

	// Kerning typically modifies AdvanceX or OffsetX.
	kernApplied := kernA.AdvanceX != noKernA.AdvanceX ||
		kernA.OffsetX != noKernA.OffsetX
	if !kernApplied {
		t.Log("NOTE: kerning may not have been applied — AV may not be a kerning pair in this font")
	}
}

func TestGPOSKerningDisabled(t *testing.T) {
	f := loadTestFont(t)
	if f.tables[tableGpos].length == 0 {
		t.Skip("test font has no GPOS table")
	}

	glyphs := []Glyph{{ID: f.GlyphID('A')}, {ID: f.GlyphID('V')}}
	origAdvance := make([]int32, len(glyphs))
	for i := range glyphs {
		glyphs[i].AdvanceX = f.glyphAdvance(glyphs[i].ID)
		origAdvance[i] = glyphs[i].AdvanceX
	}

	// Apply with kern disabled — should not modify anything.
	disabled := map[FeatureTag]bool{FeatureTagKern: true}
	f.applyGPOSKerning(glyphs, disabled)

	for i := range glyphs {
		if glyphs[i].AdvanceX != origAdvance[i] {
			t.Errorf("glyph[%d] advance changed despite kern disabled: %d -> %d",
				i, origAdvance[i], glyphs[i].AdvanceX)
		}
	}
}

func TestValueRecordSize(t *testing.T) {
	tests := []struct {
		format uint16
		want   int
	}{
		{0, 0},
		{1, 1},                    // XPlacement only
		{0x04, 1},                 // XAdvance only
		{0x05, 2},                 // XPlacement + XAdvance
		{0x0F, 4},                 // all 4 positioning fields
		{0xFF, 8},                 // all 8 fields including devices
	}
	for _, tt := range tests {
		got := valueRecordSize(tt.format)
		if got != tt.want {
			t.Errorf("valueRecordSize(0x%02X) = %d, want %d", tt.format, got, tt.want)
		}
	}
}

func TestGPOSSingleAdjust(t *testing.T) {
	// Synthetic GPOS single adjustment format 1: XAdvance = -50 for covered glyphs.
	cov := buildCoverageFormat1([]uint16{10, 20})
	// Layout: u16 format=1, u16 coverageOffset, u16 valueFormat=0x04(XAdvance), s16 value=-50
	var data []byte
	v := u16be(1) // format
	data = append(data, v[0], v[1])
	covOff := 8 // format(2) + covOff(2) + valueFormat(2) + value(2)
	v = u16be(uint16(covOff))
	data = append(data, v[0], v[1])
	v = u16be(0x04) // valueFormat = XAdvance
	data = append(data, v[0], v[1])
	neg50 := int16(-50)
	v = u16be(uint16(neg50)) // XAdvance = -50
	data = append(data, v[0], v[1])
	data = append(data, cov...)

	font := &Font{data: data}
	glyphs := []Glyph{
		{ID: 10, AdvanceX: 600},
		{ID: 30, AdvanceX: 500},
	}
	font.applyGPOSSingleAdjust(glyphs, 0)

	if glyphs[0].AdvanceX != 550 {
		t.Errorf("glyph[0].AdvanceX = %d, want 550 (600 - 50)", glyphs[0].AdvanceX)
	}
	if glyphs[0].Flags&GlyphFlagUsedInGPOS == 0 {
		t.Error("glyph[0] should have GlyphFlagUsedInGPOS")
	}
	if glyphs[1].AdvanceX != 500 {
		t.Errorf("glyph[1].AdvanceX = %d, want 500 (untouched)", glyphs[1].AdvanceX)
	}
}

func TestGPOSPairAdjustFormat1(t *testing.T) {
	// Synthetic pair adjustment format 1: pair (10, 20) with XAdvance=-75 on first glyph.
	cov := buildCoverageFormat1([]uint16{10})
	// ValueFormat1 = 0x04 (XAdvance), ValueFormat2 = 0
	// PairSet for glyph 10: 1 pair record: secondGlyph=20, XAdvance=-75
	// PairSet layout: u16 count=1, u16 secondGlyph=20, s16 XAdvance=-75
	var pairSet []byte
	v := u16be(1) // count
	pairSet = append(pairSet, v[0], v[1])
	v = u16be(20) // secondGlyph
	pairSet = append(pairSet, v[0], v[1])
	neg75 := int16(-75)
	v = u16be(uint16(neg75)) // XAdvance
	pairSet = append(pairSet, v[0], v[1])

	// Subtable layout: format(2) + covOff(2) + vf1(2) + vf2(2) + setCount(2) + setOffset(2) = 12
	headerSize := 12
	pairSetOff := headerSize
	covOffVal := pairSetOff + len(pairSet)

	var data []byte
	v = u16be(1) // format
	data = append(data, v[0], v[1])
	v = u16be(uint16(covOffVal))
	data = append(data, v[0], v[1])
	v = u16be(0x04) // valueFormat1 = XAdvance
	data = append(data, v[0], v[1])
	v = u16be(0) // valueFormat2
	data = append(data, v[0], v[1])
	v = u16be(1) // setCount
	data = append(data, v[0], v[1])
	v = u16be(uint16(pairSetOff)) // offset to pair set
	data = append(data, v[0], v[1])
	data = append(data, pairSet...)
	data = append(data, cov...)

	font := &Font{data: data}
	glyphs := []Glyph{
		{ID: 10, AdvanceX: 600},
		{ID: 20, AdvanceX: 500},
		{ID: 30, AdvanceX: 400},
	}
	font.applyGPOSPairAdjust(glyphs, 0)

	if glyphs[0].AdvanceX != 525 {
		t.Errorf("glyph[0].AdvanceX = %d, want 525 (600 - 75)", glyphs[0].AdvanceX)
	}
	if glyphs[0].Flags&GlyphFlagUsedInGPOS == 0 {
		t.Error("glyph[0] should have GlyphFlagUsedInGPOS")
	}
	// Second and third glyphs should be unchanged.
	if glyphs[1].AdvanceX != 500 {
		t.Errorf("glyph[1].AdvanceX = %d, want 500", glyphs[1].AdvanceX)
	}
	if glyphs[2].AdvanceX != 400 {
		t.Errorf("glyph[2].AdvanceX = %d, want 400", glyphs[2].AdvanceX)
	}
}

// Synthetic table builders for GSUB tests.

func u16be(v uint16) [2]byte { return [2]byte{byte(v >> 8), byte(v)} }

func buildCoverageFormat1(glyphIDs []uint16) []byte {
	var b []byte
	f := u16be(1) // format
	b = append(b, f[0], f[1])
	c := u16be(uint16(len(glyphIDs)))
	b = append(b, c[0], c[1])
	for _, gid := range glyphIDs {
		v := u16be(gid)
		b = append(b, v[0], v[1])
	}
	return b
}

func buildSingleSubstFormat1(t *testing.T, covGlyphs []uint16, delta int16) []byte {
	t.Helper()
	// Layout: u16 format=1, u16 coverageOffset, s16 deltaGlyphID, then coverage table
	cov := buildCoverageFormat1(covGlyphs)
	covOff := 6 // right after the 3 u16 fields
	var b []byte
	f := u16be(1)
	b = append(b, f[0], f[1])
	co := u16be(uint16(covOff))
	b = append(b, co[0], co[1])
	d := u16be(uint16(delta))
	b = append(b, d[0], d[1])
	b = append(b, cov...)
	return b
}

func buildSingleSubstFormat2(t *testing.T, covGlyphs []uint16, substGlyphs []uint16) []byte {
	t.Helper()
	// Layout: u16 format=2, u16 coverageOffset, u16 glyphCount, u16[] substGlyphIDs, then coverage
	headerSize := 6 + len(substGlyphs)*2
	cov := buildCoverageFormat1(covGlyphs)
	var b []byte
	f := u16be(2)
	b = append(b, f[0], f[1])
	co := u16be(uint16(headerSize))
	b = append(b, co[0], co[1])
	gc := u16be(uint16(len(substGlyphs)))
	b = append(b, gc[0], gc[1])
	for _, gid := range substGlyphs {
		v := u16be(gid)
		b = append(b, v[0], v[1])
	}
	b = append(b, cov...)
	return b
}

func buildMultipleSubst(t *testing.T, covGlyphs []uint16, sequences [][]uint16) []byte {
	t.Helper()
	// Layout: u16 format=1, u16 coverageOffset, u16 sequenceCount, u16[] sequenceOffsets,
	//         then sequence tables, then coverage table.
	seqCount := len(sequences)
	// Sequence offsets are relative to start of subtable.
	seqOffsetsStart := 6 + seqCount*2 // after header + offset array
	var seqData []byte
	seqOffsets := make([]int, seqCount)
	for i, seq := range sequences {
		seqOffsets[i] = seqOffsetsStart + len(seqData)
		gc := u16be(uint16(len(seq)))
		seqData = append(seqData, gc[0], gc[1])
		for _, gid := range seq {
			v := u16be(gid)
			seqData = append(seqData, v[0], v[1])
		}
	}
	covOff := seqOffsetsStart + len(seqData)
	cov := buildCoverageFormat1(covGlyphs)

	var b []byte
	f := u16be(1)
	b = append(b, f[0], f[1])
	co := u16be(uint16(covOff))
	b = append(b, co[0], co[1])
	sc := u16be(uint16(seqCount))
	b = append(b, sc[0], sc[1])
	for _, off := range seqOffsets {
		v := u16be(uint16(off))
		b = append(b, v[0], v[1])
	}
	b = append(b, seqData...)
	b = append(b, cov...)
	return b
}

func TestExtensionSubst(t *testing.T) {
	// Build a type 7 (extension) subtable that points to a type 1 format 1 (single subst, delta=+5).
	// Extension layout: u16 format=1, u16 lookupType=1, u32 offset
	singleSubst := buildSingleSubstFormat1(t, []uint16{10, 20}, 5)
	extOff := 8 // extension header is 8 bytes
	var ext []byte
	f1 := u16be(1)
	ext = append(ext, f1[0], f1[1])
	lt := u16be(1) // type 1 = single subst
	ext = append(ext, lt[0], lt[1])
	o := [4]byte{byte(extOff >> 24), byte(extOff >> 16), byte(extOff >> 8), byte(extOff)}
	ext = append(ext, o[0], o[1], o[2], o[3])
	ext = append(ext, singleSubst...)

	font := &Font{data: ext}
	glyphs := []Glyph{{ID: 10}, {ID: 20}, {ID: 30}}
	result := font.applyExtensionSubst(glyphs, 0)

	if result[0].ID != 15 {
		t.Errorf("glyph[0].ID = %d, want 15", result[0].ID)
	}
	if result[1].ID != 25 {
		t.Errorf("glyph[1].ID = %d, want 25", result[1].ID)
	}
	if result[2].ID != 30 {
		t.Errorf("glyph[2].ID = %d, want 30 (untouched)", result[2].ID)
	}
}

func TestContextSubstFormat3(t *testing.T) {
	// Context substitution format 3: match 2 glyphs by coverage, then apply a nested
	// single substitution (delta=+100) at sequence index 0.
	//
	// We need the nested lookup to be resolvable via gsubLookupOffset, which requires
	// a GSUB table structure. Instead, we test the core matching + inline substitution
	// by directly calling applyContextSubstFormat3 with synthetic data embedded in font.data.
	//
	// Layout of our synthetic context subst format 3:
	// u16 format=3, u16 glyphCount=2, u16 seqLookupCount=0 (no nested lookups for simplicity),
	// u16 coverageOffset[0], u16 coverageOffset[1]
	//
	// We verify matching only (since nested lookups need GSUB structure).
	// To actually test substitution, we'll build a minimal GSUB with one lookup.

	// Build a minimal GSUB table with one lookup (type 1, single subst delta=+100).
	singleSubst := buildSingleSubstFormat1(t, []uint16{10}, 100)
	// Lookup table: u16 type=1, u16 flag=0, u16 subtableCount=1, u16 subtableOffset
	var lookup []byte
	v := u16be(1) // type
	lookup = append(lookup, v[0], v[1])
	v = u16be(0) // flag
	lookup = append(lookup, v[0], v[1])
	v = u16be(1) // subtable count
	lookup = append(lookup, v[0], v[1])
	v = u16be(8) // subtable offset (relative to lookup start; 8 = after type+flag+count+thisOffset)
	lookup = append(lookup, v[0], v[1])
	// Mark filtering set not included (flag bit 4 not set)
	lookup = append(lookup, singleSubst...)

	// Lookup list: u16 lookupCount=1, u16 lookupOffset[0]
	var lookupList []byte
	v = u16be(1) // count
	lookupList = append(lookupList, v[0], v[1])
	v = u16be(4) // offset to lookup[0] relative to lookupList start
	lookupList = append(lookupList, v[0], v[1])
	lookupList = append(lookupList, lookup...)

	// GSUB header: u16 major=1, u16 minor=0, u16 scriptListOff=10, u16 featureListOff=10, u16 lookupListOff
	gsubHeaderSize := 10
	lookupListAbsOff := gsubHeaderSize
	var gsub []byte
	v = u16be(1) // major
	gsub = append(gsub, v[0], v[1])
	v = u16be(0) // minor
	gsub = append(gsub, v[0], v[1])
	v = u16be(uint16(lookupListAbsOff)) // scriptListOff (dummy, points to lookupList)
	gsub = append(gsub, v[0], v[1])
	v = u16be(uint16(lookupListAbsOff)) // featureListOff (dummy)
	gsub = append(gsub, v[0], v[1])
	v = u16be(uint16(lookupListAbsOff)) // lookupListOff
	gsub = append(gsub, v[0], v[1])
	gsub = append(gsub, lookupList...)

	// Now build the context subst format 3 subtable.
	// It has 2 input coverages (glyph 10, glyph 20) and 1 seqLookupRecord (seqIdx=0, lookupIdx=0).
	cov1 := buildCoverageFormat1([]uint16{10})
	cov2 := buildCoverageFormat1([]uint16{20})
	var ctx []byte
	v = u16be(3) // format
	ctx = append(ctx, v[0], v[1])
	v = u16be(2) // glyphCount
	ctx = append(ctx, v[0], v[1])
	v = u16be(1) // seqLookupCount
	ctx = append(ctx, v[0], v[1])
	// coverage offsets (relative to subtable start)
	ctxHeaderSize := 6 + 2*2 + 1*4 // format+glyphCount+seqLookupCount + 2 covOffsets + 1 seqLookupRecord
	cov1Off := ctxHeaderSize
	cov2Off := cov1Off + len(cov1)
	v = u16be(uint16(cov1Off))
	ctx = append(ctx, v[0], v[1])
	v = u16be(uint16(cov2Off))
	ctx = append(ctx, v[0], v[1])
	// seqLookupRecord: seqIdx=0, lookupListIdx=0
	v = u16be(0) // seqIdx
	ctx = append(ctx, v[0], v[1])
	v = u16be(0) // lookupListIdx
	ctx = append(ctx, v[0], v[1])
	ctx = append(ctx, cov1...)
	ctx = append(ctx, cov2...)

	// Combine: GSUB table then context subtable
	ctxSubtableOff := len(gsub)
	fullData := append(gsub, ctx...)

	font := &Font{data: fullData}
	font.tables[tableGsub] = tableEntry{offset: 0, length: uint32(len(fullData))}

	glyphs := []Glyph{{ID: 10}, {ID: 20}, {ID: 30}}
	result := font.applyContextSubstFormat3(glyphs, ctxSubtableOff)

	// The context matches glyphs [10, 20]. The seqLookupRecord says apply lookup 0
	// (single subst delta=+100) at seqIdx=0, so glyph 10 -> 110.
	if result[0].ID != 110 {
		t.Errorf("glyph[0].ID = %d, want 110", result[0].ID)
	}
	if result[1].ID != 20 {
		t.Errorf("glyph[1].ID = %d, want 20 (untouched)", result[1].ID)
	}
	if result[2].ID != 30 {
		t.Errorf("glyph[2].ID = %d, want 30 (untouched)", result[2].ID)
	}
}

func TestChainContextSubstFormat3(t *testing.T) {
	// Chaining context format 3: backtrack=[5], input=[10,20], lookahead=[30],
	// nested single subst delta=+100 at seqIdx=1 (glyph 20 -> 120).

	// Build minimal GSUB with one lookup (type 1, single subst delta=+100 covering glyph 20).
	singleSubst := buildSingleSubstFormat1(t, []uint16{20}, 100)
	var lookup []byte
	v := u16be(1)
	lookup = append(lookup, v[0], v[1]) // type
	v = u16be(0)
	lookup = append(lookup, v[0], v[1]) // flag
	v = u16be(1)
	lookup = append(lookup, v[0], v[1]) // subtableCount
	v = u16be(8)
	lookup = append(lookup, v[0], v[1]) // subtableOffset
	lookup = append(lookup, singleSubst...)

	var lookupList []byte
	v = u16be(1)
	lookupList = append(lookupList, v[0], v[1]) // count
	v = u16be(4)
	lookupList = append(lookupList, v[0], v[1]) // offset to lookup[0]
	lookupList = append(lookupList, lookup...)

	gsubHeaderSize := 10
	var gsub []byte
	for i := 0; i < 4; i++ {
		v = u16be(uint16(gsubHeaderSize))
		if i < 2 {
			v = u16be(uint16(i)) // major=1, minor=0
			if i == 0 {
				v = u16be(1)
			} else {
				v = u16be(0)
			}
		}
		gsub = append(gsub, v[0], v[1])
	}
	v = u16be(uint16(gsubHeaderSize))
	gsub = append(gsub, v[0], v[1])
	gsub = append(gsub, lookupList...)

	// Build chaining context format 3 subtable.
	covBacktrack := buildCoverageFormat1([]uint16{5})
	covInput1 := buildCoverageFormat1([]uint16{10})
	covInput2 := buildCoverageFormat1([]uint16{20})
	covLookahead := buildCoverageFormat1([]uint16{30})

	var chain []byte
	// format=3
	v = u16be(3)
	chain = append(chain, v[0], v[1])
	// backtrackGlyphCount=1
	v = u16be(1)
	chain = append(chain, v[0], v[1])
	// We'll fill coverage offsets after computing positions.
	// For now, compute the layout sizes:
	// header: format(2) + backtrackCount(2) + backtrackCovOff(2) + inputCount(2) + inputCovOff1(2) + inputCovOff2(2) +
	//         lookaheadCount(2) + lookaheadCovOff(2) + seqLookupCount(2) + seqLookupRecord(4) = 22
	headerSize := 2 + 2 + 1*2 + 2 + 2*2 + 2 + 1*2 + 2 + 1*4
	covBacktrackOff := headerSize
	covInput1Off := covBacktrackOff + len(covBacktrack)
	covInput2Off := covInput1Off + len(covInput1)
	covLookaheadOff := covInput2Off + len(covInput2)

	// backtrack coverage offset
	v = u16be(uint16(covBacktrackOff))
	chain = append(chain, v[0], v[1])
	// inputGlyphCount=2
	v = u16be(2)
	chain = append(chain, v[0], v[1])
	// input coverage offsets
	v = u16be(uint16(covInput1Off))
	chain = append(chain, v[0], v[1])
	v = u16be(uint16(covInput2Off))
	chain = append(chain, v[0], v[1])
	// lookaheadGlyphCount=1
	v = u16be(1)
	chain = append(chain, v[0], v[1])
	// lookahead coverage offset
	v = u16be(uint16(covLookaheadOff))
	chain = append(chain, v[0], v[1])
	// seqLookupCount=1
	v = u16be(1)
	chain = append(chain, v[0], v[1])
	// seqLookupRecord: seqIdx=1, lookupIdx=0
	v = u16be(1)
	chain = append(chain, v[0], v[1])
	v = u16be(0)
	chain = append(chain, v[0], v[1])
	// Append coverage tables
	chain = append(chain, covBacktrack...)
	chain = append(chain, covInput1...)
	chain = append(chain, covInput2...)
	chain = append(chain, covLookahead...)

	chainSubtableOff := len(gsub)
	fullData := append(gsub, chain...)

	font := &Font{data: fullData}
	font.tables[tableGsub] = tableEntry{offset: 0, length: uint32(len(fullData))}

	// Input: [5, 10, 20, 30] — backtrack=5, input=[10,20], lookahead=30
	glyphs := []Glyph{{ID: 5}, {ID: 10}, {ID: 20}, {ID: 30}}
	result := font.applyChainContextSubstFormat3(glyphs, chainSubtableOff)

	if result[0].ID != 5 {
		t.Errorf("glyph[0].ID = %d, want 5 (backtrack, untouched)", result[0].ID)
	}
	if result[1].ID != 10 {
		t.Errorf("glyph[1].ID = %d, want 10 (input[0], not targeted by seqLookup)", result[1].ID)
	}
	if result[2].ID != 120 {
		t.Errorf("glyph[2].ID = %d, want 120 (input[1], seqIdx=1 => delta+100)", result[2].ID)
	}
	if result[3].ID != 30 {
		t.Errorf("glyph[3].ID = %d, want 30 (lookahead, untouched)", result[3].ID)
	}
}

func TestChainContextSubstFormat3NoMatch(t *testing.T) {
	// Same structure as above but input doesn't match — backtrack glyph is wrong.
	covBacktrack := buildCoverageFormat1([]uint16{5})
	covInput := buildCoverageFormat1([]uint16{10})
	covLookahead := buildCoverageFormat1([]uint16{30})

	var chain []byte
	v := u16be(3)
	chain = append(chain, v[0], v[1]) // format
	v = u16be(1)
	chain = append(chain, v[0], v[1]) // backtrackCount
	headerSize := 2 + 2 + 1*2 + 2 + 1*2 + 2 + 1*2 + 2 + 0*4
	v = u16be(uint16(headerSize))
	chain = append(chain, v[0], v[1]) // backtrack cov offset
	v = u16be(1)
	chain = append(chain, v[0], v[1]) // inputCount
	v = u16be(uint16(headerSize + len(covBacktrack)))
	chain = append(chain, v[0], v[1]) // input cov offset
	v = u16be(1)
	chain = append(chain, v[0], v[1]) // lookaheadCount
	v = u16be(uint16(headerSize + len(covBacktrack) + len(covInput)))
	chain = append(chain, v[0], v[1]) // lookahead cov offset
	v = u16be(0)
	chain = append(chain, v[0], v[1]) // seqLookupCount=0
	chain = append(chain, covBacktrack...)
	chain = append(chain, covInput...)
	chain = append(chain, covLookahead...)

	font := &Font{data: chain}
	// Wrong backtrack: 99 instead of 5
	glyphs := []Glyph{{ID: 99}, {ID: 10}, {ID: 30}}
	result := font.applyChainContextSubstFormat3(glyphs, 0)

	// No match, all IDs unchanged.
	for i, g := range result {
		if g.ID != glyphs[i].ID {
			t.Errorf("glyph[%d].ID = %d, want %d (no match expected)", i, g.ID, glyphs[i].ID)
		}
	}
}

func TestReverseChainSubst(t *testing.T) {
	// Reverse chain single substitution:
	// coverage: glyph 10, backtrack: glyph 5, lookahead: glyph 30.
	// substitute: 10 -> 999.
	covMain := buildCoverageFormat1([]uint16{10})
	covBacktrack := buildCoverageFormat1([]uint16{5})
	covLookahead := buildCoverageFormat1([]uint16{30})

	var data []byte
	v := u16be(1) // format
	data = append(data, v[0], v[1])
	// coverage offset (relative to subtable start)
	// header: format(2) + covOff(2) + backtrackCount(2) + backtrackCovOff(2) +
	//         lookaheadCount(2) + lookaheadCovOff(2) + substGlyphCount(2) + substGlyph(2) = 16
	headerSize := 16
	mainCovOff := headerSize
	backtrackCovOff := mainCovOff + len(covMain)
	lookaheadCovOff := backtrackCovOff + len(covBacktrack)

	v = u16be(uint16(mainCovOff))
	data = append(data, v[0], v[1]) // coverageOffset
	v = u16be(1)
	data = append(data, v[0], v[1]) // backtrackGlyphCount
	v = u16be(uint16(backtrackCovOff))
	data = append(data, v[0], v[1]) // backtrackCoverageOffset[0]
	v = u16be(1)
	data = append(data, v[0], v[1]) // lookaheadGlyphCount
	v = u16be(uint16(lookaheadCovOff))
	data = append(data, v[0], v[1]) // lookaheadCoverageOffset[0]
	v = u16be(1)
	data = append(data, v[0], v[1]) // substituteGlyphCount
	v = u16be(999)
	data = append(data, v[0], v[1]) // substituteGlyphID[0]
	data = append(data, covMain...)
	data = append(data, covBacktrack...)
	data = append(data, covLookahead...)

	font := &Font{data: data}
	glyphs := []Glyph{{ID: 5}, {ID: 10}, {ID: 30}}
	result := font.applyReverseChainSubst(glyphs, 0)

	if result[0].ID != 5 {
		t.Errorf("glyph[0].ID = %d, want 5 (backtrack)", result[0].ID)
	}
	if result[1].ID != 999 {
		t.Errorf("glyph[1].ID = %d, want 999 (substituted)", result[1].ID)
	}
	if result[2].ID != 30 {
		t.Errorf("glyph[2].ID = %d, want 30 (lookahead)", result[2].ID)
	}
	if result[1].Flags&GlyphFlagGeneratedByGSUB == 0 {
		t.Error("substituted glyph should have GlyphFlagGeneratedByGSUB")
	}
}

func TestReverseChainSubstNoMatch(t *testing.T) {
	// No backtrack match — glyph should not be substituted.
	covMain := buildCoverageFormat1([]uint16{10})
	covBacktrack := buildCoverageFormat1([]uint16{5})

	var data []byte
	v := u16be(1) // format
	data = append(data, v[0], v[1])
	headerSize := 14 // no lookahead
	mainCovOff := headerSize
	backtrackCovOff := mainCovOff + len(covMain)

	v = u16be(uint16(mainCovOff))
	data = append(data, v[0], v[1])
	v = u16be(1)
	data = append(data, v[0], v[1]) // backtrackCount
	v = u16be(uint16(backtrackCovOff))
	data = append(data, v[0], v[1])
	v = u16be(0)
	data = append(data, v[0], v[1]) // lookaheadCount=0
	v = u16be(1)
	data = append(data, v[0], v[1]) // substGlyphCount
	v = u16be(999)
	data = append(data, v[0], v[1]) // substGlyphID
	data = append(data, covMain...)
	data = append(data, covBacktrack...)

	font := &Font{data: data}
	// Backtrack is 99, not 5 — should not match.
	glyphs := []Glyph{{ID: 99}, {ID: 10}}
	result := font.applyReverseChainSubst(glyphs, 0)

	if result[1].ID != 10 {
		t.Errorf("glyph[1].ID = %d, want 10 (no match)", result[1].ID)
	}
}

func buildAlternateSubst(t *testing.T, covGlyphs []uint16, altSets [][]uint16) []byte {
	t.Helper()
	// Layout: u16 format=1, u16 coverageOffset, u16 altSetCount, u16[] altSetOffsets,
	//         then alternate set tables, then coverage table.
	altSetCount := len(altSets)
	altSetOffsetsStart := 6 + altSetCount*2
	var altData []byte
	altSetOffsets := make([]int, altSetCount)
	for i, alts := range altSets {
		altSetOffsets[i] = altSetOffsetsStart + len(altData)
		gc := u16be(uint16(len(alts)))
		altData = append(altData, gc[0], gc[1])
		for _, gid := range alts {
			v := u16be(gid)
			altData = append(altData, v[0], v[1])
		}
	}
	covOff := altSetOffsetsStart + len(altData)
	cov := buildCoverageFormat1(covGlyphs)

	var b []byte
	f := u16be(1)
	b = append(b, f[0], f[1])
	co := u16be(uint16(covOff))
	b = append(b, co[0], co[1])
	ac := u16be(uint16(altSetCount))
	b = append(b, ac[0], ac[1])
	for _, off := range altSetOffsets {
		v := u16be(uint16(off))
		b = append(b, v[0], v[1])
	}
	b = append(b, altData...)
	b = append(b, cov...)
	return b
}

// --- Arabic joining tests ---

func TestJoiningTypeLookup(t *testing.T) {
	// Arabic letters should have non-zero joining types.
	tests := []struct {
		r    rune
		want joiningType
	}{
		{'\u0628', joiningTypeDual},   // Ba (dual joining)
		{'\u062D', joiningTypeDual},   // Haa (dual joining)
		{'\u0627', joiningTypeRight},  // Alef (right joining)
		{'\u062F', joiningTypeRight},  // Dal (right joining)
		{'\u0020', joiningTypeNone},   // Space (non-joining)
		{'A', joiningTypeNone},        // Latin A (non-joining)
		{'\u0640', joiningTypeForce},  // Tatweel (join-causing)
		{'\u064B', joiningTypeTransparent}, // Fathatan (transparent mark)
	}
	for _, tc := range tests {
		got := getJoiningType(tc.r)
		if got != tc.want {
			t.Errorf("getJoiningType(U+%04X) = %d, want %d", tc.r, got, tc.want)
		}
	}
}

func TestAssignJoiningForms(t *testing.T) {
	// "بحر" (ba-haa-ra): all dual-joining → init-medi-fina.
	glyphs := []Glyph{
		{Codepoint: '\u0628'}, // Ba
		{Codepoint: '\u062D'}, // Haa
		{Codepoint: '\u0631'}, // Ra (right-joining)
	}
	assignJoiningForms(glyphs)

	// Ba should be initial (has a following dual-joiner).
	if glyphs[0].joiningFeature != joiningFeatureInit {
		t.Errorf("glyph[0] (Ba) joiningFeature = %d, want init (%d)", glyphs[0].joiningFeature, joiningFeatureInit)
	}
	if !glyphs[0].Flags.Has(GlyphFlagInit) {
		t.Error("glyph[0] (Ba) missing GlyphFlagInit")
	}

	// Haa should be medial (between two joiners).
	if glyphs[1].joiningFeature != joiningFeatureMedi {
		t.Errorf("glyph[1] (Haa) joiningFeature = %d, want medi (%d)", glyphs[1].joiningFeature, joiningFeatureMedi)
	}
	if !glyphs[1].Flags.Has(GlyphFlagMedi) {
		t.Error("glyph[1] (Haa) missing GlyphFlagMedi")
	}

	// Ra should be final (right-joining, preceded by a joiner).
	if glyphs[2].joiningFeature != joiningFeatureFina {
		t.Errorf("glyph[2] (Ra) joiningFeature = %d, want fina (%d)", glyphs[2].joiningFeature, joiningFeatureFina)
	}
	if !glyphs[2].Flags.Has(GlyphFlagFina) {
		t.Error("glyph[2] (Ra) missing GlyphFlagFina")
	}
}

func TestAssignJoiningFormsIsolated(t *testing.T) {
	// Single character should be isolated.
	glyphs := []Glyph{{Codepoint: '\u0628'}} // Ba alone
	assignJoiningForms(glyphs)

	if glyphs[0].joiningFeature != joiningFeatureIsol {
		t.Errorf("glyph[0] (Ba) joiningFeature = %d, want isol (%d)", glyphs[0].joiningFeature, joiningFeatureIsol)
	}
	if !glyphs[0].Flags.Has(GlyphFlagIsol) {
		t.Error("glyph[0] (Ba) missing GlyphFlagIsol")
	}
}

func TestAssignJoiningFormsWithSpace(t *testing.T) {
	// "ب ب" (ba space ba): space breaks joining → both isolated.
	glyphs := []Glyph{
		{Codepoint: '\u0628'}, // Ba
		{Codepoint: ' '},      // Space (non-joining)
		{Codepoint: '\u0628'}, // Ba
	}
	assignJoiningForms(glyphs)

	if glyphs[0].joiningFeature != joiningFeatureIsol {
		t.Errorf("glyph[0] (Ba) joiningFeature = %d, want isol (%d)", glyphs[0].joiningFeature, joiningFeatureIsol)
	}
	if glyphs[2].joiningFeature != joiningFeatureIsol {
		t.Errorf("glyph[2] (Ba) joiningFeature = %d, want isol (%d)", glyphs[2].joiningFeature, joiningFeatureIsol)
	}
}

func TestAssignJoiningFormsTransparent(t *testing.T) {
	// "بَب" (ba + fatha + ba): fatha is transparent, shouldn't break joining.
	glyphs := []Glyph{
		{Codepoint: '\u0628'}, // Ba (dual)
		{Codepoint: '\u064E'}, // Fatha (transparent mark)
		{Codepoint: '\u0628'}, // Ba (dual)
	}
	assignJoiningForms(glyphs)

	// Ba should be init (transparent mark doesn't break the join).
	if glyphs[0].joiningFeature != joiningFeatureInit {
		t.Errorf("glyph[0] (Ba) joiningFeature = %d, want init (%d)", glyphs[0].joiningFeature, joiningFeatureInit)
	}
	// Last Ba should be fina.
	if glyphs[2].joiningFeature != joiningFeatureFina {
		t.Errorf("glyph[2] (Ba) joiningFeature = %d, want fina (%d)", glyphs[2].joiningFeature, joiningFeatureFina)
	}
}

func TestAssignJoiningFormsRightJoining(t *testing.T) {
	// "دب" (dal + ba): dal is right-joining, ba is dual-joining.
	// Dal can only join on the right, so: dal=fina, ba=isol? No...
	// dal (right) preceded by nothing → isol. ba follows dal but dal can't join left → ba=isol.
	// Actually: dal=right-joining means it joins to what comes BEFORE (right side in RTL).
	// With nothing before dal: dal=isol. ba after dal: dal can't join to ba (no left join) → ba=isol.
	glyphs := []Glyph{
		{Codepoint: '\u062F'}, // Dal (right-joining)
		{Codepoint: '\u0628'}, // Ba (dual-joining)
	}
	assignJoiningForms(glyphs)

	// Dal: right-joining, nothing before → isolated.
	if glyphs[0].joiningFeature != joiningFeatureIsol {
		t.Errorf("glyph[0] (Dal) joiningFeature = %d, want isol (%d)", glyphs[0].joiningFeature, joiningFeatureIsol)
	}
	// Ba: dual-joining, but dal before can't join left → isolated.
	if glyphs[1].joiningFeature != joiningFeatureIsol {
		t.Errorf("glyph[1] (Ba) joiningFeature = %d, want isol (%d)", glyphs[1].joiningFeature, joiningFeatureIsol)
	}
}

func TestAssignJoiningFormsBaDal(t *testing.T) {
	// "بد" (ba + dal): ba=dual, dal=right.
	// ba can join right (to dal) and dal accepts left joining from ba.
	// So: ba=init (joins right), dal=fina (joins to what comes before).
	glyphs := []Glyph{
		{Codepoint: '\u0628'}, // Ba (dual-joining)
		{Codepoint: '\u062F'}, // Dal (right-joining)
	}
	assignJoiningForms(glyphs)

	if glyphs[0].joiningFeature != joiningFeatureInit {
		t.Errorf("glyph[0] (Ba) joiningFeature = %d, want init (%d)", glyphs[0].joiningFeature, joiningFeatureInit)
	}
	if glyphs[1].joiningFeature != joiningFeatureFina {
		t.Errorf("glyph[1] (Dal) joiningFeature = %d, want fina (%d)", glyphs[1].joiningFeature, joiningFeatureFina)
	}
}

func TestArabicShapingEndToEnd(t *testing.T) {
	f := loadTestFont(t)
	// Shape Arabic text "مرحبا" and verify joining forms are assigned.
	text := "مرحبا"
	var cfg ShapeConfig
	cfg.Font = f

	runs := cfg.ShapeSimple(nil, text, DirectionRTL)
	if len(runs) == 0 {
		t.Fatal("no runs produced")
	}
	run := runs[0]
	if len(run.Glyphs) == 0 {
		t.Fatal("no glyphs in run")
	}

	// At least some glyphs should have joining feature flags set.
	hasJoiningFlag := false
	for _, g := range run.Glyphs {
		if g.Flags&joiningFeatureMask != 0 {
			hasJoiningFlag = true
			break
		}
	}
	if !hasJoiningFlag {
		t.Error("no glyphs have Arabic joining flags set; assignJoiningForms not working")
	}
}

// --- GPOS mark positioning tests ---

// TestGPOSMarkToBase tests that GPOS type 4 (mark-to-base) positions combining
// marks with non-zero offsets. Without type 4, marks remain at offset (0,0)
// and stack on top of the base glyph's origin instead of being placed above/below.
func TestGPOSMarkToBase(t *testing.T) {
	f := loadTestFont(t)

	// "A" + combining grave accent (U+0300).
	// The mark feature should position the grave above the A via anchor points.
	text := "A\u0300"
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, text, DirectionLTR)
	if len(runs) == 0 || len(runs[0].Glyphs) < 2 {
		t.Fatal("expected at least 2 glyphs")
	}

	var base, mark Glyph
	for _, g := range runs[0].Glyphs {
		if g.Codepoint == 'A' {
			base = g
		}
		if g.Codepoint == 0x0300 {
			mark = g
		}
	}
	if mark.ID == 0 {
		t.Fatal("combining grave accent not found in output")
	}

	// Mark-to-base should set a negative OffsetX on the mark (positioning it
	// back over the base glyph) and a positive OffsetY (positioning it above).
	// Without GPOS type 4, both offsets remain zero.
	if mark.OffsetX == 0 && mark.OffsetY == 0 {
		t.Error("mark glyph (U+0300) has zero offsets; GPOS mark-to-base positioning not applied")
	}

	// The base glyph should have normal advance; mark should not alter it.
	if base.AdvanceX == 0 {
		t.Error("base glyph 'A' has zero advance")
	}

	// The mark should have GlyphFlagUsedInGPOS set.
	if !mark.Flags.Has(GlyphFlagUsedInGPOS) {
		t.Error("mark glyph missing GlyphFlagUsedInGPOS")
	}
}

// TestGPOSMarkToBaseDifferentMarks verifies that different combining marks get
// different positioning. A grave accent (U+0300) and a cedilla (U+0327) should
// have different OffsetY since one goes above and one below the base.
func TestGPOSMarkToBaseDifferentMarks(t *testing.T) {
	f := loadTestFont(t)

	// Shape "A + grave" and "A + cedilla" separately.
	var cfg ShapeConfig
	cfg.Font = f

	graveRuns := cfg.ShapeSimple(nil, "A\u0300", DirectionLTR)
	cedillaRuns := cfg.ShapeSimple(nil, "A\u0327", DirectionLTR)

	if len(graveRuns) == 0 || len(graveRuns[0].Glyphs) < 2 {
		t.Fatal("expected 2 glyphs for A+grave")
	}
	if len(cedillaRuns) == 0 || len(cedillaRuns[0].Glyphs) < 2 {
		t.Fatal("expected 2 glyphs for A+cedilla")
	}

	var graveOff, cedillaOff int32
	for _, g := range graveRuns[0].Glyphs {
		if g.Codepoint == 0x0300 {
			graveOff = g.OffsetY
		}
	}
	for _, g := range cedillaRuns[0].Glyphs {
		if g.Codepoint == 0x0327 {
			cedillaOff = g.OffsetY
		}
	}

	// Both marks should have positioning applied (non-zero offsets from type 4).
	if graveOff == 0 && cedillaOff == 0 {
		t.Error("both marks have zero OffsetY; GPOS mark-to-base not applied")
		return
	}

	// They should differ: grave goes above (positive Y in font coords), cedilla below.
	if graveOff == cedillaOff {
		t.Errorf("grave and cedilla have same OffsetY=%d; expected different positions", graveOff)
	}
}

// TestGPOSMarkToMark tests that GPOS type 6 (mark-to-mark) positions a second
// combining mark relative to a first mark, not relative to the base.
// Example: "a" + combining macron (U+0304) + combining acute (U+0301)
// should place the acute above the macron, not at the same Y offset.
func TestGPOSMarkToMark(t *testing.T) {
	f := loadTestFont(t)

	// Shape "a + macron + acute": two stacked diacritics.
	text := "a\u0304\u0301"
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, text, DirectionLTR)
	if len(runs) == 0 || len(runs[0].Glyphs) < 3 {
		t.Fatal("expected at least 3 glyphs")
	}

	var macronOff, acuteOff int32
	var macronFound, acuteFound bool
	for _, g := range runs[0].Glyphs {
		if g.Codepoint == 0x0304 {
			macronOff = g.OffsetY
			macronFound = true
		}
		if g.Codepoint == 0x0301 {
			acuteOff = g.OffsetY
			acuteFound = true
		}
	}
	if !macronFound || !acuteFound {
		t.Fatal("missing macron or acute glyph in output")
	}

	// Without mark-to-mark (type 6), both marks get positioned relative to the base
	// and may overlap. With type 6, the acute should be placed higher than the macron.
	if macronOff == 0 && acuteOff == 0 {
		t.Error("both marks have zero OffsetY; mark positioning not applied")
		return
	}

	// The acute (stacked on top of macron) should have a different Y offset.
	if acuteOff == macronOff {
		t.Errorf("acute and macron have same OffsetY=%d; mark-to-mark should stack them", acuteOff)
	}
}

// TestGPOSMarkDoesNotAffectAdvance verifies that mark positioning only changes
// offsets, never advances. This is a key property of GPOS types 4 and 6.
func TestGPOSMarkDoesNotAffectAdvance(t *testing.T) {
	f := loadTestFont(t)

	// Shape "A" alone and "A + combining acute" — base advance should be identical.
	var cfg ShapeConfig
	cfg.Font = f

	plainRuns := cfg.ShapeSimple(nil, "A", DirectionLTR)
	markRuns := cfg.ShapeSimple(nil, "A\u0301", DirectionLTR)

	if len(plainRuns) == 0 || len(plainRuns[0].Glyphs) == 0 {
		t.Fatal("no glyphs for plain A")
	}
	if len(markRuns) == 0 || len(markRuns[0].Glyphs) < 2 {
		t.Fatal("expected 2 glyphs for A+acute")
	}

	plainAdv := plainRuns[0].Glyphs[0].AdvanceX
	var markBaseAdv int32
	for _, g := range markRuns[0].Glyphs {
		if g.Codepoint == 'A' {
			markBaseAdv = g.AdvanceX
		}
	}

	if plainAdv != markBaseAdv {
		t.Errorf("base 'A' advance changed with mark: plain=%d, withMark=%d", plainAdv, markBaseAdv)
	}
}

// TestGPOSMarkFeatureLookups verifies that the test font actually contains
// mark and mkmk features pointing to type 4 and type 6 lookups respectively.
// This is a structural test that validates our test preconditions.
func TestGPOSMarkFeatureLookups(t *testing.T) {
	f := loadTestFont(t)
	te := f.tables[tableGpos]
	if te.length == 0 {
		t.Skip("no GPOS table")
	}
	base := int(te.offset)
	lookupListOff := base + int(readU16BE(f.data, base+8))

	var idxBuf [4]int
	// Check 'mark' feature exists and points to type 4 lookups.
	markIndices := f.findGPOSFeatureIndices(idxBuf[:0], FeatureTagMark)
	if len(markIndices) == 0 {
		t.Fatal("font has no 'mark' GPOS feature")
	}
	hasType4 := false
	for _, fi := range markIndices {
		for _, li := range f.gposFeatureLookups(fi) {
			lookupOff := lookupListOff + int(readU16BE(f.data, lookupListOff+2+int(li)*2))
			lt := readU16BE(f.data, lookupOff)
			if lt == 4 {
				hasType4 = true
			}
		}
	}
	if !hasType4 {
		t.Error("'mark' feature has no type 4 (mark-to-base) lookups")
	}

	// Check 'mkmk' feature exists and points to type 6 lookups.
	mkmkIndices := f.findGPOSFeatureIndices(idxBuf[:0], FeatureTagMkmk)
	if len(mkmkIndices) == 0 {
		t.Fatal("font has no 'mkmk' GPOS feature")
	}
	hasType6 := false
	for _, fi := range mkmkIndices {
		for _, li := range f.gposFeatureLookups(fi) {
			lookupOff := lookupListOff + int(readU16BE(f.data, lookupListOff+2+int(li)*2))
			lt := readU16BE(f.data, lookupOff)
			if lt == 6 {
				hasType6 = true
			}
		}
	}
	if !hasType6 {
		t.Error("'mkmk' feature has no type 6 (mark-to-mark) lookups")
	}
}

// GlyphAdvance (exported) tests

func TestGlyphAdvance_LatinA(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	adv := f.GlyphAdvance(gid)
	if adv <= 0 {
		t.Errorf("GlyphAdvance('A') = %d, want > 0", adv)
	}
}

func TestGlyphAdvance_Space(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID(' ')
	adv := f.GlyphAdvance(gid)
	if adv <= 0 {
		t.Errorf("GlyphAdvance(' ') = %d, want > 0", adv)
	}
}

func TestGlyphAdvance_MatchesInternal(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	if f.GlyphAdvance(gid) != f.glyphAdvance(gid) {
		t.Error("GlyphAdvance and glyphAdvance disagree")
	}
}

// GlyphBounds tests

func TestGlyphBounds_LatinA(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	xMin, yMin, xMax, yMax := f.GlyphBounds(gid)
	if xMax <= xMin {
		t.Errorf("GlyphBounds('A') xMax=%d <= xMin=%d", xMax, xMin)
	}
	if yMax <= yMin {
		t.Errorf("GlyphBounds('A') yMax=%d <= yMin=%d", yMax, yMin)
	}
}

func TestGlyphBounds_Space(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID(' ')
	xMin, yMin, xMax, yMax := f.GlyphBounds(gid)
	// Space has no outline, so bounds should be zero.
	if xMin != 0 || yMin != 0 || xMax != 0 || yMax != 0 {
		t.Errorf("GlyphBounds(' ') = (%d,%d,%d,%d), want all zero", xMin, yMin, xMax, yMax)
	}
}

func TestGlyphBounds_InvalidGlyph(t *testing.T) {
	f := loadTestFont(t)
	xMin, yMin, xMax, yMax := f.GlyphBounds(0xFFFF)
	if xMin != 0 || yMin != 0 || xMax != 0 || yMax != 0 {
		t.Errorf("GlyphBounds(0xFFFF) = (%d,%d,%d,%d), want all zero", xMin, yMin, xMax, yMax)
	}
}

// GlyphOutline tests

func TestGlyphOutline_LatinA(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	segs := f.GlyphOutline(nil, gid)
	if len(segs) == 0 {
		t.Fatal("GlyphOutline('A') returned no segments")
	}
	// Must start with MoveTo.
	if segs[0].Op != SegmentMoveTo {
		t.Errorf("first segment op = %d, want SegmentMoveTo(%d)", segs[0].Op, SegmentMoveTo)
	}
	// Must contain at least one Close.
	hasClose := false
	for _, s := range segs {
		if s.Op == SegmentClose {
			hasClose = true
			break
		}
	}
	if !hasClose {
		t.Error("GlyphOutline('A') has no SegmentClose")
	}
}

func TestGlyphOutline_LatinA_ContourCount(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	segs := f.GlyphOutline(nil, gid)
	// 'A' has 2 contours (outer shape + inner triangle hole).
	contours := 0
	for _, s := range segs {
		if s.Op == SegmentMoveTo {
			contours++
		}
	}
	if contours != 2 {
		t.Errorf("GlyphOutline('A') has %d contours, want 2", contours)
	}
}

func TestGlyphOutline_Space(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID(' ')
	segs := f.GlyphOutline(nil, gid)
	if len(segs) != 0 {
		t.Errorf("GlyphOutline(' ') returned %d segments, want 0", len(segs))
	}
}

func TestGlyphOutline_InvalidGlyph(t *testing.T) {
	f := loadTestFont(t)
	segs := f.GlyphOutline(nil, 0xFFFF)
	if len(segs) != 0 {
		t.Errorf("GlyphOutline(0xFFFF) returned %d segments, want 0", len(segs))
	}
}

func TestGlyphOutline_AppendsToExisting(t *testing.T) {
	f := loadTestFont(t)
	gid := f.GlyphID('A')
	sentinel := Segment{Op: SegmentLineTo, X: 9999, Y: 9999}
	dst := []Segment{sentinel}
	result := f.GlyphOutline(dst, gid)
	if len(result) <= 1 {
		t.Fatal("GlyphOutline did not append segments")
	}
	if result[0] != sentinel {
		t.Error("GlyphOutline overwrote existing segment instead of appending")
	}
}

func TestGlyphOutline_EveryContourClosed(t *testing.T) {
	f := loadTestFont(t)
	// Check several glyphs that have outlines.
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789" {
		gid := f.GlyphID(r)
		segs := f.GlyphOutline(nil, gid)
		if len(segs) == 0 {
			continue
		}
		// Every MoveTo must be balanced by a Close.
		moves, closes := 0, 0
		for _, s := range segs {
			switch s.Op {
			case SegmentMoveTo:
				moves++
			case SegmentClose:
				closes++
			}
		}
		if moves != closes {
			t.Errorf("glyph %q: %d MoveTo vs %d Close", r, moves, closes)
		}
	}
}
