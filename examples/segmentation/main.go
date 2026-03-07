// Example segmentation demonstrates the kbts text segmentation APIs.
package main

import (
	"fmt"

	kb "github.com/soypat/lefevre"
)

func main() {
	// GuessTextProperties detects direction and script from text.
	fmt.Println("=== GuessTextProperties ===")
	samples := []string{
		"Hello, world!",
		"مرحبا بالعالم",
		"שלום עולם",
		"Hello مرحبا",
	}
	for _, s := range samples {
		dir, script := kb.GuessTextProperties(s)
		fmt.Printf("  %-20q → dir=%-3s script=%s\n", s, dir, script)
	}

	// Breaker segments text into all break types at once.
	fmt.Println("\n=== Breaker ===")
	text := "Hello world!\nمرحبا"
	var b kb.Breaker
	b.Direction = kb.DirectionLTR
	breaks, _ := b.AppendBreak(nil, []byte(text))
	breaks = b.End(breaks)
	fmt.Printf("  Text: %q\n", text)
	fmt.Printf("  %d break points:\n", len(breaks))
	for _, br := range breaks {
		var types []string
		if br.Flags.HasAll(kb.BreakFlagGrapheme) {
			types = append(types, "grapheme")
		}
		if br.Flags.HasAll(kb.BreakFlagWord) {
			types = append(types, "word")
		}
		if br.Flags.HasAll(kb.BreakFlagLineSoft) {
			types = append(types, "line-soft")
		}
		if br.Flags.IsHardBreak() {
			types = append(types, "line-hard")
		}
		if br.Flags.HasAll(kb.BreakFlagScript) {
			types = append(types, fmt.Sprintf("script→%s", br.Script))
		}
		if br.Flags.HasAll(kb.BreakFlagDirection) {
			types = append(types, fmt.Sprintf("dir→%s", br.Direction))
		}
		if len(types) > 0 {
			fmt.Printf("    pos=%2d  %v\n", br.Position, types)
		}
	}

	// Streaming API with BreakState — process codepoints one at a time.
	fmt.Println("\n=== BreakState (streaming) ===")
	stream := "café"
	fmt.Printf("  Text: %q\n", stream)
	var bs kb.BreakState
	bs.Begin(kb.DirectionLTR, kb.JapaneseLineBreakStrict, kb.BreakConfigNone)
	runes := []rune(stream)
	for i, r := range runes {
		eot := i == len(runes)-1
		bs.AddCodepoint(r, 1, eot)
		for {
			brk, ok := bs.Next()
			if !ok {
				break
			}
			if brk.Flags.HasAll(kb.BreakFlagGrapheme) {
				fmt.Printf("    grapheme cluster start at pos %d\n", brk.Position)
			}
		}
	}
}
