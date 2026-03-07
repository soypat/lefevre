// Example font demonstrates the kbts font parsing APIs.
package main

import (
	"fmt"
	"os"

	kb "github.com/soypat/lefevre"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: font <path.ttf>")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// FontCount reports how many fonts are in the file (>1 for TTC collections).
	fmt.Printf("Fonts in file: %d\n", kb.FontCount(data))

	// Parse the first font.
	f, err := kb.FontFromMemory(data, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", err)
		os.Exit(1)
	}
	fmt.Printf("Valid: %v\n\n", f.IsValid())

	// Font metadata from name, head, OS/2, and hhea tables.
	info := f.Info()
	fmt.Println("=== Font Info ===")
	fmt.Printf("  Family:      %s\n", info.Family)
	fmt.Printf("  Subfamily:   %s\n", info.Subfamily)
	fmt.Printf("  Full name:   %s\n", info.FullName)
	fmt.Printf("  PostScript:  %s\n", info.PostScriptName)
	fmt.Printf("  Version:     %s\n", info.Version)
	fmt.Printf("  Weight:      %d\n", info.Weight)
	fmt.Printf("  Width:       %d\n", info.Width)
	fmt.Printf("  Style:       %d\n", info.StyleFlags)

	fmt.Println("\n=== Metrics ===")
	fmt.Printf("  UnitsPerEm:  %d\n", info.UnitsPerEm)
	fmt.Printf("  Bbox:        [%d, %d] - [%d, %d]\n", info.XMin, info.YMin, info.XMax, info.YMax)
	fmt.Printf("  Ascent:      %d\n", info.Ascent)
	fmt.Printf("  Descent:     %d\n", info.Descent)
	fmt.Printf("  LineGap:     %d\n", info.LineGap)
	fmt.Printf("  CapHeight:   %d\n", info.CapHeight)

	// Glyph ID lookup via cmap.
	fmt.Println("\n=== Glyph IDs ===")
	for _, r := range []rune{'A', 'z', '0', ' ', '€', '→', 0x10FFFF} {
		id := f.GlyphID(r)
		if id != 0 {
			fmt.Printf("  U+%04X %q  → glyph %d\n", r, string(r), id)
		} else {
			fmt.Printf("  U+%04X      → .notdef\n", r)
		}
	}
}
