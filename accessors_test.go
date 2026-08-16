package lefevre

import (
	"sync"
	"testing"
)

// The accessors are the low-level face a consumer sees without knowing this
// package's structs, so every one of them is checked against the FontInfo it is
// backed by: a mis-wired accessor is the failure this catches.
func TestFontAccessors(t *testing.T) {
	for _, tt := range []struct {
		name string
		load func(*testing.T) *Font
		// Open Sans carries Windows name records only, DejaVu Sans both.
		postScriptName    string
		family, subfamily string
		unitsPerEm        int
	}{
		{"DejaVuSans", loadTestFont, "DejaVuSans", "DejaVu Sans", "Book", 2048},
		{"OpenSans", loadOpenSans, "OpenSans-Regular", "Open Sans", "Regular", 2048},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.load(t)
			var info FontInfo
			f.ReadInfo(&info)

			if got := f.UnitsPerEm(); got != tt.unitsPerEm {
				t.Errorf("UnitsPerEm() = %d, want %d", got, tt.unitsPerEm)
			}
			if got := f.PostScriptName(); got != tt.postScriptName {
				t.Errorf("PostScriptName() = %q, want %q", got, tt.postScriptName)
			}
			if gotFam, gotSub := f.Family(); gotFam != tt.family || gotSub != tt.subfamily {
				t.Errorf("Family() = %q, %q; want %q, %q", gotFam, gotSub, tt.family, tt.subfamily)
			}

			// The same values reached the long way, so a rewired accessor and a
			// rewired FontInfo cannot agree with each other and be wrong.
			if got := f.UnitsPerEm(); got != int(info.UnitsPerEm) {
				t.Errorf("UnitsPerEm() = %d, want Info().UnitsPerEm = %d", got, info.UnitsPerEm)
			}
			if got := f.PostScriptName(); got != info.PostScriptName {
				t.Errorf("PostScriptName() = %q, want Info().PostScriptName = %q", got, info.PostScriptName)
			}
			if gotFam, gotSub := f.Family(); gotFam != info.TypographicFamily || gotSub != info.TypographicSubfamily {
				t.Errorf("Family() = %q, %q; want Info() typographic pair %q, %q",
					gotFam, gotSub, info.TypographicFamily, info.TypographicSubfamily)
			}

			xMin, yMin, xMax, yMax := f.Bounds()
			if xMin != info.XMin || yMin != info.YMin || xMax != info.XMax || yMax != info.YMax {
				t.Errorf("Bounds() = %d, %d, %d, %d; want %d, %d, %d, %d",
					xMin, yMin, xMax, yMax, info.XMin, info.YMin, info.XMax, info.YMax)
			}
			if xMin >= xMax || yMin >= yMax {
				t.Errorf("Bounds() = %d, %d, %d, %d is not a box", xMin, yMin, xMax, yMax)
			}

			ascent, descent, lineGap, capHeight := f.VMetrics()
			if ascent != info.Ascent || descent != info.Descent ||
				lineGap != info.LineGap || capHeight != info.CapHeight {
				t.Errorf("VMetrics() = %d, %d, %d, %d; want %d, %d, %d, %d",
					ascent, descent, lineGap, capHeight,
					info.Ascent, info.Descent, info.LineGap, info.CapHeight)
			}
			if ascent <= 0 || descent >= 0 {
				t.Errorf("VMetrics() ascent=%d descent=%d, want ascent above and descent below the baseline", ascent, descent)
			}

			weight, italicAngle, fixedPitch := f.Style()
			if weight != int(info.WeightClass) || italicAngle != info.ItalicAngle || fixedPitch != info.IsFixedPitch {
				t.Errorf("Style() = %d, %v, %v; want %d, %v, %v", weight, italicAngle, fixedPitch,
					info.WeightClass, info.ItalicAngle, info.IsFixedPitch)
			}
			if weight < 100 || weight > 1000 {
				t.Errorf("Style() weight = %d, want a usWeightClass in 100..1000", weight)
			}
		})
	}
}

// Info parses at load, so asking for it costs nothing. A caller that wants two
// fields used to pay for the name table twice.
func TestInfoDoesNotAllocate(t *testing.T) {
	for _, tt := range []struct {
		name string
		load func(*testing.T) *Font
	}{
		{"DejaVuSans", loadTestFont},
		{"OpenSans", loadOpenSans},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.load(t)
			var info FontInfo
			allocs := testing.AllocsPerRun(100, func() {
				f.ReadInfo(&info)
			})
			if allocs != 0 {
				t.Errorf("ReadInfo allocates %v times per call, want 0", allocs)
			}
		})
	}
}

// A *Font is read-only once parsed, so any number of goroutines may read one.
// Meaningful under -race; it is what keeps the FontInfo cache eager.
func TestFontConcurrentReads(t *testing.T) {
	f := loadTestFont(t)
	var want FontInfo
	f.ReadInfo(&want)
	wantGID := f.GlyphID('A')
	wantAdvance := f.GlyphAdvance(wantGID)

	const goroutines = 8
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var got FontInfo
			for range 100 {
				if f.ReadInfo(&got); got != want {
					t.Errorf("ReadInfo = %+v, want %+v", got, want)
					return
				}
				if got := f.GlyphID('A'); got != wantGID {
					t.Errorf("GlyphID('A') = %d, want %d", got, wantGID)
					return
				}
				if got := f.GlyphAdvance(wantGID); got != wantAdvance {
					t.Errorf("GlyphAdvance(%d) = %d, want %d", wantGID, got, wantAdvance)
					return
				}
				if got := f.UnitsPerEm(); got != int(want.UnitsPerEm) {
					t.Errorf("UnitsPerEm() = %d, want %d", got, want.UnitsPerEm)
					return
				}
				if got := f.PostScriptName(); got != want.PostScriptName {
					t.Errorf("PostScriptName() = %q, want %q", got, want.PostScriptName)
					return
				}
				weight, _, _ := f.Style()
				if weight != int(want.WeightClass) {
					t.Errorf("Style() weight = %d, want %d", weight, want.WeightClass)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// The raw OS/2 classes are what a PDF descriptor and a family matcher need: the
// buckets cannot tell 600 from 700, and those pick different faces.
func TestOS2RawClasses(t *testing.T) {
	for _, tt := range []struct {
		name        string
		load        func(*testing.T) *Font
		weightClass uint16
		widthClass  uint16
		xHeight     int16 // 0 where the font's OS/2 is too short to carry one.
	}{
		{"DejaVuSans", loadTestFont, 400, 5, 0}, // OS/2 version 1, 86 bytes: no sxHeight.
		{"OpenSans", loadOpenSans, 400, 5, 1096},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var info FontInfo
			tt.load(t).ReadInfo(&info)
			if info.WeightClass != tt.weightClass {
				t.Errorf("WeightClass = %d, want %d", info.WeightClass, tt.weightClass)
			}
			if info.WidthClass != tt.widthClass {
				t.Errorf("WidthClass = %d, want %d", info.WidthClass, tt.widthClass)
			}
			if info.XHeight != tt.xHeight {
				t.Errorf("XHeight = %d, want %d", info.XHeight, tt.xHeight)
			}
			// The buckets stay, and stay consistent with the numbers they came from.
			if want := weightClassToFontWeight(info.WeightClass); info.Weight != want {
				t.Errorf("Weight = %v, want %v for usWeightClass %d", info.Weight, want, info.WeightClass)
			}
			if want := widthClassToFontWidth(info.WidthClass); info.Width != want {
				t.Errorf("Width = %v, want %v for usWidthClass %d", info.Width, want, info.WidthClass)
			}
		})
	}
}

// Every accessor answers zero when a font is broken, which is the same answer
// as "this font has no such glyph". Err tells the two apart.
func TestErrReportsParseFailure(t *testing.T) {
	f := loadTestFont(t)
	if err := f.Err(); err != nil {
		t.Errorf("Err() = %v on a good font, want nil", err)
	}

	// Half a font: the directory still parses, the tables it points at do not.
	data := loadTestFontData(t)
	cut, err := FontFromMemory(data[:len(data)/2], 0)
	if err != nil {
		t.Skipf("truncated font rejected at parse: %v", err)
	}
	if cut.Err() == nil {
		t.Error("Err() = nil on a truncated font, want an error")
	}
	if cut.IsValid() {
		t.Error("IsValid() = true on a truncated font")
	}
}
