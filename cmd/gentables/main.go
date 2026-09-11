// Command gentables extracts Unicode property tables from kb_text_shape.h
// and generates Go source files.
//
// Each property is a two-level page table: PageIndices maps codepoint>>shift to
// a page number, and Data holds 1<<shift entries per page. A tier is a
// codepoint range a build cares about; restricting the tables to it is the
// difference between ~273 KB and ~1.4 KB, which is what makes shaping viable on
// a microcontroller. Adding a tier is one entry in [tiers].
//
// The full tier is emitted verbatim from the C source so its output stays
// byte-identical as this command changes; reduced tiers are compacted to the
// pages their range actually reaches.
//
// Usage:
//
//	gentables <kb_text_shape.h> <output-dir>
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// tableSpec is one Unicode property: a page-index array, a data array, and the
// geometry needed to read them.
type tableSpec struct {
	goName   string // identifier stem; arrays are goName+"PageIndices"/"Data"
	field    string // the UnicodeTables field this fills
	cPages   string
	cData    string
	elemType string
	shift    uint   // 7 for 128-entry pages, 5 for mirroring
	highPage uint32 // data offset for codepoints past the page-index array
	comment  string
}

var tables = []tableSpec{
	{"graphemeBreakClass", "grapheme", "kbts__UnicodeGraphemeBreakClass_PageIndices", "kbts__UnicodeGraphemeBreakClass_Data", "uint8", 7, 256, "grapheme break class"},
	{"lineBreakClass", "line", "kbts__UnicodeLineBreakClass_PageIndices", "kbts__UnicodeLineBreakClass_Data", "uint8", 7, 256, "line break class"},
	{"wordBreakClass", "word", "kbts__UnicodeWordBreakClass_PageIndices", "kbts__UnicodeWordBreakClass_Data", "uint8", 7, 7296, "word break class"},
	{"bidiClass", "bidi", "kbts__UnicodeBidirectionalClass_PageIndices", "kbts__UnicodeBidirectionalClass_Data", "uint8", 7, 7296, "bidirectional class"},
	{"scriptExtension", "script", "kbts__UnicodeScriptExtension_PageIndices", "kbts__UnicodeScriptExtension_Data", "uint16", 7, 7808, "script extension"},
	{"unicodeFlags", "flags", "kbts__UnicodeFlags_PageIndices", "kbts__UnicodeFlags_Data", "uint8", 7, 256, "unicode flags"},
	{"mirrorCodepoint", "mirror", "kbts__UnicodeMirrorCodepoint_PageIndices", "kbts__UnicodeMirrorCodepoint_Data", "uint32", 5, 0, "mirror codepoint"},
	{"joiningType", "joining", "kbts__UnicodeJoiningType_PageIndices", "kbts__UnicodeJoiningType_Data", "uint8", 7, 0, "Arabic joining type"},
}

// maxRuneFull is the exclusive codepoint bound every full table answers within.
const maxRuneFull = 1114110

// tier is a codepoint range a build may restrict itself to.
type tier struct {
	name   string // constructor suffix: UnicodeFull, UnicodeASCII
	suffix string // identifier suffix; empty for the full tier
	maxCP  uint32 // inclusive
	file   string
	doc    string
}

var tiers = []tier{
	{"Full", "", 0x10FFFD, "unicode_tables.go", "every codepoint. Roughly 273 KB of tables."},
	{"ASCII", "ASCII", 0x7F, "unicode_tables_ascii.go", "U+0000..U+007F. Roughly 1.4 KB of tables."},
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: gentables <input.h> <output-dir>")
	}
	inputPath, outDir := os.Args[1], os.Args[2]

	data, err := os.ReadFile(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	content := string(data)

	emitted := make(map[string]map[string]bool, len(tiers)) // tier suffix -> goName -> present
	for _, tr := range tiers {
		present := map[string]bool{}
		path := filepath.Join(outDir, tr.file)
		err := withFile(path, func(w *bufio.Writer) {
			fmt.Fprintf(w, "// %s tables: %s\n\n", tr.name, tr.doc)
			for _, t := range tables {
				if emitTable(w, content, t, tr) {
					present[t.goName] = true
				}
			}
			if tr.suffix == "" {
				emitShared(w, content)
			}
		})
		if err != nil {
			log.Fatal(err)
		}
		emitted[tr.suffix] = present
		fmt.Println("Generated", path)
	}

	// Constructors are generated because they must stay in sync with tiers; a
	// hand-maintained one is exactly where adding a tier would break.
	setsPath := filepath.Join(outDir, "unicode_sets.go")
	err = withFile(setsPath, func(w *bufio.Writer) {
		for _, tr := range tiers {
			emitConstructor(w, tr, emitted[tr.suffix])
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Generated", setsPath)
}

// emitTable writes one property's arrays for one tier and reports whether it
// emitted anything. The full tier is copied through as source text so its output
// does not churn; reduced tiers are compacted, and one that turns out to be all
// zero is dropped outright — the zero pageTable already answers zero for every
// codepoint, so an omitted table costs neither bytes nor a branch.
func emitTable(w *bufio.Writer, content string, t tableSpec, tr tier) bool {
	if tr.suffix == "" {
		pages := extractArray(content, t.cPages)
		vals := extractArray(content, t.cData)
		if pages == "" || vals == "" {
			log.Printf("WARNING: table %s not found", t.goName)
			return false
		}
		fmt.Fprintf(w, "// %s page indices\n", t.comment)
		fmt.Fprintf(w, "var %sPageIndices = [...]uint8{%s}\n\n", t.goName, pages)
		fmt.Fprintf(w, "// %s data\n", t.comment)
		fmt.Fprintf(w, "var %sData = [...]%s{%s}\n\n", t.goName, t.elemType, vals)
		return true
	}

	pages, err := parseArray(content, t.cPages)
	if err != nil {
		log.Fatalf("%s: %v", t.cPages, err)
	}
	vals, err := parseArray(content, t.cData)
	if err != nil {
		log.Fatalf("%s: %v", t.cData, err)
	}
	pages, vals = compact(pages, vals, t.shift, tr.maxCP)
	if allZero(vals) {
		return false
	}
	fmt.Fprintf(w, "// %s page indices\n", t.comment)
	fmt.Fprintf(w, "var %sPageIndices%s = [...]uint8{%s}\n\n", t.goName, tr.suffix, join(pages))
	fmt.Fprintf(w, "// %s data\n", t.comment)
	fmt.Fprintf(w, "var %sData%s = [...]%s{%s}\n\n", t.goName, tr.suffix, t.elemType, join(vals))
	return true
}

// emitConstructor writes the UnicodeX function for one tier. Tables the tier
// omitted are left as the zero pageTable, which answers zero everywhere.
func emitConstructor(w *bufio.Writer, tr tier, present map[string]bool) {
	fmt.Fprintf(w, "// Unicode%s returns the table set covering %s\n", tr.name, tr.doc)
	fmt.Fprintln(w, "//")
	fmt.Fprintln(w, "// It is a function, not a variable, and must stay one: a package-level")
	fmt.Fprintln(w, "// variable holding a [UnicodeTables] forces a package init that references")
	fmt.Fprintln(w, "// every table, and the linker would then retain all of them even for a")
	fmt.Fprintln(w, "// program that only ever asks for the smallest set.")
	fmt.Fprintf(w, "func Unicode%s() UnicodeTables {\n", tr.name)
	fmt.Fprintln(w, "\treturn UnicodeTables{")
	fmt.Fprintf(w, "\t\tMaxRune: 0x%X,\n", tr.maxCP)
	for _, t := range tables {
		if !present[t.goName] {
			continue
		}
		maxCP, high := uint32(maxRuneFull), t.highPage
		if tr.suffix != "" {
			// MaxRune gates before the high-page branch can be reached.
			maxCP, high = tr.maxCP+1, 0
		}
		fmt.Fprintf(w, "\t\t%s: pageTable[%s]{pages: %sPageIndices%s[:], data: %sData%s[:], maxCP: %d, highPage: %d, shift: %d},\n",
			t.field, t.elemType, t.goName, tr.suffix, t.goName, tr.suffix, maxCP, high, t.shift)
	}
	if tr.suffix == "" {
		// Reduced tiers reach no codepoint with more than one script extension,
		// so the list they would index into is dead weight.
		fmt.Fprintln(w, "\t\tscriptExt: scriptExtensions[:],")
	}
	fmt.Fprintln(w, "\t}")
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)
}

// emitShared writes the tables indexed by class or script rather than by
// codepoint, which are therefore the same for every tier.
func emitShared(w *bufio.Writer, content string) {
	if s := extractArray(content, "kbts__ScriptExtensions"); s != "" {
		fmt.Fprintln(w, "// script extensions array")
		fmt.Fprintf(w, "var scriptExtensions = [...]uint8{%s}\n\n", s)
	}
	if s := extractNestedArray(content, "kbts_GraphemeBreakTransition"); s != "" {
		fmt.Fprintln(w, "// grapheme break transition table [class][state]")
		fmt.Fprintf(w, "var graphemeBreakTransition = %s\n\n", s)
	}
}

// compact restricts a page table to codepoints <= maxCP, keeping only the pages
// that range reaches and renumbering the indices to match. Pages are kept in
// first-appearance order so the output is deterministic.
func compact(pages, data []uint64, shift uint, maxCP uint32) (outPages, outData []uint64) {
	n := int(maxCP>>shift) + 1
	if n > len(pages) {
		n = len(pages)
	}
	pageSize := 1 << shift
	remap := make(map[uint64]int, n)
	outPages = make([]uint64, n)
	for i, p := range pages[:n] {
		idx, ok := remap[p]
		if !ok {
			idx = len(remap)
			remap[p] = idx
			base := int(p) << shift
			if base+pageSize > len(data) {
				log.Fatalf("page %d out of range (data len %d)", p, len(data))
			}
			outData = append(outData, data[base:base+pageSize]...)
		}
		outPages[i] = uint64(idx)
	}
	return outPages, outData
}

func allZero(v []uint64) bool {
	for _, x := range v {
		if x != 0 {
			return false
		}
	}
	return true
}

func join(v []uint64) string {
	var sb strings.Builder
	for i, x := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		if i%32 == 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(strconv.FormatUint(x, 10))
	}
	return sb.String()
}

func withFile(path string, fn func(*bufio.Writer)) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	fmt.Fprintln(w, "// Code generated by cmd/gentables; DO NOT EDIT.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "package lefevre")
	fmt.Fprintln(w)
	fn(w)
	return w.Flush()
}

// parseArray returns the numeric values of a C array initializer.
func parseArray(content, name string) ([]uint64, error) {
	inner := extractArray(content, name)
	if inner == "" {
		return nil, fmt.Errorf("not found")
	}
	fields := strings.FieldsFunc(inner, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	vals := make([]uint64, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimRight(f, "uUlL")
		if f == "" {
			continue
		}
		v, err := strconv.ParseUint(f, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("value %q: %w", f, err)
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func extractArray(content, name string) string {
	idx := strings.Index(content, name+"[")
	if idx < 0 {
		idx = strings.Index(content, name+" [")
		if idx < 0 {
			return ""
		}
	}
	braceStart := strings.Index(content[idx:], "{")
	if braceStart < 0 {
		return ""
	}
	braceStart += idx

	depth := 0
	end := braceStart
	for i := braceStart; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}

	inner := removeCComments(content[braceStart+1 : end])
	inner = strings.TrimSpace(inner)
	return strings.TrimRight(inner, ", \t\n\r")
}

func extractNestedArray(content, name string) string {
	idx := strings.Index(content, name+"[")
	if idx < 0 {
		idx = strings.Index(content, name+" [")
		if idx < 0 {
			return ""
		}
	}
	braceStart := strings.Index(content[idx:], "{")
	if braceStart < 0 {
		return ""
	}
	braceStart += idx

	depth := 0
	end := braceStart
	for i := braceStart; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}

	inner := removeCComments(content[braceStart+1 : end])

	var rows []string
	for {
		rowStart := strings.Index(inner, "{")
		if rowStart < 0 {
			break
		}
		rowEnd := strings.Index(inner[rowStart:], "}")
		if rowEnd < 0 {
			break
		}
		rowEnd += rowStart
		row := strings.TrimRight(strings.TrimSpace(inner[rowStart+1:rowEnd]), ", ")
		rows = append(rows, row)
		inner = inner[rowEnd+1:]
	}
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "[%d][%d]uint8{\n", len(rows), countFields(rows[0]))
	for _, row := range rows {
		sb.WriteString("\t{")
		sb.WriteString(row)
		sb.WriteString("},\n")
	}
	sb.WriteString("}")
	return sb.String()
}

func countFields(row string) int {
	count := 0
	for _, p := range strings.Split(row, ",") {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	return count
}

func removeCComments(s string) string {
	for {
		start := strings.Index(s, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], "*/")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+end+2:]
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}
