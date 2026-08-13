package sfnt_test

import (
	"testing"

	"github.com/soypat/lefevre/sfnt"
)

// tableRecord finds the named table in an assembled font and returns the
// checksum and the bytes the directory points at, the way any consumer of the
// file reads it back.
func tableRecord(t *testing.T, font []byte, tag string) (checksum uint32, data []byte) {
	t.Helper()
	if len(font) < 12 {
		t.Fatalf("font is %d bytes, too short for a directory", len(font))
	}
	n := int(font[4])<<8 | int(font[5])
	for i := range n {
		rec := 12 + 16*i
		if rec+16 > len(font) {
			t.Fatalf("directory record %d runs past the %d byte font", i, len(font))
		}
		if string(font[rec:rec+4]) != tag {
			continue
		}
		be32 := func(b []byte) uint32 {
			return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		}
		off, length := be32(font[rec+8:]), be32(font[rec+12:])
		if uint64(off)+uint64(length) > uint64(len(font)) {
			t.Fatalf("%q runs to %d in a %d byte font", tag, uint64(off)+uint64(length), len(font))
		}
		return be32(font[rec+4:]), font[off : off+length]
	}
	t.Fatalf("font carries no %q table", tag)
	return 0, nil
}

// headOfSource is a head table as it comes out of a real font: every shipped
// font has a checkSumAdjustment already computed for the file it came from.
func headOfSource() []byte {
	head := make([]byte, 54)
	head[8], head[9], head[10], head[11] = 0xDE, 0xAD, 0xBE, 0xEF
	head[12], head[13], head[14], head[15] = 0x5F, 0x0F, 0x3C, 0xF5 // magicNumber
	return head
}

// Append fills in head.checkSumAdjustment, so a caller may pass a head straight
// out of a font. The whole-file sum is defined over one whose field reads zero.
func TestAppendZeroesInheritedCheckSumAdjustment(t *testing.T) {
	font, err := sfnt.Append(nil, []sfnt.Table{
		{Tag: "head", Data: headOfSource()},
		{Tag: "maxp", Data: make([]byte, 6)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := sfnt.Checksum(font); got != 0xB1B0AFBA {
		t.Errorf("whole-file checksum = %#08x, want 0xb1b0afba (off by %#08x)", got, 0xB1B0AFBA-got)
	}
}

// head's directory checksum is the one not taken over the bytes as written, but
// over a zeroed field. fontTools recomputes it and rejects a file that disagrees.
func TestAppendChecksumsHeadWithAZeroedAdjustment(t *testing.T) {
	font, err := sfnt.Append(nil, []sfnt.Table{
		{Tag: "head", Data: headOfSource()},
		{Tag: "maxp", Data: make([]byte, 6)},
	})
	if err != nil {
		t.Fatal(err)
	}
	declared, head := tableRecord(t, font, "head")
	if len(head) < 54 {
		t.Fatalf("head is %d bytes, want 54", len(head))
	}
	zeroed := append([]byte(nil), head...)
	zeroed[8], zeroed[9], zeroed[10], zeroed[11] = 0, 0, 0, 0
	if want := sfnt.Checksum(zeroed); declared != want {
		t.Errorf("directory declares head checksum %#08x, want %#08x", declared, want)
	}
}

// A subset rewrites its own head, so it must stay right whatever Append does
// about an inherited adjustment.
func TestSubsetChecksumsAgreeWithTheDirectory(t *testing.T) {
	_, _, program := subset(t, "DejaVuSans.ttf", "Hello", nil)
	if got := sfnt.Checksum(program); got != 0xB1B0AFBA {
		t.Errorf("whole-file checksum = %#08x, want 0xb1b0afba", got)
	}
	for _, tag := range []string{"glyf", "loca", "hmtx", "maxp"} {
		declared, data := tableRecord(t, program, tag)
		if want := sfnt.Checksum(data); declared != want {
			t.Errorf("directory declares %q checksum %#08x, want %#08x", tag, declared, want)
		}
	}
	declared, head := tableRecord(t, program, "head")
	zeroed := append([]byte(nil), head...)
	zeroed[8], zeroed[9], zeroed[10], zeroed[11] = 0, 0, 0, 0
	if want := sfnt.Checksum(zeroed); declared != want {
		t.Errorf("directory declares head checksum %#08x, want %#08x", declared, want)
	}
}

// A tag is four characters, always. "cvt" for "cvt " is the typo the package
// warns about twice, and it cannot be written down: the directory has no room
// for a fifth character and no meaning for a third.
func TestAppendRejectsTagsThatAreNotFourCharacters(t *testing.T) {
	for _, tag := range []string{"cvt", "", "glyf!", "post\x00"} {
		out, err := sfnt.Append(nil, []sfnt.Table{
			{Tag: "head", Data: make([]byte, 54)},
			{Tag: tag, Data: make([]byte, 4)},
		})
		if err == nil {
			t.Errorf("Append with tag %q = nil error, want a failure (wrote %d bytes)", tag, len(out))
		}
	}
}

// Two records for one tag is a directory no reader can resolve.
func TestAppendRejectsDuplicateTags(t *testing.T) {
	if _, err := sfnt.Append(nil, []sfnt.Table{
		{Tag: "head", Data: make([]byte, 54)},
		{Tag: "maxp", Data: make([]byte, 6)},
		{Tag: "maxp", Data: make([]byte, 6)},
	}); err == nil {
		t.Error("Append with two maxp tables = nil error, want a failure")
	}
}

// head is 54 bytes with the adjustment at byte 8, written once the file is
// assembled. Too short to hold it is an error, owed up front, never a panic.
func TestAppendRejectsAShortHead(t *testing.T) {
	for _, n := range []int{0, 4, 8, 11, 53} {
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("Append with a %d byte head panicked: %v", n, p)
				}
			}()
			if _, err := sfnt.Append(nil, []sfnt.Table{
				{Tag: "head", Data: make([]byte, n)},
			}); err == nil {
				t.Errorf("Append with a %d byte head = nil error, want a failure", n)
			}
		}()
	}
}

// Every table is four-aligned and zero-padded, so the per-table checksums
// already sum to the file's and Append needs no second pass to find it.
func BenchmarkAppendSubsetWholeFont(b *testing.B) {
	src, _ := loadFont(b, "DejaVuSans.ttf")
	all := make([]uint16, src.NumGlyphs())
	for i := range all {
		all[i] = uint16(i)
	}
	var s sfnt.Subsetter
	dst, err := s.AppendSubset(nil, src, all)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(dst)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		dst, _ = s.AppendSubset(dst[:0], src, all)
	}
}

func BenchmarkAppendSubsetLine(b *testing.B) {
	src, _ := loadFont(b, "DejaVuSans.ttf")
	var gids []uint16
	for _, r := range "The quick brown fox jumps over the lazy dog 0123456789" {
		gids = append(gids, src.GlyphID(r))
	}
	var s sfnt.Subsetter
	dst, err := s.AppendSubset(nil, src, gids)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(dst)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		dst, _ = s.AppendSubset(dst[:0], src, gids)
	}
}

func BenchmarkChecksum(b *testing.B) {
	_, data := loadFont(b, "DejaVuSans.ttf")
	b.SetBytes(int64(len(data)))
	for range b.N {
		_ = sfnt.Checksum(data)
	}
}
