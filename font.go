package lefevre

import (
	"fmt"
	"unicode/utf16"
)

// OpenType table tag identifiers.
const (
	tableHead = iota
	tableCmap
	tableGdef
	tableGsub
	tableGpos
	tableHhea
	tableVhea
	tableHmtx
	tableVmtx
	tableMaxp
	tableOS2
	tableName
	tableCount
)

// tableEntry holds the offset and length of an OpenType table within font data.
type tableEntry struct {
	offset uint32
	length uint32
}

// Big-endian reading helpers.

func readU16BE(data []byte, off int) uint16 {
	return uint16(data[off])<<8 | uint16(data[off+1])
}

func readU32BE(data []byte, off int) uint32 {
	return uint32(data[off])<<24 | uint32(data[off+1])<<16 | uint32(data[off+2])<<8 | uint32(data[off+3])
}

func readS16BE(data []byte, off int) int16 {
	return int16(readU16BE(data, off))
}

// OpenType magic bytes.
const (
	magicTTF  = 0x00010000
	magicOTTO = 0x4F54544F // 'OTTO'
	magicTTC  = 0x74746366 // 'ttcf'
)

// fontCount returns the number of fonts in the data.
func fontCount(data []byte) int {
	if len(data) < 4 {
		return 0
	}
	magic := readU32BE(data, 0)
	switch magic {
	case magicTTF, magicOTTO:
		return 1
	case magicTTC:
		if len(data) < 12 {
			return 0
		}
		return int(readU32BE(data, 8))
	default:
		return 0
	}
}

// Known OpenType table tags (big-endian fourcc).
var tableTagMap = map[uint32]int{
	0x68656164: tableHead, // 'head'
	0x636D6170: tableCmap, // 'cmap'
	0x47444546: tableGdef, // 'GDEF'
	0x47535542: tableGsub, // 'GSUB'
	0x47504F53: tableGpos, // 'GPOS'
	0x68686561: tableHhea, // 'hhea'
	0x76686561: tableVhea, // 'vhea'
	0x686D7478: tableHmtx, // 'hmtx'
	0x766D7478: tableVmtx, // 'vmtx'
	0x6D617870: tableMaxp, // 'maxp'
	0x4F532F32: tableOS2,  // 'OS/2'
	0x6E616D65: tableName, // 'name'
}

// fontFromMemory parses a font from raw bytes.
func fontFromMemory(data []byte, fontIndex int) (*Font, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("kbts: font data too short (%d bytes)", len(data))
	}

	magic := readU32BE(data, 0)
	var dirOffset int
	switch magic {
	case magicTTF, magicOTTO:
		if fontIndex != 0 {
			return nil, fmt.Errorf("kbts: font index %d out of range (single font file)", fontIndex)
		}
		dirOffset = 0
	case magicTTC:
		count := int(readU32BE(data, 8))
		if fontIndex < 0 || fontIndex >= count {
			return nil, fmt.Errorf("kbts: font index %d out of range (collection has %d fonts)", fontIndex, count)
		}
		offTableStart := 12 + fontIndex*4
		if offTableStart+4 > len(data) {
			return nil, fmt.Errorf("kbts: TTC offset table truncated")
		}
		dirOffset = int(readU32BE(data, offTableStart))
	default:
		return nil, fmt.Errorf("kbts: unrecognized font format (magic 0x%08X)", magic)
	}

	// Parse table directory.
	if dirOffset+12 > len(data) {
		return nil, fmt.Errorf("kbts: table directory truncated")
	}
	numTables := int(readU16BE(data, dirOffset+4))
	recordsStart := dirOffset + 12
	if recordsStart+numTables*16 > len(data) {
		return nil, fmt.Errorf("kbts: table records truncated")
	}

	f := &Font{data: data}
	for i := 0; i < numTables; i++ {
		recOff := recordsStart + i*16
		tag := readU32BE(data, recOff)
		// skip checksum at recOff+4
		tblOffset := readU32BE(data, recOff+8)
		tblLength := readU32BE(data, recOff+12)
		if uint64(tblOffset)+uint64(tblLength) > uint64(len(data)) {
			continue // skip tables that extend beyond file
		}
		if id, ok := tableTagMap[tag]; ok {
			f.tables[id] = tableEntry{offset: tblOffset, length: tblLength}
		}
	}

	// Select cmap subtable.
	f.selectCmap()

	return f, nil
}

// selectCmap finds the best cmap subtable and stores it on the Font.
func (f *Font) selectCmap() {
	te := f.tables[tableCmap]
	if te.length == 0 {
		return
	}
	base := int(te.offset)
	if base+4 > len(f.data) {
		return
	}
	numSubtables := int(readU16BE(f.data, base+2))
	if base+4+numSubtables*8 > len(f.data) {
		return
	}

	bestFormat := uint16(0)
	bestOffset := 0
	for i := 0; i < numSubtables; i++ {
		recOff := base + 4 + i*8
		platformID := readU16BE(f.data, recOff)
		_ = readU16BE(f.data, recOff+2) // encodingID
		subtableOffset := int(readU32BE(f.data, recOff+4))
		absOffset := base + subtableOffset
		if absOffset+2 > len(f.data) {
			continue
		}
		format := readU16BE(f.data, absOffset)
		// Rank: format 12 > 4 > others. Prefer Windows platform (3).
		better := false
		switch {
		case format == 12 && bestFormat != 12:
			better = true
		case format == 12 && bestFormat == 12 && platformID == 3:
			better = true
		case format == 4 && bestFormat < 4:
			better = true
		case format == 4 && bestFormat == 4 && platformID == 3:
			better = true
		}
		if better {
			bestFormat = format
			bestOffset = absOffset
		}
	}
	if bestOffset == 0 {
		return
	}
	f.cmapFormat = bestFormat
	f.cmapOffset = uint32(bestOffset)
}

// glyphID looks up a codepoint in the selected cmap subtable.
func (f *Font) glyphID(cp rune) uint16 {
	if f == nil || f.cmapOffset == 0 {
		return 0
	}
	switch f.cmapFormat {
	case 4:
		return f.cmapFormat4Lookup(cp)
	case 12:
		return f.cmapFormat12Lookup(cp)
	default:
		return 0
	}
}

// cmapFormat4Lookup implements cmap format 4 (segment mapping to delta values).
func (f *Font) cmapFormat4Lookup(cp rune) uint16 {
	if cp > 0xFFFF {
		return 0 // format 4 only handles BMP
	}
	c := uint16(cp)
	off := int(f.cmapOffset)
	if off+14 > len(f.data) {
		return 0
	}
	segCountX2 := int(readU16BE(f.data, off+6))
	segCount := segCountX2 / 2
	if segCount == 0 {
		return 0
	}
	// Array layout after format(2)+length(2)+language(2)+segCountX2(2)+searchRange(2)+entrySelector(2)+rangeShift(2) = 14 bytes
	endCodeOff := off + 14
	startCodeOff := endCodeOff + segCountX2 + 2 // +2 for reservedPad
	idDeltaOff := startCodeOff + segCountX2
	idRangeOffsetOff := idDeltaOff + segCountX2

	needed := idRangeOffsetOff + segCountX2
	if needed > len(f.data) {
		return 0
	}

	// Binary search for segment.
	lo, hi := 0, segCount-1
	for lo <= hi {
		mid := (lo + hi) / 2
		endCode := readU16BE(f.data, endCodeOff+mid*2)
		if c > endCode {
			lo = mid + 1
			continue
		}
		startCode := readU16BE(f.data, startCodeOff+mid*2)
		if c < startCode {
			hi = mid - 1
			continue
		}
		// Found segment.
		idRangeOffset := readU16BE(f.data, idRangeOffsetOff+mid*2)
		if idRangeOffset == 0 {
			idDelta := readU16BE(f.data, idDeltaOff+mid*2)
			return c + idDelta
		}
		// idRangeOffset is relative to its own position in the array.
		glyphIndexOff := idRangeOffsetOff + mid*2 + int(idRangeOffset) + int(c-startCode)*2
		if glyphIndexOff+2 > len(f.data) {
			return 0
		}
		glyphID := readU16BE(f.data, glyphIndexOff)
		if glyphID == 0 {
			return 0
		}
		idDelta := readU16BE(f.data, idDeltaOff+mid*2)
		return glyphID + idDelta
	}
	return 0
}

// cmapFormat12Lookup implements cmap format 12 (segmented coverage for 32-bit).
func (f *Font) cmapFormat12Lookup(cp rune) uint16 {
	off := int(f.cmapOffset)
	if off+16 > len(f.data) {
		return 0
	}
	// Format 12: u16 format, u16 reserved, u32 length, u32 language, u32 nGroups
	nGroups := int(readU32BE(f.data, off+12))
	groupsOff := off + 16
	if groupsOff+nGroups*12 > len(f.data) {
		return 0
	}

	c := uint32(cp)
	lo, hi := 0, nGroups-1
	for lo <= hi {
		mid := (lo + hi) / 2
		gOff := groupsOff + mid*12
		startCharCode := readU32BE(f.data, gOff)
		endCharCode := readU32BE(f.data, gOff+4)
		if c < startCharCode {
			hi = mid - 1
		} else if c > endCharCode {
			lo = mid + 1
		} else {
			startGlyphID := readU32BE(f.data, gOff+8)
			return uint16(startGlyphID + (c - startCharCode))
		}
	}
	return 0
}

// fontInfo extracts metadata from the parsed font tables.
func (f *Font) fontInfo() FontInfo {
	if f == nil || f.data == nil {
		return FontInfo{}
	}
	var info FontInfo
	f.readNameTable(&info)
	f.readHeadTable(&info)
	f.readOS2Table(&info)
	f.readHheaTable(&info)
	return info
}

func (f *Font) readNameTable(info *FontInfo) {
	te := f.tables[tableName]
	if te.length == 0 {
		return
	}
	base := int(te.offset)
	end := base + int(te.length)
	if end > len(f.data) || base+6 > len(f.data) {
		return
	}
	numRecords := int(readU16BE(f.data, base+2))
	stringStorageOff := base + int(readU16BE(f.data, base+4))
	recordsOff := base + 6

	if recordsOff+numRecords*12 > len(f.data) {
		return
	}

	for i := 0; i < numRecords; i++ {
		rOff := recordsOff + i*12
		platformID := readU16BE(f.data, rOff)
		encodingID := readU16BE(f.data, rOff+2)
		languageID := readU16BE(f.data, rOff+4)
		nameID := readU16BE(f.data, rOff+6)
		strLen := int(readU16BE(f.data, rOff+8))
		strOff := stringStorageOff + int(readU16BE(f.data, rOff+10))

		if languageID != 0 {
			continue
		}
		if strOff+strLen > len(f.data) {
			continue
		}

		raw := f.data[strOff : strOff+strLen]
		var s string
		if platformID == 3 || (platformID == 0 && encodingID > 0) {
			// UTF-16BE
			s = decodeUTF16BE(raw)
		} else {
			// Latin-1 / ASCII
			s = string(raw)
		}

		dst := nameIDToField(info, nameID)
		if dst != nil && *dst == "" {
			*dst = s
		}
	}

	// Fallback: typographic family/subfamily to regular family/subfamily.
	if info.TypographicFamily == "" {
		info.TypographicFamily = info.Family
	}
	if info.TypographicSubfamily == "" {
		info.TypographicSubfamily = info.Subfamily
	}
}

func nameIDToField(info *FontInfo, nameID uint16) *string {
	switch nameID {
	case 0:
		return &info.Copyright
	case 1:
		return &info.Family
	case 2:
		return &info.Subfamily
	case 4:
		return &info.FullName
	case 5:
		return &info.Version
	case 6:
		return &info.PostScriptName
	case 8:
		return &info.Manufacturer
	case 16:
		return &info.TypographicFamily
	case 17:
		return &info.TypographicSubfamily
	default:
		return nil
	}
}

func decodeUTF16BE(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		u16s[i] = uint16(b[i*2])<<8 | uint16(b[i*2+1])
	}
	return string(utf16.Decode(u16s))
}

func (f *Font) readHeadTable(info *FontInfo) {
	te := f.tables[tableHead]
	if te.length < 54 {
		return
	}
	base := int(te.offset)
	if base+54 > len(f.data) {
		return
	}
	info.UnitsPerEm = readU16BE(f.data, base+18)
	info.XMin = readS16BE(f.data, base+36)
	info.YMin = readS16BE(f.data, base+38)
	info.XMax = readS16BE(f.data, base+40)
	info.YMax = readS16BE(f.data, base+42)
}

func (f *Font) readOS2Table(info *FontInfo) {
	te := f.tables[tableOS2]
	if te.length < 68 {
		return
	}
	base := int(te.offset)
	if base+int(te.length) > len(f.data) {
		return
	}

	weightClass := readU16BE(f.data, base+4)
	info.Weight = weightClassToFontWeight(weightClass)

	widthClass := readU16BE(f.data, base+6)
	info.Width = widthClassToFontWidth(widthClass)

	if te.length >= 64 {
		selection := readU16BE(f.data, base+62)
		if selection&(1<<0) != 0 { // italic
			info.StyleFlags |= FontStyleItalic
		}
		if selection&(1<<5) != 0 { // bold
			info.StyleFlags |= FontStyleBold
		}
		if selection&(1<<6) != 0 { // regular
			info.StyleFlags |= FontStyleRegular
		}
	}

	if te.length >= 74 {
		info.Ascent = readS16BE(f.data, base+68)
		info.Descent = readS16BE(f.data, base+70)
		info.LineGap = readS16BE(f.data, base+72)
	}

	if te.length >= 90 {
		info.CapHeight = readS16BE(f.data, base+88)
	}
}

func (f *Font) readHheaTable(info *FontInfo) {
	te := f.tables[tableHhea]
	if te.length < 10 {
		return
	}
	base := int(te.offset)
	if base+10 > len(f.data) {
		return
	}
	// Only use hhea as fallback if OS/2 didn't set these.
	if info.Ascent == 0 {
		info.Ascent = readS16BE(f.data, base+4)
	}
	if info.Descent == 0 {
		info.Descent = readS16BE(f.data, base+6)
	}
	if info.LineGap == 0 {
		info.LineGap = readS16BE(f.data, base+8)
	}
}

func weightClassToFontWeight(wc uint16) FontWeight {
	switch {
	case wc <= 100:
		return FontWeightThin
	case wc <= 200:
		return FontWeightExtraLight
	case wc <= 300:
		return FontWeightLight
	case wc <= 400:
		return FontWeightNormal
	case wc <= 500:
		return FontWeightMedium
	case wc <= 600:
		return FontWeightSemiBold
	case wc <= 700:
		return FontWeightBold
	case wc <= 800:
		return FontWeightExtraBold
	default:
		return FontWeightBlack
	}
}

func widthClassToFontWidth(wc uint16) FontWidth {
	switch wc {
	case 1:
		return FontWidthUltraCondensed
	case 2:
		return FontWidthExtraCondensed
	case 3:
		return FontWidthCondensed
	case 4:
		return FontWidthSemiCondensed
	case 5:
		return FontWidthNormal
	case 6:
		return FontWidthSemiExpanded
	case 7:
		return FontWidthExpanded
	case 8:
		return FontWidthExtraExpanded
	case 9:
		return FontWidthUltraExpanded
	default:
		return FontWidthUnknown
	}
}

// findGSUBFeatureIndices returns all indices in the GSUB FeatureList matching a feature tag.
func (f *Font) findGSUBFeatureIndices(dst []int, tag FeatureTag) []int {
	te := f.tables[tableGsub]
	if te.length == 0 {
		return dst
	}
	base := int(te.offset)
	if base+10 > len(f.data) {
		return dst
	}
	featureListOff := base + int(readU16BE(f.data, base+6))
	if featureListOff+2 > len(f.data) {
		return dst
	}
	featureCount := int(readU16BE(f.data, featureListOff))
	if featureListOff+2+featureCount*6 > len(f.data) {
		return dst
	}
	tagU32 := uint32(tag)
	for i := 0; i < featureCount; i++ {
		recOff := featureListOff + 2 + i*6
		ftag := uint32(f.data[recOff]) | uint32(f.data[recOff+1])<<8 | uint32(f.data[recOff+2])<<16 | uint32(f.data[recOff+3])<<24
		if ftag == tagU32 {
			dst = append(dst, i)
		}
	}
	return dst
}

// findGSUBFeatureIndex searches the GSUB FeatureList for the first matching feature tag.
// Returns the feature index, or -1 if not found.
func (f *Font) findGSUBFeatureIndex(tag FeatureTag) int {
	var buf [4]int
	indices := f.findGSUBFeatureIndices(buf[:0], tag)
	if len(indices) == 0 {
		return -1
	}
	return indices[0]
}

// gsubFeatureLookups returns the lookup indices for a GSUB feature by feature index.
func (f *Font) gsubFeatureLookups(featureIndex int) []uint16 {
	te := f.tables[tableGsub]
	if te.length == 0 {
		return nil
	}
	base := int(te.offset)
	if base+10 > len(f.data) {
		return nil
	}
	featureListOff := base + int(readU16BE(f.data, base+6))
	if featureListOff+2 > len(f.data) {
		return nil
	}
	featureCount := int(readU16BE(f.data, featureListOff))
	if featureIndex < 0 || featureIndex >= featureCount {
		return nil
	}
	recOff := featureListOff + 2 + featureIndex*6
	if recOff+6 > len(f.data) {
		return nil
	}
	featureOffset := featureListOff + int(readU16BE(f.data, recOff+4))
	if featureOffset+4 > len(f.data) {
		return nil
	}
	// Feature table: u16 featureParams, u16 lookupCount, u16[] lookupListIndices
	lookupCount := int(readU16BE(f.data, featureOffset+2))
	if featureOffset+4+lookupCount*2 > len(f.data) {
		return nil
	}
	lookups := make([]uint16, lookupCount)
	for i := range lookups {
		lookups[i] = readU16BE(f.data, featureOffset+4+i*2)
	}
	return lookups
}

// applyGSUBLigatures applies GSUB ligature substitutions (lookup type 4) to glyphs.
// This is a minimal implementation covering the 'liga' feature path.
func (f *Font) applyGSUBLigatures(glyphs []Glyph) []Glyph {
	te := f.tables[tableGsub]
	if te.length == 0 {
		return glyphs
	}
	var idxBuf [4]int
	featureIndices := f.findGSUBFeatureIndices(idxBuf[:0], FeatureTagLiga)
	if len(featureIndices) == 0 {
		return glyphs
	}

	base := int(te.offset)
	if base+10 > len(f.data) {
		return glyphs
	}
	lookupListOff := base + int(readU16BE(f.data, base+8))
	if lookupListOff+2 > len(f.data) {
		return glyphs
	}
	lookupCount := int(readU16BE(f.data, lookupListOff))

	for _, fi := range featureIndices {
		lookupIndices := f.gsubFeatureLookups(fi)
		for _, li := range lookupIndices {
			if int(li) >= lookupCount {
				continue
			}
			lookupOff := lookupListOff + int(readU16BE(f.data, lookupListOff+2+int(li)*2))
			glyphs = f.applyGSUBLookup(glyphs, lookupOff)
		}
	}
	return glyphs
}

// applyGSUBLookup applies a single GSUB lookup to glyphs.
func (f *Font) applyGSUBLookup(glyphs []Glyph, lookupOff int) []Glyph {
	if lookupOff+6 > len(f.data) {
		return glyphs
	}
	lookupType := readU16BE(f.data, lookupOff)
	// lookupFlag at lookupOff+2
	subtableCount := int(readU16BE(f.data, lookupOff+4))
	if lookupOff+6+subtableCount*2 > len(f.data) {
		return glyphs
	}

	if lookupType != 4 {
		return glyphs // only handle ligature substitution (type 4)
	}

	for si := 0; si < subtableCount; si++ {
		subtableOff := lookupOff + int(readU16BE(f.data, lookupOff+6+si*2))
		glyphs = f.applyLigatureSubtable(glyphs, subtableOff)
	}
	return glyphs
}

// applyLigatureSubtable applies a GSUB type 4 (ligature) subtable.
func (f *Font) applyLigatureSubtable(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return glyphs
	}
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	ligSetCount := int(readU16BE(f.data, subtableOff+4))
	if subtableOff+6+ligSetCount*2 > len(f.data) {
		return glyphs
	}

	// Process glyphs left to right, attempting ligature match at each position.
	i := 0
	for i < len(glyphs) {
		covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
		if covIdx < 0 || covIdx >= ligSetCount {
			i++
			continue
		}
		ligSetOff := subtableOff + int(readU16BE(f.data, subtableOff+6+covIdx*2))
		matched, newGlyph := f.matchLigatureSet(glyphs, i, ligSetOff)
		if matched > 0 {
			// Replace matched glyphs with ligature glyph.
			glyphs[i] = newGlyph
			glyphs[i].Flags |= GlyphFlagLigature
			glyphs = append(glyphs[:i+1], glyphs[i+1+matched:]...)
			i++
		} else {
			i++
		}
	}
	return glyphs
}

// coverageIndex returns the coverage index for a glyph ID, or -1 if not covered.
func (f *Font) coverageIndex(coverageOff int, glyphID uint16) int {
	if coverageOff+4 > len(f.data) {
		return -1
	}
	format := readU16BE(f.data, coverageOff)
	switch format {
	case 1: // Individual glyph IDs
		count := int(readU16BE(f.data, coverageOff+2))
		if coverageOff+4+count*2 > len(f.data) {
			return -1
		}
		// Binary search.
		lo, hi := 0, count-1
		for lo <= hi {
			mid := (lo + hi) / 2
			gid := readU16BE(f.data, coverageOff+4+mid*2)
			if glyphID < gid {
				hi = mid - 1
			} else if glyphID > gid {
				lo = mid + 1
			} else {
				return mid
			}
		}
		return -1
	case 2: // Range records
		count := int(readU16BE(f.data, coverageOff+2))
		if coverageOff+4+count*6 > len(f.data) {
			return -1
		}
		lo, hi := 0, count-1
		for lo <= hi {
			mid := (lo + hi) / 2
			recOff := coverageOff + 4 + mid*6
			startGlyph := readU16BE(f.data, recOff)
			endGlyph := readU16BE(f.data, recOff+2)
			if glyphID < startGlyph {
				hi = mid - 1
			} else if glyphID > endGlyph {
				lo = mid + 1
			} else {
				startCovIdx := readU16BE(f.data, recOff+4)
				return int(startCovIdx) + int(glyphID-startGlyph)
			}
		}
		return -1
	}
	return -1
}

// matchLigatureSet tries to match a ligature from a LigatureSet at glyphs[pos].
// Returns (number of additional glyphs consumed, ligature glyph) on match,
// or (0, Glyph{}) if no match.
func (f *Font) matchLigatureSet(glyphs []Glyph, pos int, ligSetOff int) (int, Glyph) {
	if ligSetOff+2 > len(f.data) {
		return 0, Glyph{}
	}
	ligCount := int(readU16BE(f.data, ligSetOff))
	if ligSetOff+2+ligCount*2 > len(f.data) {
		return 0, Glyph{}
	}

	for li := 0; li < ligCount; li++ {
		ligOff := ligSetOff + int(readU16BE(f.data, ligSetOff+2+li*2))
		if ligOff+4 > len(f.data) {
			continue
		}
		ligGlyphID := readU16BE(f.data, ligOff)
		compCount := int(readU16BE(f.data, ligOff+2))
		if compCount < 2 {
			continue // must have at least 2 components (including first glyph)
		}
		numExtra := compCount - 1 // components after the first
		if ligOff+4+numExtra*2 > len(f.data) {
			continue
		}
		if pos+1+numExtra > len(glyphs) {
			continue // not enough glyphs remaining
		}
		// Check component glyphs match.
		match := true
		for ci := 0; ci < numExtra; ci++ {
			compGlyphID := readU16BE(f.data, ligOff+4+ci*2)
			if glyphs[pos+1+ci].ID != compGlyphID {
				match = false
				break
			}
		}
		if match {
			g := glyphs[pos]
			g.ID = ligGlyphID
			return numExtra, g
		}
	}
	return 0, Glyph{}
}
