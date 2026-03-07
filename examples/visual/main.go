// Example visual renders shaped text as an SVG with glyph outlines.
//
// Usage:
//
//	echo "Hello, World!" | go run ./examples/visual testdata/DejaVuSans.ttf > out.svg
//	go run ./examples/visual -text-file input.txt testdata/DejaVuSans.ttf > out.svg
//	go run ./examples/visual testdata/DejaVuSans.ttf > out.svg   # uses default demo text
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	kb "github.com/soypat/lefevre"
)

func main() {
	textFile := flag.String("text-file", "", "read input text from file instead of stdin")
	noLigatures := flag.Bool("noligatures", false, "disable ligature substitution")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: visual [flags] <font.ttf>")
		os.Exit(1)
	}
	fontPath := flag.Arg(0)

	// Determine input text.
	var text string
	switch {
	case *textFile != "":
		b, err := os.ReadFile(*textFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		text = strings.TrimRight(string(b), "\r\n")
	case !isTerminal(os.Stdin):
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		text = strings.TrimRight(string(b), "\r\n")
	default:
		text = "Hello, World! Office ffi"
	}

	// Load font.
	data, err := os.ReadFile(fontPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	font, err := kb.FontFromMemory(data, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "font parse error:", err)
		os.Exit(1)
	}
	info := font.Info()

	// Shape text.
	dir, _ := kb.GuessTextProperties(text)
	var cfg kb.ShapeConfig
	cfg.Font = font
	if *noLigatures {
		cfg.Features = []kb.FeatureOverride{{Tag: kb.FeatureTagLiga, Value: 0}}
	}
	runs := cfg.ShapeSimple(nil, text, dir)

	// Parse raw font tables for glyph outlines.
	headOff, _ := findTable(data, "head")
	locaOff, _ := findTable(data, "loca")
	glyfOff, _ := findTable(data, "glyf")
	maxpOff, _ := findTable(data, "maxp")
	if headOff == 0 || locaOff == 0 || glyfOff == 0 || maxpOff == 0 {
		fmt.Fprintln(os.Stderr, "font missing required tables (head/loca/glyf/maxp)")
		os.Exit(1)
	}
	longLoca := readS16BE(data, int(headOff)+50) == 1
	numGlyphs := int(readU16BE(data, int(maxpOff)+4))

	// Collect all glyphs with their cursor positions.
	type posGlyph struct {
		g        kb.Glyph
		cursorX  int32
		ligature bool
	}
	var glyphs []posGlyph
	var cursorX int32
	for _, run := range runs {
		for _, g := range run.Glyphs {
			glyphs = append(glyphs, posGlyph{
				g:        g,
				cursorX:  cursorX,
				ligature: g.Flags.Has(kb.GlyphFlagLigature),
			})
			cursorX += g.AdvanceX
		}
	}
	totalAdvance := cursorX

	// SVG dimensions.
	ascent := int32(info.Ascent)
	descent := int32(info.Descent) // negative
	height := ascent - descent
	padding := int32(float64(height) * 0.15)

	svgW := float64(totalAdvance+padding*2) * 0.05
	svgH := float64(height+padding*2) * 0.05
	if svgW < 100 {
		svgW = 100
	}

	w := os.Stdout
	fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="%.0f" height="%.0f">`+"\n",
		svgW, svgH, svgW*2, svgH*2)
	// Background.
	fmt.Fprintf(w, `  <rect width="100%%" height="100%%" fill="#1a1a2e"/>`+"\n")

	// Scale factor: map font units to SVG units.
	scale := svgW / float64(totalAdvance+padding*2)
	baselineY := float64(ascent+padding) * scale

	// Baseline.
	fmt.Fprintf(w, `  <line x1="0" y1="%.2f" x2="%.1f" y2="%.2f" stroke="#ff6b6b" stroke-width="0.5" opacity="0.4"/>`+"\n",
		baselineY, svgW, baselineY)
	// Ascent line.
	ascentY := float64(padding) * scale
	fmt.Fprintf(w, `  <line x1="0" y1="%.2f" x2="%.1f" y2="%.2f" stroke="#4ecdc4" stroke-width="0.3" opacity="0.3"/>`+"\n",
		ascentY, svgW, ascentY)
	// Descent line.
	descentY := float64(ascent-descent+padding) * scale
	fmt.Fprintf(w, `  <line x1="0" y1="%.2f" x2="%.1f" y2="%.2f" stroke="#4ecdc4" stroke-width="0.3" opacity="0.3"/>`+"\n",
		descentY, svgW, descentY)

	// Render each glyph.
	for _, pg := range glyphs {
		cx := float64(pg.cursorX+padding) * scale
		advW := float64(pg.g.AdvanceX) * scale

		// Advance box.
		boxColor := "100,180,255" // blue
		if pg.ligature {
			boxColor = "255,165,0" // orange for ligatures
		}
		fmt.Fprintf(w, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="rgba(%s,0.12)" stroke="rgba(%s,0.35)" stroke-width="0.3"/>`+"\n",
			cx, ascentY, advW, descentY-ascentY, boxColor, boxColor)

		// Glyph outline.
		if int(pg.g.ID) >= numGlyphs {
			continue
		}
		outline := parseGlyph(data, locaOff, glyfOff, pg.g.ID, longLoca, numGlyphs, 0)
		if len(outline.contours) == 0 {
			continue
		}

		// Build SVG path.
		var pathD string
		for _, contour := range outline.contours {
			pathD += contourToSVGPath(contour)
		}

		offX := float64(pg.cursorX+padding+pg.g.OffsetX) * scale
		offY := baselineY // baseline in SVG coords
		// Glyph paths are in font units (Y-up). We scale and flip Y.
		fmt.Fprintf(w, `  <path d="%s" transform="translate(%.2f,%.2f) scale(%.6f,%.6f)" fill="white"/>`+"\n",
			pathD, offX, offY, scale, -scale)
	}

	fmt.Fprintln(w, `</svg>`)
}

// --- Binary reading helpers ---

func readU16BE(d []byte, off int) uint16 {
	return uint16(d[off])<<8 | uint16(d[off+1])
}

func readS16BE(d []byte, off int) int16 {
	return int16(readU16BE(d, off))
}

func readU32BE(d []byte, off int) uint32 {
	return uint32(d[off])<<24 | uint32(d[off+1])<<16 | uint32(d[off+2])<<8 | uint32(d[off+3])
}

// findTable locates an OpenType table by its 4-byte ASCII tag.
func findTable(data []byte, tag string) (offset, length uint32) {
	if len(data) < 12 || len(tag) != 4 {
		return 0, 0
	}
	wantTag := uint32(tag[0])<<24 | uint32(tag[1])<<16 | uint32(tag[2])<<8 | uint32(tag[3])
	magic := readU32BE(data, 0)
	var dirOff int
	switch magic {
	case 0x00010000, 0x4F54544F: // TTF, OTTO
		dirOff = 0
	case 0x74746366: // ttcf
		dirOff = int(readU32BE(data, 12)) // first font
	default:
		return 0, 0
	}
	numTables := int(readU16BE(data, dirOff+4))
	for i := 0; i < numTables; i++ {
		rec := dirOff + 12 + i*16
		if rec+16 > len(data) {
			break
		}
		if readU32BE(data, rec) == wantTag {
			return readU32BE(data, rec+8), readU32BE(data, rec+12)
		}
	}
	return 0, 0
}

// --- Glyph outline parsing ---

type point struct {
	x, y    int16
	onCurve bool
}

type glyphOutline struct {
	contours           [][]point
	xMin, yMin         int16
	xMax, yMax         int16
}

// glyphDataRange returns the offset and length of a glyph's data in the glyf table.
func glyphDataRange(data []byte, locaOff, glyfOff uint32, glyphID uint16, longLoca bool, numGlyphs int) (off, length uint32) {
	if int(glyphID) >= numGlyphs {
		return 0, 0
	}
	var start, end uint32
	if longLoca {
		idx := int(locaOff) + int(glyphID)*4
		if idx+8 > len(data) {
			return 0, 0
		}
		start = readU32BE(data, idx)
		end = readU32BE(data, idx+4)
	} else {
		idx := int(locaOff) + int(glyphID)*2
		if idx+4 > len(data) {
			return 0, 0
		}
		start = uint32(readU16BE(data, idx)) * 2
		end = uint32(readU16BE(data, idx+2)) * 2
	}
	if end <= start {
		return 0, 0 // empty glyph (e.g., space)
	}
	return glyfOff + start, end - start
}

const maxCompositeDepth = 8

func parseGlyph(data []byte, locaOff, glyfOff uint32, glyphID uint16, longLoca bool, numGlyphs int, depth int) glyphOutline {
	off, length := glyphDataRange(data, locaOff, glyfOff, glyphID, longLoca, numGlyphs)
	if length == 0 || int(off)+int(length) > len(data) {
		return glyphOutline{}
	}
	p := int(off)
	nContours := readS16BE(data, p)
	xMin := readS16BE(data, p+2)
	yMin := readS16BE(data, p+4)
	xMax := readS16BE(data, p+6)
	yMax := readS16BE(data, p+8)
	p += 10

	if nContours >= 0 {
		return parseSimpleGlyph(data, p, int(nContours), xMin, yMin, xMax, yMax)
	}
	if depth >= maxCompositeDepth {
		return glyphOutline{}
	}
	return parseCompositeGlyph(data, p, locaOff, glyfOff, longLoca, numGlyphs, depth, xMin, yMin, xMax, yMax)
}

func parseSimpleGlyph(data []byte, p int, nContours int, xMin, yMin, xMax, yMax int16) glyphOutline {
	if nContours == 0 {
		return glyphOutline{}
	}
	// Read endPtsOfContours.
	if p+nContours*2 > len(data) {
		return glyphOutline{}
	}
	endPts := make([]int, nContours)
	for i := 0; i < nContours; i++ {
		endPts[i] = int(readU16BE(data, p+i*2))
	}
	p += nContours * 2
	totalPoints := endPts[nContours-1] + 1

	// Skip instructions.
	if p+2 > len(data) {
		return glyphOutline{}
	}
	instrLen := int(readU16BE(data, p))
	p += 2 + instrLen
	if p > len(data) {
		return glyphOutline{}
	}

	// Parse flags.
	flags := make([]byte, totalPoints)
	for i := 0; i < totalPoints; {
		if p >= len(data) {
			return glyphOutline{}
		}
		f := data[p]
		p++
		flags[i] = f
		i++
		if f&0x08 != 0 { // repeat
			if p >= len(data) {
				return glyphOutline{}
			}
			repeat := int(data[p])
			p++
			for r := 0; r < repeat && i < totalPoints; r++ {
				flags[i] = f
				i++
			}
		}
	}

	// Parse X coordinates.
	xs := make([]int16, totalPoints)
	var xAccum int16
	for i := 0; i < totalPoints; i++ {
		f := flags[i]
		if f&0x02 != 0 { // short
			if p >= len(data) {
				return glyphOutline{}
			}
			d := int16(data[p])
			p++
			if f&0x10 == 0 {
				d = -d
			}
			xAccum += d
		} else if f&0x10 == 0 { // long delta
			if p+2 > len(data) {
				return glyphOutline{}
			}
			xAccum += readS16BE(data, p)
			p += 2
		}
		// else: same as previous (delta = 0)
		xs[i] = xAccum
	}

	// Parse Y coordinates.
	ys := make([]int16, totalPoints)
	var yAccum int16
	for i := 0; i < totalPoints; i++ {
		f := flags[i]
		if f&0x04 != 0 { // short
			if p >= len(data) {
				return glyphOutline{}
			}
			d := int16(data[p])
			p++
			if f&0x20 == 0 {
				d = -d
			}
			yAccum += d
		} else if f&0x20 == 0 { // long delta
			if p+2 > len(data) {
				return glyphOutline{}
			}
			yAccum += readS16BE(data, p)
			p += 2
		}
		ys[i] = yAccum
	}

	// Build contours.
	contours := make([][]point, nContours)
	start := 0
	for c := 0; c < nContours; c++ {
		end := endPts[c]
		n := end - start + 1
		pts := make([]point, n)
		for i := 0; i < n; i++ {
			idx := start + i
			pts[i] = point{
				x:       xs[idx],
				y:       ys[idx],
				onCurve: flags[idx]&0x01 != 0,
			}
		}
		contours[c] = pts
		start = end + 1
	}
	return glyphOutline{contours: contours, xMin: xMin, yMin: yMin, xMax: xMax, yMax: yMax}
}

// Composite glyph flags.
const (
	compArg1And2AreWords = 1 << 0
	compArgsAreXYValues  = 1 << 1
	compWeHaveAScale     = 1 << 3
	compMoreComponents   = 1 << 5
	compWeHaveAnXYScale  = 1 << 6
	compWeHaveATwoByTwo  = 1 << 7
)

func parseCompositeGlyph(data []byte, p int, locaOff, glyfOff uint32, longLoca bool, numGlyphs int, depth int, xMin, yMin, xMax, yMax int16) glyphOutline {
	var out glyphOutline
	out.xMin, out.yMin, out.xMax, out.yMax = xMin, yMin, xMax, yMax

	for {
		if p+4 > len(data) {
			break
		}
		cflags := readU16BE(data, p)
		glyphIdx := readU16BE(data, p+2)
		p += 4

		var dx, dy int16
		if cflags&compArg1And2AreWords != 0 {
			if p+4 > len(data) {
				break
			}
			if cflags&compArgsAreXYValues != 0 {
				dx = readS16BE(data, p)
				dy = readS16BE(data, p+2)
			}
			p += 4
		} else {
			if p+2 > len(data) {
				break
			}
			if cflags&compArgsAreXYValues != 0 {
				dx = int16(int8(data[p]))
				dy = int16(int8(data[p+1]))
			}
			p += 2
		}

		// Skip scale/matrix data.
		if cflags&compWeHaveAScale != 0 {
			p += 2
		} else if cflags&compWeHaveAnXYScale != 0 {
			p += 4
		} else if cflags&compWeHaveATwoByTwo != 0 {
			p += 8
		}

		// Recursively parse the component glyph.
		comp := parseGlyph(data, locaOff, glyfOff, glyphIdx, longLoca, numGlyphs, depth+1)
		for _, contour := range comp.contours {
			translated := make([]point, len(contour))
			for i, pt := range contour {
				translated[i] = point{x: pt.x + dx, y: pt.y + dy, onCurve: pt.onCurve}
			}
			out.contours = append(out.contours, translated)
		}

		if cflags&compMoreComponents == 0 {
			break
		}
	}
	return out
}

// --- SVG path generation ---

func contourToSVGPath(pts []point) string {
	n := len(pts)
	if n < 2 {
		return ""
	}

	var b strings.Builder
	// Find first on-curve point or compute implied start.
	startIdx := -1
	for i, pt := range pts {
		if pt.onCurve {
			startIdx = i
			break
		}
	}

	var startX, startY float64
	var walkStart int
	if startIdx >= 0 {
		startX = float64(pts[startIdx].x)
		startY = float64(pts[startIdx].y)
		walkStart = startIdx + 1
	} else {
		// All off-curve: start at midpoint of first two.
		startX = float64(pts[0].x+pts[1].x) / 2
		startY = float64(pts[0].y+pts[1].y) / 2
		walkStart = 1
	}
	fmt.Fprintf(&b, "M%.1f %.1f", startX, startY)

	// Walk points starting from walkStart, wrapping around.
	prevOnCurve := true
	var prevX, prevY float64 // off-curve control point
	for step := 0; step < n; step++ {
		idx := (walkStart + step) % n
		pt := pts[idx]
		px, py := float64(pt.x), float64(pt.y)

		if pt.onCurve {
			if prevOnCurve {
				fmt.Fprintf(&b, "L%.1f %.1f", px, py)
			} else {
				fmt.Fprintf(&b, "Q%.1f %.1f %.1f %.1f", prevX, prevY, px, py)
			}
			prevOnCurve = true
		} else {
			if !prevOnCurve {
				// Implied on-curve midpoint.
				mx := (prevX + px) / 2
				my := (prevY + py) / 2
				fmt.Fprintf(&b, "Q%.1f %.1f %.1f %.1f", prevX, prevY, mx, my)
			}
			prevX, prevY = px, py
			prevOnCurve = false
		}
	}

	// Close: if last point was off-curve, curve to start.
	if !prevOnCurve {
		fmt.Fprintf(&b, "Q%.1f %.1f %.1f %.1f", prevX, prevY, startX, startY)
	}
	b.WriteString("Z")
	return b.String()
}

// isTerminal returns true if the file is a terminal (not a pipe/redirect).
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return true
	}
	return stat.Mode()&os.ModeCharDevice != 0
}
