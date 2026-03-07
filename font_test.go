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
