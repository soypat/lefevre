package lefevre

// ShapeConfig configures text shaping. The zero value uses default settings.
type ShapeConfig struct {
	Font     *Font             // Font to shape with (required)
	Size     int32             // Font size in font units; 0 means use UnitsPerEm from font
	Features []FeatureOverride // OpenType feature overrides (e.g., disable ligatures)
}

// Shape shapes text into positioned glyph runs, appending results to dst.
// breaks is the segmentation output from a Breaker for the same text.
// Each run in the output corresponds to a segment of uniform script/direction.
// Returns dst unchanged if Font is nil or invalid.
func (cfg *ShapeConfig) Shape(dst []Run, text string, breaks []Break) []Run {
	if cfg.Font == nil || !cfg.Font.IsValid() || len(text) == 0 {
		return dst
	}
	f := cfg.Font

	// Extract run properties from breaks.
	// Breaks are NOT sorted by position — scan all of them.
	// Position-0 breaks set initial properties;
	// later breaks with direction/script changes define run boundaries.
	type runProps struct {
		dir, parDir Direction
		script      Script
	}
	type boundary struct {
		runePos int
		props   runProps
	}

	cur := runProps{dir: DirectionLTR, parDir: DirectionLTR}
	dirSet := false
	for _, b := range breaks {
		if b.Position != 0 {
			continue
		}
		if b.Flags.HasAll(BreakFlagDirection) {
			cur.dir = b.Direction
			dirSet = true
		}
		if b.Flags.HasAll(BreakFlagParagraphDirection) {
			cur.parDir = b.ParagraphDirection
		}
		if b.Flags.HasAll(BreakFlagScript) {
			cur.script = b.Script
		}
	}
	// If no explicit direction break, inherit from paragraph direction.
	if !dirSet && cur.parDir != DirectionUnknown {
		cur.dir = cur.parDir
	}

	// Collect run boundaries (positions > 0 where direction or script changes).
	var boundaries [16]boundary
	nbounds := 0
	next := cur
	for _, b := range breaks {
		if b.Position == 0 {
			continue
		}
		changed := false
		if b.Flags.HasAll(BreakFlagDirection) {
			next.dir = b.Direction
			changed = true
		}
		if b.Flags.HasAll(BreakFlagScript) {
			next.script = b.Script
			changed = true
		}
		if b.Flags.HasAll(BreakFlagParagraphDirection) {
			next.parDir = b.ParagraphDirection
		}
		if changed {
			if nbounds < len(boundaries) {
				boundaries[nbounds] = boundary{runePos: b.Position, props: next}
			}
			nbounds++
		}
	}

	// Fall back to heap slice if more than 16 boundaries (rare).
	var heapBounds []boundary
	if nbounds > len(boundaries) {
		heapBounds = make([]boundary, 0, nbounds)
		next = cur
		for _, b := range breaks {
			if b.Position == 0 {
				continue
			}
			changed := false
			if b.Flags.HasAll(BreakFlagDirection) {
				next.dir = b.Direction
				changed = true
			}
			if b.Flags.HasAll(BreakFlagScript) {
				next.script = b.Script
				changed = true
			}
			if b.Flags.HasAll(BreakFlagParagraphDirection) {
				next.parDir = b.ParagraphDirection
			}
			if changed {
				heapBounds = append(heapBounds, boundary{runePos: b.Position, props: next})
			}
		}
	}

	getBound := func(i int) boundary {
		if heapBounds != nil {
			return heapBounds[i]
		}
		return boundaries[i]
	}

	// Check if ligatures are disabled via feature overrides.
	ligaEnabled := true
	for _, fo := range cfg.Features {
		if fo.Tag == FeatureTagLiga && fo.Value == 0 {
			ligaEnabled = false
		}
	}

	emitRun := func(glyphs []Glyph, props runProps) Run {
		if ligaEnabled {
			glyphs = f.applyGSUBLigatures(glyphs)
		}
		// Recompute advances after substitution (ligature glyphs have different advances).
		for i := range glyphs {
			glyphs[i].AdvanceX = f.glyphAdvance(glyphs[i].ID)
		}
		return Run{
			Font:               f,
			Script:             props.script,
			Direction:          props.dir,
			ParagraphDirection: props.parDir,
			Glyphs:             glyphs,
		}
	}

	// Iterate text once, building glyphs. Emit a run at each boundary.
	var glyphs []Glyph
	runeIdx := 0
	bi := 0

	for _, r := range text {
		if bi < nbounds && runeIdx == getBound(bi).runePos {
			if len(glyphs) > 0 {
				dst = append(dst, emitRun(glyphs, cur))
				glyphs = nil
			}
			cur = getBound(bi).props
			bi++
		}

		gid := f.GlyphID(r)
		glyphs = append(glyphs, Glyph{
			Codepoint: r,
			ID:        gid,
			UserID:    runeIdx,
		})
		runeIdx++
	}

	if len(glyphs) > 0 {
		dst = append(dst, emitRun(glyphs, cur))
	}

	return dst
}

// ShapeSimple segments and shapes text in one call.
// Equivalent to running a Breaker then calling Shape.
func (cfg *ShapeConfig) ShapeSimple(dst []Run, text string, dir Direction) []Run {
	if cfg.Font == nil || !cfg.Font.IsValid() || len(text) == 0 {
		return dst
	}
	var b Breaker
	b.Direction = dir
	breaks, _ := b.AppendBreak(nil, []byte(text))
	breaks = b.End(breaks)
	return cfg.Shape(dst, text, breaks)
}

// glyphAdvance returns the horizontal advance width for a glyph ID from the hmtx table.
func (f *Font) glyphAdvance(glyphID uint16) int32 {
	hhea := f.tables[tableHhea]
	hmtx := f.tables[tableHmtx]
	if hhea.length < 36 || hmtx.length == 0 {
		return 0
	}
	hheaBase := int(hhea.offset)
	if hheaBase+36 > len(f.data) {
		return 0
	}
	numHMetrics := readU16BE(f.data, hheaBase+34)
	if numHMetrics == 0 {
		return 0
	}

	idx := glyphID
	if idx >= numHMetrics {
		idx = numHMetrics - 1
	}
	off := int(hmtx.offset) + int(idx)*4
	if off+2 > len(f.data) {
		return 0
	}
	return int32(readU16BE(f.data, off))
}
