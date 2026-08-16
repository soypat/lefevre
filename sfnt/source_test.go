package sfnt_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"slices"
	"testing"

	"github.com/soypat/lefevre"
	"github.com/soypat/lefevre/sfnt"
)

// rawFont is a [sfnt.Source] built from nothing but the font's bytes: it is the
// claim rung 4 makes, that a subsetter needs a font's tables and not the font.
// Nothing in this type knows lefevre exists.
type rawFont struct {
	data []byte
}

func (r *rawFont) Table(tag string) []byte {
	if len(r.data) < 12 || len(tag) != 4 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(r.data[4:]))
	for i := range n {
		rec := 12 + i*16
		if rec+16 > len(r.data) || string(r.data[rec:rec+4]) != tag {
			continue
		}
		off := int(binary.BigEndian.Uint32(r.data[rec+8:]))
		length := int(binary.BigEndian.Uint32(r.data[rec+12:]))
		if length == 0 || off+length > len(r.data) {
			return nil
		}
		return r.data[off : off+length]
	}
	return nil
}

func (r *rawFont) NumGlyphs() int {
	maxp := r.Table("maxp")
	if len(maxp) < 6 {
		return 0
	}
	return int(binary.BigEndian.Uint16(maxp[4:]))
}

// GlyphData walks loca in whichever of its two forms head declares.
func (r *rawFont) GlyphData(glyph uint16) []byte {
	head, loca, glyf := r.Table("head"), r.Table("loca"), r.Table("glyf")
	if len(head) < 54 {
		return nil
	}
	var start, end int
	if binary.BigEndian.Uint16(head[50:]) == 0 { // short loca, offsets halved
		if int(glyph)*2+4 > len(loca) {
			return nil
		}
		start = int(binary.BigEndian.Uint16(loca[glyph*2:])) * 2
		end = int(binary.BigEndian.Uint16(loca[glyph*2+2:])) * 2
	} else {
		if int(glyph)*4+8 > len(loca) {
			return nil
		}
		start = int(binary.BigEndian.Uint32(loca[glyph*4:]))
		end = int(binary.BigEndian.Uint32(loca[glyph*4+4:]))
	}
	if start >= end || end > len(glyf) {
		return nil
	}
	return glyf[start:end]
}

// A subset built through the interface and one built through *lefevre.Font are
// the same bytes: Source is the whole of what subsetting reads.
func TestSubsetFromSource(t *testing.T) {
	for _, name := range []string{"DejaVuSans.ttf", "OpenSans.ttf"} {
		t.Run(name, func(t *testing.T) {
			f, data, err := loadRaw(t, name)
			if err != nil {
				t.Skipf("test font not available: %v", err)
			}
			// Ä and Ç are composite, so the closure has work to do.
			var gids []uint16
			for _, r := range "Hello, wörld! ÄÇ" {
				gids = append(gids, f.GlyphID(r))
			}

			var viaSource, viaFont sfnt.Subsetter
			want, err := viaFont.AppendSubset(nil, f, gids)
			if err != nil {
				t.Fatalf("AppendSubset via *lefevre.Font: %v", err)
			}
			got, err := viaSource.AppendSubset(nil, &rawFont{data: data}, gids)
			if err != nil {
				t.Fatalf("AppendSubset via Source: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("subset via Source is %d bytes, via *lefevre.Font %d bytes; want identical",
					len(got), len(want))
			}

			wantClosure := viaFont.AppendClosure(nil, f, gids)
			gotClosure := viaSource.AppendClosure(nil, &rawFont{data: data}, gids)
			if !slices.Equal(gotClosure, wantClosure) {
				t.Errorf("AppendClosure via Source = %v, via *lefevre.Font = %v", gotClosure, wantClosure)
			}
		})
	}
}

func loadRaw(t *testing.T, name string) (*lefevre.Font, []byte, error) {
	t.Helper()
	data, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		return nil, nil, err
	}
	f, err := lefevre.FontFromMemory(data, 0)
	if err != nil {
		t.Fatalf("FontFromMemory(%s): %v", name, err)
	}
	return f, data, nil
}

// Subsetting reads composition out of the glyph record; lefevre reads it out of
// the same record, independently. Over every glyph of two real fonts the two
// agree, which is what makes AppendGlyphComponents a method Source need not ask
// for.
func TestClosureMatchesFontComponents(t *testing.T) {
	for _, name := range []string{"DejaVuSans.ttf", "OpenSans.ttf"} {
		t.Run(name, func(t *testing.T) {
			f, _ := loadFont(t, name)
			var s sfnt.Subsetter
			var want, stack []uint16
			seen := make([]bool, f.NumGlyphs())
			composites := 0
			for gid := range f.NumGlyphs() {
				// The closure of gid, walked with the font's own component call.
				clear(seen)
				want, stack = want[:0], append(stack[:0], uint16(gid))
				seen[gid] = true
				for len(stack) > 0 {
					g := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					n := len(stack)
					stack = f.AppendGlyphComponents(stack, g)
					w := n
					for _, comp := range stack[n:] {
						if int(comp) < len(seen) && !seen[comp] {
							seen[comp] = true
							stack[w] = comp
							w++
						}
					}
					stack = stack[:w]
				}
				for id, kept := range seen {
					if kept {
						want = append(want, uint16(id))
					}
				}
				if len(want) > 1 {
					composites++
				}
				if got := s.AppendClosure(nil, f, []uint16{uint16(gid)}); !slices.Equal(got, want) {
					t.Fatalf("AppendClosure(glyph %d) = %v, want %v", gid, got, want)
				}
			}
			if composites == 0 {
				t.Error("no composite glyph in the font: the walk was never exercised")
			}
		})
	}
}
