package raster

import (
	"slices"
	"testing"

	"github.com/soypat/lefevre"
)

// sortedPlacements packs glyphs then sorts by GlyphID for lookup.
func sortedPlacements(t *testing.T, cfg PackConfig, glyphs []uint16) []PackedGlyph {
	t.Helper()
	placements, err := cfg.Pack(nil, glyphs, 256, 256)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	slices.SortFunc(placements, func(a, b PackedGlyph) int {
		return int(a.GlyphID) - int(b.GlyphID)
	})
	return placements
}

func TestBuildQuads_CountMatchesNonEmptyGlyphs(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	gA := f.GlyphID('A')
	gB := f.GlyphID('B')
	gSpace := f.GlyphID(' ')
	placements := sortedPlacements(t, cfg, []uint16{gA, gB, gSpace})

	runs := []lefevre.Run{{
		Font: f,
		Glyphs: []lefevre.Glyph{
			{ID: gA, AdvanceX: 500},
			{ID: gSpace, AdvanceX: 300},
			{ID: gB, AdvanceX: 500},
		},
	}}

	quads := BuildQuads(nil, runs, placements, 0, 0, scale)
	if len(quads) != 2 {
		t.Fatalf("quad count = %d, want 2 (space has zero-size placement)", len(quads))
	}
}

func TestBuildQuads_SrcRectMatchesPlacement(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	gA := f.GlyphID('A')
	placements := sortedPlacements(t, cfg, []uint16{gA})
	pa, ok := FindPackedGlyph(placements, gA)
	if !ok {
		t.Fatal("packed 'A' missing")
	}

	runs := []lefevre.Run{{Font: f, Glyphs: []lefevre.Glyph{{ID: gA, AdvanceX: 500}}}}
	quads := BuildQuads(nil, runs, placements, 0, 0, scale)
	if len(quads) != 1 {
		t.Fatalf("got %d quads, want 1", len(quads))
	}
	q := quads[0]
	if q.SrcX != pa.X || q.SrcY != pa.Y || q.SrcW != pa.W || q.SrcH != pa.H {
		t.Errorf("src rect = (%d,%d,%d,%d), want placement (%d,%d,%d,%d)",
			q.SrcX, q.SrcY, q.SrcW, q.SrcH, pa.X, pa.Y, pa.W, pa.H)
	}
	if q.DstW != pa.W || q.DstH != pa.H {
		t.Errorf("dst size = %dx%d, want %dx%d", q.DstW, q.DstH, pa.W, pa.H)
	}
}

func TestBuildQuads_DstAppliesOriginAndPenOffsets(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	gA := f.GlyphID('A')
	placements := sortedPlacements(t, cfg, []uint16{gA})
	pa, _ := FindPackedGlyph(placements, gA)

	runs := []lefevre.Run{{Font: f, Glyphs: []lefevre.Glyph{{ID: gA, AdvanceX: 500}}}}
	const ox, oy = 100, 200
	quads := BuildQuads(nil, runs, placements, ox, oy, scale)
	q := quads[0]

	wantX := ox + pa.Xoff
	wantY := oy + pa.Yoff
	if q.DstX != wantX || q.DstY != wantY {
		t.Errorf("dst pos = (%d,%d), want (%d,%d)", q.DstX, q.DstY, wantX, wantY)
	}
}

func TestBuildQuads_PenAdvancesBetweenGlyphs(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	gA := f.GlyphID('A')
	gB := f.GlyphID('B')
	placements := sortedPlacements(t, cfg, []uint16{gA, gB})
	pa, _ := FindPackedGlyph(placements, gA)
	pb, _ := FindPackedGlyph(placements, gB)

	const adv = 1000
	runs := []lefevre.Run{{Font: f, Glyphs: []lefevre.Glyph{
		{ID: gA, AdvanceX: adv},
		{ID: gB, AdvanceX: adv},
	}}}
	quads := BuildQuads(nil, runs, placements, 0, 0, scale)
	if len(quads) != 2 {
		t.Fatalf("got %d quads, want 2", len(quads))
	}
	gapX := quads[1].DstX - quads[0].DstX
	// Gap is advance(A) + Xoff(B) - Xoff(A) in pixels.
	want := int(float32(adv)*scale) + pb.Xoff - pa.Xoff
	if gapX != want {
		t.Errorf("pen gap = %d, want %d", gapX, want)
	}
}

func TestBuildQuads_AppendsToExisting(t *testing.T) {
	f := loadTestFont(t)
	info := f.Info()
	scale := float32(32) / float32(info.UnitsPerEm)
	cfg := PackConfig{Font: f, Scale: scale, Padding: 1}

	gA := f.GlyphID('A')
	placements := sortedPlacements(t, cfg, []uint16{gA})
	runs := []lefevre.Run{{Font: f, Glyphs: []lefevre.Glyph{{ID: gA, AdvanceX: 500}}}}

	sentinel := DrawQuad{DstX: -1, DstY: -2, DstW: -3, DstH: -4}
	quads := BuildQuads([]DrawQuad{sentinel}, runs, placements, 0, 0, scale)
	if len(quads) < 2 {
		t.Fatalf("got %d quads, want >= 2", len(quads))
	}
	if quads[0] != sentinel {
		t.Errorf("sentinel overwritten: got %+v", quads[0])
	}
}