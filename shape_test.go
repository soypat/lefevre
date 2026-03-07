package lefevre

import "testing"

func TestShapeBasicLatin(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, "Hello", DirectionLTR)
	if len(runs) == 0 {
		t.Fatal("Shape(\"Hello\") produced no runs")
	}
	totalGlyphs := 0
	for _, r := range runs {
		totalGlyphs += len(r.Glyphs)
	}
	if totalGlyphs != 5 {
		t.Errorf("expected 5 glyphs for \"Hello\", got %d", totalGlyphs)
	}
}

func TestShapeGlyphIDsNonZero(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, "ABC", DirectionLTR)
	if len(runs) == 0 {
		t.Fatal("Shape(\"ABC\") produced no runs")
	}
	for ri, r := range runs {
		for gi, g := range r.Glyphs {
			if g.ID == 0 {
				t.Errorf("run[%d].Glyphs[%d].ID = 0, want non-zero", ri, gi)
			}
		}
	}
}

func TestShapeAdvancesPositive(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, "Hello", DirectionLTR)
	if len(runs) == 0 {
		t.Fatal("no runs produced")
	}
	for ri, r := range runs {
		for gi, g := range r.Glyphs {
			if g.AdvanceX <= 0 {
				t.Errorf("run[%d].Glyphs[%d].AdvanceX = %d, want > 0", ri, gi, g.AdvanceX)
			}
		}
	}
}

func TestShapeDirectionPreserved(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f

	ltrRuns := cfg.ShapeSimple(nil, "Hello", DirectionLTR)
	if len(ltrRuns) == 0 {
		t.Fatal("no LTR runs")
	}
	if ltrRuns[0].Direction != DirectionLTR {
		t.Errorf("LTR run direction = %v, want LTR", ltrRuns[0].Direction)
	}

	rtlRuns := cfg.ShapeSimple(nil, "\u0645\u0631\u062D\u0628\u0627", DirectionRTL)
	if len(rtlRuns) == 0 {
		t.Fatal("no RTL runs")
	}
	if rtlRuns[0].Direction != DirectionRTL {
		t.Errorf("RTL run direction = %v, want RTL", rtlRuns[0].Direction)
	}
}

func TestShapeFeatureOverrideLiga(t *testing.T) {
	f := loadTestFont(t)

	var cfgDefault ShapeConfig
	cfgDefault.Font = f
	runsDefault := cfgDefault.ShapeSimple(nil, "ffi", DirectionLTR)

	var cfgNoLiga ShapeConfig
	cfgNoLiga.Font = f
	cfgNoLiga.Features = []FeatureOverride{
		{Tag: FeatureTagLiga, Value: 0},
	}
	runsNoLiga := cfgNoLiga.ShapeSimple(nil, "ffi", DirectionLTR)

	if len(runsDefault) == 0 || len(runsNoLiga) == 0 {
		t.Fatal("expected runs from both shape calls")
	}

	defaultGlyphs := 0
	for _, r := range runsDefault {
		defaultGlyphs += len(r.Glyphs)
	}
	noLigaGlyphs := 0
	for _, r := range runsNoLiga {
		noLigaGlyphs += len(r.Glyphs)
	}

	if noLigaGlyphs != 3 {
		t.Errorf("with liga=0, expected 3 glyphs for \"ffi\", got %d", noLigaGlyphs)
	}
	if defaultGlyphs >= noLigaGlyphs {
		t.Errorf("with default features, expected fewer glyphs than %d (ligature), got %d", noLigaGlyphs, defaultGlyphs)
	}
}

func TestShapeNilFont(t *testing.T) {
	var cfg ShapeConfig
	runs := cfg.ShapeSimple(nil, "Hello", DirectionLTR)
	if len(runs) != 0 {
		t.Errorf("expected 0 runs for nil font, got %d", len(runs))
	}
}

func TestShapeEmptyText(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, "", DirectionLTR)
	if len(runs) != 0 {
		t.Errorf("expected 0 runs for empty text, got %d", len(runs))
	}
}

func TestShapeRunFontPointer(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, "A", DirectionLTR)
	if len(runs) == 0 {
		t.Fatal("no runs")
	}
	if runs[0].Font != f {
		t.Error("run.Font does not match ShapeConfig.Font")
	}
}

func TestShapeAppendPattern(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	existing := []Run{{Direction: DirectionRTL}}
	runs := cfg.ShapeSimple(existing, "A", DirectionLTR)
	if len(runs) < 2 {
		t.Fatalf("expected at least 2 runs (1 existing + 1 new), got %d", len(runs))
	}
	if runs[0].Direction != DirectionRTL {
		t.Error("pre-existing run was overwritten")
	}
}

func TestShapeRunScript(t *testing.T) {
	f := loadTestFont(t)
	var cfg ShapeConfig
	cfg.Font = f
	runs := cfg.ShapeSimple(nil, "Hello", DirectionLTR)
	if len(runs) == 0 {
		t.Fatal("no runs")
	}
	if runs[0].Script != ScriptLatin {
		t.Errorf("run.Script = %v, want ScriptLatin", runs[0].Script)
	}
}
