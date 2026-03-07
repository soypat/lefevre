package lefevre

import (
	"testing"
)

func appendBreaks(t *testing.T, text string, dir Direction, flags BreakConfigFlags) []Break {
	t.Helper()
	var b Breaker
	b.Direction = dir
	b.Flags = flags
	breaks, _ := b.AppendBreak(nil, []byte(text))
	breaks = b.End(breaks)
	return breaks
}

func TestGuessTextProperties(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantDir Direction
		wantScr Script
	}{
		{"Latin", "Hello world", DirectionLTR, ScriptLatin},
		{"Arabic", "مرحبا", DirectionRTL, ScriptArabic},
		{"Hebrew", "שלום", DirectionRTL, ScriptHebrew},
		{"LatinThenArabic", "Hello مرحبا", DirectionLTR, ScriptLatin},
		{"SpaceThenArabic", " مرحبا", DirectionRTL, ScriptArabic},
		{"Empty", "", DirectionLTR, ScriptUnknown},
		{"Digits", "123", DirectionLTR, ScriptUnknown}, // digits have no strong direction or script
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotScr := GuessTextProperties(tt.text)
			if gotDir != tt.wantDir {
				t.Errorf("direction = %v, want %v", gotDir, tt.wantDir)
			}
			if gotScr != tt.wantScr {
				t.Errorf("script = %v, want %v", gotScr, tt.wantScr)
			}
		})
	}
}

func TestBreakStringLineHard(t *testing.T) {
	breaks := appendBreaks(t, "hello\nworld", DirectionLTR, BreakConfigNone)
	found := false
	for _, b := range breaks {
		if b.Flags.IsHardBreak() {
			found = true
			if b.Position != 6 {
				t.Errorf("hard break at position %d, want 6", b.Position)
			}
		}
	}
	if !found {
		t.Error("no hard line break found in \"hello\\nworld\"")
	}
}

func TestBreakStringLineSoft(t *testing.T) {
	breaks := appendBreaks(t, "hello world", DirectionLTR, BreakConfigNone)
	found := false
	for _, b := range breaks {
		if b.Flags.HasAll(BreakFlagLineSoft) && !b.Flags.IsHardBreak() {
			found = true
			break
		}
	}
	if !found {
		t.Error("no soft line break found in \"hello world\"")
	}
}

func TestBreakStringWord(t *testing.T) {
	breaks := appendBreaks(t, "hello world", DirectionLTR, BreakConfigNone)
	wordBreaks := 0
	for _, b := range breaks {
		if b.Flags.HasAll(BreakFlagWord) {
			wordBreaks++
		}
	}
	if wordBreaks < 1 {
		t.Errorf("expected at least 1 word break, got %d", wordBreaks)
	}
}

func TestBreakStringGrapheme(t *testing.T) {
	// "e" + combining acute accent = one grapheme cluster.
	breaks := appendBreaks(t, "e\u0301x", DirectionLTR, BreakConfigNone)
	graphemes := 0
	for _, b := range breaks {
		if b.Flags.HasAll(BreakFlagGrapheme) {
			graphemes++
		}
	}
	if graphemes != 3 {
		t.Errorf("expected 3 grapheme breaks (including start-of-text), got %d", graphemes)
	}
	for _, b := range breaks {
		if b.Flags.HasAll(BreakFlagGrapheme) && b.Position == 1 {
			t.Error("unexpected grapheme break at position 1 (between base 'e' and combining accent)")
		}
	}
}

func TestBreakStringScript(t *testing.T) {
	breaks := appendBreaks(t, "Helloمرحبا", DirectionLTR, BreakConfigNone)
	found := false
	for _, b := range breaks {
		if b.Flags.HasAll(BreakFlagScript) && b.Position > 0 {
			found = true
			if b.Script == ScriptArabic {
				if b.Position != 5 {
					t.Errorf("script break to Arabic at position %d, want 5", b.Position)
				}
			}
		}
	}
	if !found {
		t.Error("no script break found between Latin and Arabic")
	}
}

func TestBreakStringDirection(t *testing.T) {
	breaks := appendBreaks(t, "Hello مرحبا", DirectionLTR, BreakConfigNone)
	found := false
	for _, b := range breaks {
		if b.Flags.HasAll(BreakFlagDirection) && b.Direction == DirectionRTL {
			found = true
		}
	}
	if !found {
		t.Error("no RTL direction break found in \"Hello مرحبا\"")
	}
}

func TestBreakStateStreaming(t *testing.T) {
	text := "hello\nworld"
	// Batch result.
	batchBreaks := appendBreaks(t, text, DirectionLTR, BreakConfigNone)

	// Streaming result.
	var bs BreakState
	bs.Begin(DirectionLTR, JapaneseLineBreakStrict, BreakConfigNone)
	var streamBreaks []Break
	for _, r := range text {
		bs.AddCodepoint(r, 1, false)
		for {
			b, ok := bs.Next()
			if !ok {
				break
			}
			streamBreaks = append(streamBreaks, b)
		}
	}
	bs.End()
	for {
		b, ok := bs.Next()
		if !ok {
			break
		}
		streamBreaks = append(streamBreaks, b)
	}

	if len(streamBreaks) != len(batchBreaks) {
		t.Errorf("streaming produced %d breaks, batch produced %d", len(streamBreaks), len(batchBreaks))
	}
}

func TestBreakStringEndOfTextHardBreak(t *testing.T) {
	breaks := appendBreaks(t, "hello", DirectionLTR, BreakConfigEndOfTextGeneratesHardBreak)
	found := false
	for _, b := range breaks {
		if b.Flags.IsHardBreak() {
			found = true
		}
	}
	if !found {
		t.Error("BreakConfigEndOfTextGeneratesHardBreak did not produce a hard break at end of text")
	}
}

func TestBreakerChunked(t *testing.T) {
	text := "hello\nworld"
	// Single-call result.
	singleBreaks := appendBreaks(t, text, DirectionLTR, BreakConfigNone)

	// Chunked result — feed 2 bytes at a time.
	var b Breaker
	b.Direction = DirectionLTR
	var chunkedBreaks []Break
	remaining := text
	for len(remaining) > 0 {
		end := 2
		if end > len(remaining) {
			end = len(remaining)
		}
		chunk := remaining[:end]
		var n int
		chunkedBreaks, n = b.AppendBreak(chunkedBreaks, []byte(chunk))
		remaining = remaining[n:]
	}
	chunkedBreaks = b.End(chunkedBreaks)

	if len(chunkedBreaks) != len(singleBreaks) {
		t.Errorf("chunked produced %d breaks, single produced %d", len(chunkedBreaks), len(singleBreaks))
	}
}
