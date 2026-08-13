package sfnt

import (
	"encoding/binary"
	"errors"
	"slices"
	"strings"
)

var (
	// ErrNoOutlines is returned by [Subsetter.AppendSubset], which subsets by copying glyf records.
	ErrNoOutlines = errors.New("sfnt: font has no glyf outlines (CFF/OTTO fonts are unsupported)")
	// ErrNoHead is returned by [Append]: head is where the whole-file checksum goes.
	ErrNoHead = errors.New("sfnt: cannot assemble a font without a head table")
	// ErrTruncated covers both a truncated file and a table too short for its fields.
	ErrTruncated = errors.New("sfnt: font data truncated")
	// ErrBadTag is returned by [Append]: a directory addresses each table by tag.
	ErrBadTag = errors.New("sfnt: table tag is not four characters, or is repeated")
)

// headSize is the length of a head table. Every field in it is required, so
// there is no shorter valid one.
const headSize = 54

// Table is one table of a font being assembled. Data is not copied until the
// font is written, and is never modified.
type Table struct {
	// Tag is the table's four-character name, e.g. "glyf" or "cvt " — note
	// the trailing space, tags are always four characters.
	Tag  string
	Data []byte
}

// Append appends tables to dst as a font file, sorted in place, with hints,
// checksums and head's adjustment filled in. Errors leave dst untouched.
func Append(dst []byte, tables []Table) ([]byte, error) {
	// Everything is checked before the first write: past that there is no way
	// to report a failure, nor to hand dst back as it came in.
	head := -1
	for i, t := range tables {
		if len(t.Tag) != 4 {
			return dst, ErrBadTag
		}
		if t.Tag == "head" {
			head = i
		}
	}
	if head < 0 {
		return dst, ErrNoHead
	}
	if len(tables[head].Data) < headSize {
		return dst, ErrTruncated
	}
	slices.SortFunc(tables, func(a, b Table) int { return strings.Compare(a.Tag, b.Tag) })
	// Sorted, so a repeat is an immediate neighbour.
	for i := 1; i < len(tables); i++ {
		if tables[i].Tag == tables[i-1].Tag {
			return dst, ErrBadTag
		}
	}

	base, n := len(dst), len(tables)
	dirLen := 12 + 16*n
	size := dirLen
	for _, t := range tables {
		size += (len(t.Data) + 3) &^ 3
	}
	// Grow once, then take the directory's space up front: every offset the
	// directory records is only known as the tables are laid down after it.
	dst = slices.Grow(dst, size)
	dst = dst[:base+dirLen]
	clear(dst[base:])

	putBE32(dst[base:], 0x00010000)
	putBE16(dst[base+4:], uint16(n))
	// The binary-search hints the directory carries: the largest power of two
	// not exceeding n, expressed as a byte range and its log.
	sel := 0
	for 1<<(sel+1) <= n {
		sel++
	}
	putBE16(dst[base+6:], uint16(16<<sel))
	putBE16(dst[base+8:], uint16(sel))
	putBE16(dst[base+10:], uint16(16*n-16<<sel))

	headOff := 0
	var sum uint32 // the whole file's, accumulated as it is written
	for i, t := range tables {
		rec, off := base+12+16*i, len(dst)-base
		copy(dst[rec:], t.Tag)
		putBE32(dst[rec+8:], uint32(off))
		putBE32(dst[rec+12:], uint32(len(t.Data)))
		dst = append(dst, t.Data...)
		for (len(dst)-base)%4 != 0 {
			dst = append(dst, 0)
		}
		if t.Tag == "head" {
			headOff = off
			// checkSumAdjustment describes the file being written, so whatever
			// the source recorded reads zero before anything is summed.
			putBE32(dst[base+off+8:], 0)
		}
		// The checksum is of the bytes as written, not of the caller's copy,
		// which is what makes the zeroing above count.
		c := Checksum(dst[base+off : base+off+len(t.Data)])
		putBE32(dst[rec+4:], c)
		sum += c
	}
	// ISO/IEC 14496-22 head.checkSumAdjustment makes the file sum to the magic
	// constant. dirLen and every table offset are 4-aligned, so sum is the file's.
	font := dst[base:]
	putBE32(font[headOff+8:], 0xB1B0AFBA-(sum+Checksum(font[:dirLen])))
	return dst, nil
}

// Checksum sums b as big-endian uint32s, zero-padding a ragged tail. It is the
// sfnt table checksum, and over a whole font file it comes out to 0xB1B0AFBA.
func Checksum(b []byte) uint32 {
	var sum uint32
	n := len(b) &^ 3
	// One byte-swapping load. Reading the four by hand costs 2.5x over a font.
	for i := 0; i < n; i += 4 {
		sum += binary.BigEndian.Uint32(b[i:])
	}
	if n < len(b) {
		var tail [4]byte
		copy(tail[:], b[n:])
		sum += binary.BigEndian.Uint32(tail[:])
	}
	return sum
}

func putBE16(b []byte, v uint16) { b[0], b[1] = byte(v>>8), byte(v) }

func putBE32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v>>24), byte(v>>16), byte(v>>8), byte(v)
}
