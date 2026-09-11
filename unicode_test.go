package lefevre

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// asciiText is every codepoint a restricted set is expected to describe.
func asciiText() string {
	var sb strings.Builder
	for r := rune(0); r <= 0x7F; r++ {
		sb.WriteRune(r)
	}
	return sb.String()
}

func breaksWith(t *testing.T, u UnicodeTables, text string, dir Direction) []Break {
	t.Helper()
	b := Breaker{Direction: dir, Unicode: u}
	breaks, n := b.AppendBreak(nil, []byte(text))
	if n != len(text) {
		t.Fatalf("consumed %d of %d bytes", n, len(text))
	}
	breaks = b.End(breaks)
	if err := b.Err(); err != nil {
		t.Fatalf("Breaker.Err = %v", err)
	}
	return breaks
}

// TestASCIITablesAgreeWithFull is the contract that makes the reduced tier
// safe: over the range it covers, it must segment exactly as the full set does.
// It exercises all seven segmentation properties at once, through the exported
// API, which is why it is worth more than per-table unit tests.
func TestASCIITablesAgreeWithFull(t *testing.T) {
	full, ascii := UnicodeFull(), UnicodeASCII()
	for _, tc := range []struct {
		name string
		text string
	}{
		{"all ASCII codepoints", asciiText()},
		{"words", "Hello world! The quick brown fox."},
		{"punctuation", "a,b;c:d.e?f!g"},
		{"brackets", "(a) [b] {c} <d>"},
		{"newlines", "one\ntwo\r\nthree\rfour"},
		{"tabs and spaces", "a\tb  c   d"},
		{"digits", "1234 56.78 9,10"},
		{"hyphenation", "well-known co-op re-use"},
		{"quotes", `he said "hi" and 'bye'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := breaksWith(t, full, tc.text, DirectionLTR)
			got := breaksWith(t, ascii, tc.text, DirectionLTR)
			if len(got) != len(want) {
				t.Fatalf("break count = %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("break %d = %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}
}

// TestASCIIMirrorsBrackets pins the reason the mirror table survives into the
// ASCII tier. Mirroring reads like a bidi concern, so the tempting call is to
// drop it from a Latin-only build; ASCII's own parens, brackets and braces all
// have mirrors, and dropping it would silently stop them pairing.
func TestASCIIMirrorsBrackets(t *testing.T) {
	u := UnicodeASCII()
	for _, pair := range []struct{ open, close rune }{
		{'(', ')'}, {'[', ']'}, {'{', '}'}, {'<', '>'},
	} {
		if got := u.mirrorCodepoint(pair.open); got != uint32(pair.close) {
			t.Errorf("mirror(%q) = %d, want %q", pair.open, got, pair.close)
		}
		if got := u.mirrorCodepoint(pair.close); got != uint32(pair.open) {
			t.Errorf("mirror(%q) = %d, want %q", pair.close, got, pair.open)
		}
	}
}

// TestUnicodeASCIIOmitsEmptyTables records that a table which is all zero over
// the tier's range is left out entirely. The zero pageTable already answers
// zero, so omitting it costs neither bytes nor a branch.
func TestUnicodeASCIIOmitsEmptyTables(t *testing.T) {
	u := UnicodeASCII()
	if len(u.joining.data) != 0 {
		t.Errorf("ASCII set carries %d bytes of joining data; it is zero over ASCII", len(u.joining.data))
	}
	if len(u.scriptExt) != 0 {
		t.Errorf("ASCII set carries a script extension list; no ASCII codepoint has more than one script")
	}
	// Omitted tables must still answer, not fault.
	for r := rune(0); r <= 0x7F; r++ {
		if got := u.joiningType(r); got != joiningTypeNone {
			t.Fatalf("joiningType(%q) = %d, want %d", r, got, joiningTypeNone)
		}
	}
}

// TestUnicodeTablesOutOfRange checks that a set answers zero rather than
// faulting for codepoints past its range, so a restricted build cannot be
// crashed by arbitrary input.
func TestUnicodeTablesOutOfRange(t *testing.T) {
	sets := map[string]UnicodeTables{
		"ascii": UnicodeASCII(),
		"full":  UnicodeFull(),
		"zero":  {},
	}
	probes := []rune{0x80, 0x100, 'é', '中', 0x0627, 0x1F600, 0x10FFFF, 0x110000, -1}
	for name, u := range sets {
		for _, r := range probes {
			if u.Covered(r) {
				continue
			}
			// Must not panic, and must read as zero.
			if got := u.bidiClass(r); got != 0 {
				t.Errorf("%s: bidiClass(%U) = %d, want 0 for uncovered rune", name, r, got)
			}
			if got := u.mirrorCodepoint(r); got != 0 {
				t.Errorf("%s: mirrorCodepoint(%U) = %d, want 0 for uncovered rune", name, r, got)
			}
			_ = u.graphemeBreakClass(r)
			_ = u.lineBreakClass(r)
			_ = u.wordBreakClass(r)
			_ = u.unicodeFlags(r)
			_ = u.scriptExtension(r)
			_ = u.joiningType(r)
		}
	}
}

// TestBreakerWithoutTablesReportsError pins the chosen failure mode for a
// zero-value Breaker. It cannot default to the full tables, because reaching
// them from AppendBreak would make every build carry all of them; so it must
// fail loudly instead. It must still report the input consumed, or a caller
// draining a stream never terminates.
func TestBreakerWithoutTablesReportsError(t *testing.T) {
	var b Breaker
	text := []byte("hello world")
	breaks, n := b.AppendBreak(nil, text)
	if n != len(text) {
		t.Errorf("consumed %d bytes, want %d; a short count spins a drain loop forever", n, len(text))
	}
	if len(breaks) != 0 {
		t.Errorf("emitted %d breaks with no tables, want 0", len(breaks))
	}
	if b.Err() != ErrNoUnicodeTables {
		t.Errorf("Err = %v, want %v", b.Err(), ErrNoUnicodeTables)
	}
}

// TestASCIISetIsSmall guards the point of the tier. The numbers are generous;
// the test exists to catch a table creeping back in, not to police bytes.
func TestASCIISetIsSmall(t *testing.T) {
	u := UnicodeASCII()
	total := len(u.grapheme.data) + len(u.line.data) + len(u.word.data) +
		len(u.bidi.data) + len(u.flags.data) + len(u.joining.data) +
		2*len(u.script.data) + 4*len(u.mirror.data) + len(u.scriptExt)
	if total > 4096 {
		t.Errorf("ASCII tables are %d bytes; expected well under 4 KB", total)
	}
	t.Logf("ASCII table bytes: %d", total)
}

// TestBinaryOmitsFullTablesWhenOnlyASCIIUsed is the only real guard on the
// rule that table sets must be reached through functions and never through
// package-level variables. A variable holding a UnicodeTables forces a package
// init that references every table, and the linker then keeps all ~273 KB even
// for a program that asked only for the ASCII set. That failure is invisible in
// every other test: the code still works, it just ships a quarter megabyte.
func TestBinaryOmitsFullTablesWhenOnlyASCIIUsed(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary")
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not on PATH")
	}
	repo, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module dcefixture\n\ngo 1.25.7\n\nrequire github.com/soypat/lefevre v0.0.0\n\nreplace github.com/soypat/lefevre => "+repo+"\n")
	write("main.go", `package main

import (
	"fmt"

	"github.com/soypat/lefevre"
)

func main() {
	br := lefevre.Breaker{Direction: lefevre.DirectionLTR, Unicode: lefevre.UnicodeASCII()}
	breaks, _ := br.AppendBreak(nil, []byte("hello world"))
	breaks = br.End(breaks)
	fmt.Println(len(breaks))
}
`)

	bin := filepath.Join(dir, "fixture")
	cmd := exec.Command(goTool, "build", "-o", bin, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, out)
	}

	cmd = exec.Command(goTool, "tool", "nm", "-size", bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go tool nm: %v\n%s", err, out)
	}
	syms := string(out)

	// Collect symbol names, which nm prints last on each line and fully
	// qualified. Matching must be exact: every full-tier name is a prefix of
	// its ASCII counterpart, so a substring test would never fail.
	linked := map[string]bool{}
	for _, line := range strings.Split(syms, "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			linked[f[len(f)-1]] = true
		}
	}

	const pkg = "github.com/soypat/lefevre."
	// The full tier's data arrays are the ~273 KB that must not be linked.
	for _, name := range []string{
		"graphemeBreakClassData",
		"lineBreakClassData",
		"wordBreakClassData",
		"bidiClassData",
		"unicodeFlagsData",
		"scriptExtensionData",
		"mirrorCodepointData",
		"joiningTypeData",
	} {
		if linked[pkg+name] {
			t.Errorf("full-tier table %s is linked into an ASCII-only binary;\n"+
				"something reaches UnicodeFull from the ASCII path, most likely a\n"+
				"package-level variable holding a UnicodeTables", name)
		}
	}
	// Sanity: without these the test would pass against a binary that linked
	// no tables at all, and so would be checking nothing.
	if !linked[pkg+"bidiClassDataASCII"] {
		t.Error("ASCII tables absent from the fixture; the test is not exercising what it claims")
	}
}
