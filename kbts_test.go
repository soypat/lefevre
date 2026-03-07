package lefevre

import "testing"

func TestDirectionReverse(t *testing.T) {
	tests := []struct {
		in, want Direction
	}{
		{DirectionUnknown, DirectionUnknown},
		{DirectionLTR, DirectionRTL},
		{DirectionRTL, DirectionLTR},
	}
	for _, tt := range tests {
		got := tt.in.Reverse()
		if got != tt.want {
			t.Errorf("Direction(%d).Reverse() = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestDirectionString(t *testing.T) {
	tests := []struct {
		d    Direction
		want string
	}{
		{DirectionUnknown, "Unknown"},
		{DirectionLTR, "LTR"},
		{DirectionRTL, "RTL"},
		{Direction(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("Direction(%d).String() = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestBreakFlags(t *testing.T) {
	t.Run("Has", func(t *testing.T) {
		f := BreakFlagDirection | BreakFlagWord
		if !f.HasAll(BreakFlagDirection) {
			t.Error("expected Has(Direction) = true")
		}
		if !f.HasAll(BreakFlagWord) {
			t.Error("expected Has(Word) = true")
		}
		if f.HasAll(BreakFlagScript) {
			t.Error("expected Has(Script) = false")
		}
		// Has with combined flags
		if !f.HasAll(BreakFlagDirection | BreakFlagWord) {
			t.Error("expected Has(Direction|Word) = true")
		}
		if f.HasAll(BreakFlagDirection | BreakFlagScript) {
			t.Error("expected Has(Direction|Script) = false")
		}
	})

	t.Run("IsLineBreak", func(t *testing.T) {
		if BreakFlagLineSoft.IsLineBreak() != true {
			t.Error("LineSoft should be a line break")
		}
		if BreakFlagLineHard.IsLineBreak() != true {
			t.Error("LineHard should be a line break")
		}
		if (BreakFlagLineSoft | BreakFlagLineHard).IsLineBreak() != true {
			t.Error("LineSoft|LineHard should be a line break")
		}
		if BreakFlagWord.IsLineBreak() != false {
			t.Error("Word should not be a line break")
		}
	})

	t.Run("IsHardBreak", func(t *testing.T) {
		if BreakFlagLineHard.IsHardBreak() != true {
			t.Error("LineHard should be a hard break")
		}
		if BreakFlagLineSoft.IsHardBreak() != false {
			t.Error("LineSoft should not be a hard break")
		}
	})

	t.Run("constants", func(t *testing.T) {
		if BreakFlagLine != BreakFlagLineSoft|BreakFlagLineHard {
			t.Error("BreakFlagLine should be LineSoft|LineHard")
		}
		if BreakFlagAny&BreakFlagManual != 0 {
			t.Error("BreakFlagAny should not include Manual")
		}
	})
}

func TestGlyphFlagsHas(t *testing.T) {
	f := GlyphFlagIsol | GlyphFlagLigature
	if !f.Has(GlyphFlagIsol) {
		t.Error("expected Has(Isol)")
	}
	if !f.Has(GlyphFlagLigature) {
		t.Error("expected Has(Ligature)")
	}
	if f.Has(GlyphFlagFina) {
		t.Error("unexpected Has(Fina)")
	}
}

func TestShaperIsComplex(t *testing.T) {
	if ShaperDefault.IsComplex() {
		t.Error("Default should not be complex")
	}
	complexShapers := []Shaper{ShaperArabic, ShaperHangul, ShaperHebrew, ShaperIndic, ShaperKhmer, ShaperMyanmar, ShaperTibetan, ShaperUSE}
	for _, s := range complexShapers {
		if !s.IsComplex() {
			t.Errorf("Shaper %v should be complex", s)
		}
	}
}

func TestShapeErrorImplementsError(t *testing.T) {
	var err error = ShapeErrorInvalidFont
	if err.Error() != "invalid font" {
		t.Errorf("got %q", err.Error())
	}
	if ShapeErrorNone.Error() != "no error" {
		t.Error("None should return 'no error'")
	}
}

func TestScriptDirection(t *testing.T) {
	// Arabic and Hebrew are RTL.
	if ScriptArabic.Direction() != DirectionRTL {
		t.Error("Arabic should be RTL")
	}
	if ScriptHebrew.Direction() != DirectionRTL {
		t.Error("Hebrew should be RTL")
	}
	// Latin, Greek, Cyrillic etc. are LTR.
	ltrScripts := []Script{ScriptLatin, ScriptGreek, ScriptCyrillic, ScriptDevanagari, ScriptUnknown}
	for _, s := range ltrScripts {
		if s.Direction() != DirectionLTR {
			t.Errorf("Script %d should be LTR", s)
		}
	}
}

func TestScriptIsComplex(t *testing.T) {
	// Simple scripts (ShaperDefault).
	simpleScripts := []Script{ScriptLatin, ScriptGreek, ScriptCyrillic, ScriptArmenian, ScriptGeorgian}
	for _, s := range simpleScripts {
		if s.IsComplex() {
			t.Errorf("Script %d should not be complex", s)
		}
	}
	// Complex scripts.
	complexScripts := []Script{ScriptArabic, ScriptDevanagari, ScriptBengali, ScriptHangul, ScriptHebrew, ScriptKhmer, ScriptMyanmar, ScriptTibetan}
	for _, s := range complexScripts {
		if !s.IsComplex() {
			t.Errorf("Script %d should be complex", s)
		}
	}
}

func TestScriptTag(t *testing.T) {
	tests := []struct {
		script Script
		tag    ScriptTag
	}{
		{ScriptLatin, ScriptTagLatin},
		{ScriptArabic, ScriptTagArabic},
		{ScriptHebrew, ScriptTagHebrew},
		{ScriptDevanagari, ScriptTagDevanagari},
		{ScriptHangul, ScriptTagHangul},
		{ScriptUnknown, ScriptTagUnknown},
	}
	for _, tt := range tests {
		got := tt.script.Tag()
		if got != tt.tag {
			t.Errorf("Script(%d).Tag() = 0x%x, want 0x%x", tt.script, got, tt.tag)
		}
	}
}

func TestScriptShaper(t *testing.T) {
	tests := []struct {
		script Script
		shaper Shaper
	}{
		{ScriptLatin, ShaperDefault},
		{ScriptArabic, ShaperArabic},
		{ScriptHebrew, ShaperHebrew},
		{ScriptDevanagari, ShaperIndic},
		{ScriptBengali, ShaperIndic},
		{ScriptHangul, ShaperHangul},
		{ScriptKhmer, ShaperKhmer},
		{ScriptMyanmar, ShaperMyanmar},
		{ScriptTibetan, ShaperTibetan},
		{ScriptAdlam, ShaperUSE},
	}
	for _, tt := range tests {
		got := tt.script.Shaper()
		if got != tt.shaper {
			t.Errorf("Script(%d).Shaper() = %v, want %v", tt.script, got, tt.shaper)
		}
	}
}

func TestScriptPropertiesTableLength(t *testing.T) {
	// Verify the table has the right number of entries.
	if len(scriptProps) != int(scriptCount) {
		t.Errorf("scriptProps has %d entries, want %d", len(scriptProps), scriptCount)
	}
}

func TestScriptOutOfBounds(t *testing.T) {
	// Out-of-bounds script should not panic.
	s := Script(255)
	_ = s.Direction()
	_ = s.IsComplex()
	_ = s.Tag()
	_ = s.Shaper()
}

func TestFourcc(t *testing.T) {
	// Test that fourcc matches expected values.
	// 'latn' = 'l' | 'a'<<8 | 't'<<16 | 'n'<<24
	got := fourcc('l', 'a', 't', 'n')
	want := uint32('l') | uint32('a')<<8 | uint32('t')<<16 | uint32('n')<<24
	if got != want {
		t.Errorf("fourcc('l','a','t','n') = 0x%x, want 0x%x", got, want)
	}
}

func TestFeatureTagValues(t *testing.T) {
	// Spot-check a few feature tags.
	if FeatureTagKern != FeatureTag(fourcc('k', 'e', 'r', 'n')) {
		t.Error("FeatureTagKern mismatch")
	}
	if FeatureTagLiga != FeatureTag(fourcc('l', 'i', 'g', 'a')) {
		t.Error("FeatureTagLiga mismatch")
	}
	if FeatureTagCalt != FeatureTag(fourcc('c', 'a', 'l', 't')) {
		t.Error("FeatureTagCalt mismatch")
	}
}
