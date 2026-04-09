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

// glyphClass represents an OpenType glyph class from the GDEF table.
type glyphClass uint16

const (
	glyphClassZero      glyphClass = 0 // not in class definition (unclassified)
	glyphClassBase      glyphClass = 1
	glyphClassLigature  glyphClass = 2
	glyphClassMark      glyphClass = 3
	glyphClassComponent glyphClass = 4
)

// glyphClassDef returns the glyph class for a glyph ID from the GDEF GlyphClassDef table.
// Returns glyphClassZero if the font has no GDEF table or the glyph is not classified.
func (f *Font) glyphClassDef(glyphID uint16) glyphClass {
	te := f.tables[tableGdef]
	if te.length < 8 {
		return glyphClassZero
	}
	base := int(te.offset)
	if base+8 > len(f.data) {
		return glyphClassZero
	}
	classDefOff := int(readU16BE(f.data, base+4))
	if classDefOff == 0 {
		return glyphClassZero
	}
	return glyphClass(classDefLookup(f.data, base+classDefOff, glyphID))
}

// markAttachmentClass returns the mark attachment class for a glyph ID from the GDEF table.
// Only meaningful when glyphClassDef returns glyphClassMark.
// Returns 0 if not available.
func (f *Font) markAttachmentClass(glyphID uint16) uint16 {
	te := f.tables[tableGdef]
	if te.length < 12 {
		return 0
	}
	base := int(te.offset)
	if base+12 > len(f.data) {
		return 0
	}
	markAttachOff := int(readU16BE(f.data, base+10))
	if markAttachOff == 0 {
		return 0
	}
	return classDefLookup(f.data, base+markAttachOff, glyphID)
}

// classDefLookup looks up a glyph ID in an OpenType ClassDef table (format 1 or 2).
// Returns the class value, or 0 if the glyph is not found.
// absOff is the absolute offset of the ClassDef table in data.
func classDefLookup(data []byte, absOff int, glyphID uint16) uint16 {
	if absOff+4 > len(data) {
		return 0
	}
	format := readU16BE(data, absOff)
	switch format {
	case 1:
		return classDefFormat1Lookup(data, absOff, glyphID)
	case 2:
		return classDefFormat2Lookup(data, absOff, glyphID)
	default:
		return 0
	}
}

// classDefFormat1Lookup implements ClassDef format 1 (range of glyph IDs).
// Layout: u16 format=1, u16 startGlyphID, u16 glyphCount, u16[glyphCount] classValues
func classDefFormat1Lookup(data []byte, off int, glyphID uint16) uint16 {
	if off+6 > len(data) {
		return 0
	}
	startGlyph := readU16BE(data, off+2)
	glyphCount := int(readU16BE(data, off+4))
	if glyphID < startGlyph {
		return 0
	}
	idx := int(glyphID - startGlyph)
	if idx >= glyphCount {
		return 0
	}
	arrOff := off + 6 + idx*2
	if arrOff+2 > len(data) {
		return 0
	}
	return readU16BE(data, arrOff)
}

// classDefFormat2Lookup implements ClassDef format 2 (ranges with class values).
// Layout: u16 format=2, u16 rangeCount, {u16 startGlyph, u16 endGlyph, u16 class}[rangeCount]
func classDefFormat2Lookup(data []byte, off int, glyphID uint16) uint16 {
	if off+4 > len(data) {
		return 0
	}
	rangeCount := int(readU16BE(data, off+2))
	if off+4+rangeCount*6 > len(data) {
		return 0
	}
	// Binary search.
	lo, hi := 0, rangeCount-1
	for lo <= hi {
		mid := (lo + hi) / 2
		recOff := off + 4 + mid*6
		startGlyph := readU16BE(data, recOff)
		endGlyph := readU16BE(data, recOff+2)
		if glyphID < startGlyph {
			hi = mid - 1
		} else if glyphID > endGlyph {
			lo = mid + 1
		} else {
			return readU16BE(data, recOff+4)
		}
	}
	return 0
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

// defaultGSUBFeatures is the standard set of GSUB features applied for the Default shaper.
// Order matters: ccmp and locl first (composition/decomposition), then ligatures and contextual alternates.
var defaultGSUBFeatures = [...]FeatureTag{
	FeatureTagLocl, // Localized forms
	FeatureTagCcmp, // Glyph composition/decomposition
	FeatureTagRlig, // Required ligatures
	FeatureTagRclt, // Required contextual alternates
	FeatureTagCalt, // Contextual alternates
	FeatureTagLiga, // Standard ligatures
}

// arabicGSUBFeatures is the feature set for the Arabic shaper.
// Applied after joining form assignment: ccmp, locl, then joining features, then ligatures.
var arabicGSUBFeatures = [...]FeatureTag{
	FeatureTagCcmp, // Glyph composition/decomposition
	FeatureTagLocl, // Localized forms
}

// arabicJoiningFeatures maps joiningFeature values to their GSUB feature tags.
// Applied in order: isol, fina, fin2, fin3, medi, med2, init.
var arabicJoiningFeatures = [...]struct {
	feature joiningFeature
	tag     FeatureTag
}{
	{joiningFeatureIsol, FeatureTagIsol},
	{joiningFeatureFina, FeatureTagFina},
	{joiningFeatureFin2, FeatureTagFin2},
	{joiningFeatureFin3, FeatureTagFin3},
	{joiningFeatureMedi, FeatureTagMedi},
	{joiningFeatureMed2, FeatureTagMed2},
	{joiningFeatureInit, FeatureTagInit},
}

// arabicPostJoiningFeatures are applied after joining features.
var arabicPostJoiningFeatures = [...]FeatureTag{
	FeatureTagRlig, // Required ligatures
	FeatureTagRclt, // Required contextual alternates
	FeatureTagCalt, // Contextual alternates
	FeatureTagLiga, // Standard ligatures
}

// joiningFeatureToGlyphFlag converts a joiningFeature to the corresponding GlyphFlags bit.
func joiningFeatureToGlyphFlag(jf joiningFeature) GlyphFlags {
	if jf == joiningFeatureNone {
		return 0
	}
	return 1 << (jf - 1) // isol=bit0, fina=bit1, ... matches GlyphFlag enum order
}

// joiningFeatureMask is the set of all joining feature glyph flags.
const joiningFeatureMask = GlyphFlagIsol | GlyphFlagFina | GlyphFlagFin2 | GlyphFlagFin3 |
	GlyphFlagMedi | GlyphFlagMed2 | GlyphFlagInit

// assignJoiningForms assigns Arabic joining forms (isol/init/medi/fina) to glyphs
// based on their Unicode Joining_Type property and neighboring glyphs.
//
// The algorithm mirrors the C implementation (KBTS__OP_KIND_FLAG_JOINING_LETTERS):
//   - Skip transparent glyphs (marks)
//   - For each non-transparent glyph, check if it can join with the previous non-transparent glyph
//   - Use a transition table to update the previous glyph's form when a join occurs
func assignJoiningForms(glyphs []Glyph) {
	// canJoinRight is a bitmask: for a given previous joining type (byte index),
	// bit positions indicate which current joining types can join.
	// Left, Dual, and Force types can join to the right of Right, Dual, and Force types.
	const canJoinRight = (1<<joiningTypeRight | 1<<joiningTypeDual | 1<<joiningTypeForce) << (8 * joiningTypeLeft) |
		(1<<joiningTypeRight | 1<<joiningTypeDual | 1<<joiningTypeForce) << (8 * joiningTypeDual) |
		(1<<joiningTypeRight | 1<<joiningTypeDual | 1<<joiningTypeForce) << (8 * joiningTypeForce)

	// transition maps a glyph's current joining feature to its new form when
	// a following glyph joins to it: isol→init, fina→medi.
	const transition = uint64(joiningFeatureInit)<<(8*joiningFeatureIsol) |
		uint64(joiningFeatureMedi)<<(8*joiningFeatureFina) |
		uint64(joiningFeatureMedi)<<(8*joiningFeatureMedi) |
		uint64(joiningFeatureMed2)<<(8*joiningFeatureMed2)

	prevIdx := -1 // index of previous non-transparent glyph
	var prevJT joiningType

	for i := range glyphs {
		jt := getJoiningType(glyphs[i].Codepoint)
		if jt == joiningTypeTransparent {
			continue
		}

		var jf joiningFeature
		if prevIdx >= 0 && canJoinRight&(1<<uint(jt+8*prevJT)) != 0 {
			// Join succeeds: update previous glyph's form via transition table.
			prevJF := joiningFeature((transition >> (8 * uint(glyphs[prevIdx].joiningFeature))) & 0xFF)
			glyphs[prevIdx].joiningFeature = prevJF
			glyphs[prevIdx].Flags = (glyphs[prevIdx].Flags &^ joiningFeatureMask) | joiningFeatureToGlyphFlag(prevJF)

			// Current glyph gets final form.
			jf = joiningFeatureFina
		} else {
			// No join: current glyph gets isolated form.
			jf = joiningFeatureIsol
		}

		glyphs[i].joiningFeature = jf
		glyphs[i].Flags = (glyphs[i].Flags &^ joiningFeatureMask) | joiningFeatureToGlyphFlag(jf)

		prevIdx = i
		prevJT = jt
	}
}

// applyArabicShaping runs the full Arabic shaping pipeline on glyphs:
// 1. Assign joining forms based on neighbors
// 2. Apply ccmp/locl GSUB features
// 3. Apply joining GSUB features (isol/fina/medi/init) filtered by each glyph's joining form flag
// 4. Apply post-joining GSUB features (rlig/rclt/calt/liga)
func (f *Font) applyArabicShaping(glyphs []Glyph, disabledFeatures map[FeatureTag]bool) []Glyph {
	assignJoiningForms(glyphs)

	// Phase 1: ccmp/locl.
	glyphs = f.applyGSUBFeatures(glyphs, arabicGSUBFeatures[:], disabledFeatures)

	// Phase 2: joining features, filtered by glyph flag.
	for _, jf := range arabicJoiningFeatures {
		if disabledFeatures != nil && disabledFeatures[jf.tag] {
			continue
		}
		requiredFlag := joiningFeatureToGlyphFlag(jf.feature)
		glyphs = f.applyGSUBFeaturesFiltered(glyphs, jf.tag, disabledFeatures, requiredFlag)
	}

	// Phase 3: post-joining features.
	glyphs = f.applyGSUBFeatures(glyphs, arabicPostJoiningFeatures[:], disabledFeatures)

	return glyphs
}

// applyGSUBFeaturesFiltered applies a single GSUB feature, but only substitutes
// glyphs whose Flags contain requiredFlag. Used for Arabic joining features.
// Only single substitution (type 1) is filtered; other lookup types apply normally.
func (f *Font) applyGSUBFeaturesFiltered(glyphs []Glyph, tag FeatureTag, disabledFeatures map[FeatureTag]bool, requiredFlag GlyphFlags) []Glyph {
	if disabledFeatures != nil && disabledFeatures[tag] {
		return glyphs
	}
	te := f.tables[tableGsub]
	if te.length == 0 {
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

	var idxBuf [4]int
	featureIndices := f.findGSUBFeatureIndices(idxBuf[:0], tag)
	for _, fi := range featureIndices {
		lookupIndices := f.gsubFeatureLookups(fi)
		for _, li := range lookupIndices {
			if int(li) >= lookupCount {
				continue
			}
			lookupOff := lookupListOff + int(readU16BE(f.data, lookupListOff+2+int(li)*2))
			glyphs = f.applyGSUBLookupFiltered(glyphs, lookupOff, requiredFlag)
		}
	}
	return glyphs
}

// applyGSUBLookupFiltered applies a single GSUB lookup, but only to glyphs
// whose Flags contain requiredFlag. For type 1 (single substitution),
// non-matching glyphs are skipped. Other types apply normally.
func (f *Font) applyGSUBLookupFiltered(glyphs []Glyph, lookupOff int, requiredFlag GlyphFlags) []Glyph {
	if lookupOff+6 > len(f.data) {
		return glyphs
	}
	lookupType := readU16BE(f.data, lookupOff)
	subtableCount := int(readU16BE(f.data, lookupOff+4))
	if lookupOff+6+subtableCount*2 > len(f.data) {
		return glyphs
	}

	for si := 0; si < subtableCount; si++ {
		subtableOff := lookupOff + int(readU16BE(f.data, lookupOff+6+si*2))
		switch lookupType {
		case 1:
			glyphs = f.applySingleSubstFiltered(glyphs, subtableOff, requiredFlag)
		case 7:
			// Extension: resolve inner lookup type and apply filtered if type 1.
			if subtableOff+8 > len(f.data) {
				continue
			}
			innerType := readU16BE(f.data, subtableOff+2)
			innerOff := subtableOff + int(readU32BE(f.data, subtableOff+4))
			if innerType == 1 {
				glyphs = f.applySingleSubstFiltered(glyphs, innerOff, requiredFlag)
			} else {
				// For non-single-subst extension lookups, apply unfiltered.
				glyphs = f.applyGSUBLookup(glyphs, lookupOff)
			}
		default:
			// Non-single-subst lookups: apply unfiltered.
			glyphs = f.applyGSUBLookup(glyphs, lookupOff)
		}
	}
	return glyphs
}

// applySingleSubstFiltered applies GSUB type 1 only to glyphs with requiredFlag set.
func (f *Font) applySingleSubstFiltered(glyphs []Glyph, subtableOff int, requiredFlag GlyphFlags) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))

	switch format {
	case 1:
		deltaGlyphID := int16(readU16BE(f.data, subtableOff+4))
		for i := range glyphs {
			if glyphs[i].Flags&requiredFlag == 0 {
				continue
			}
			covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
			if covIdx >= 0 {
				glyphs[i].ID = uint16(int32(glyphs[i].ID) + int32(deltaGlyphID))
				glyphs[i].Flags |= GlyphFlagGeneratedByGSUB
			}
		}
	case 2:
		glyphCount := int(readU16BE(f.data, subtableOff+4))
		for i := range glyphs {
			if glyphs[i].Flags&requiredFlag == 0 {
				continue
			}
			covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
			if covIdx >= 0 && covIdx < glyphCount {
				newID := readU16BE(f.data, subtableOff+6+covIdx*2)
				glyphs[i].ID = newID
				glyphs[i].Flags |= GlyphFlagGeneratedByGSUB
			}
		}
	}
	return glyphs
}

// applyGSUBFeatures applies GSUB substitutions for a set of features.
// disabledFeatures is a set of feature tags that have been disabled via overrides.
func (f *Font) applyGSUBFeatures(glyphs []Glyph, features []FeatureTag, disabledFeatures map[FeatureTag]bool) []Glyph {
	te := f.tables[tableGsub]
	if te.length == 0 {
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

	var idxBuf [4]int
	for _, tag := range features {
		if disabledFeatures != nil && disabledFeatures[tag] {
			continue
		}
		featureIndices := f.findGSUBFeatureIndices(idxBuf[:0], tag)
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
	}
	return glyphs
}

// applyGSUBLigatures applies GSUB ligature substitutions (lookup type 4) to glyphs.
// Deprecated: Use applyGSUBFeatures instead. Kept for backward compatibility.
func (f *Font) applyGSUBLigatures(glyphs []Glyph) []Glyph {
	return f.applyGSUBFeatures(glyphs, []FeatureTag{FeatureTagLiga}, nil)
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

	for si := 0; si < subtableCount; si++ {
		subtableOff := lookupOff + int(readU16BE(f.data, lookupOff+6+si*2))
		switch lookupType {
		case 1:
			glyphs = f.applySingleSubst(glyphs, subtableOff)
		case 2:
			glyphs = f.applyMultipleSubst(glyphs, subtableOff)
		case 3:
			glyphs = f.applyAlternateSubst(glyphs, subtableOff, 0)
		case 4:
			glyphs = f.applyLigatureSubtable(glyphs, subtableOff)
		case 5:
			glyphs = f.applyContextSubst(glyphs, subtableOff)
		case 6:
			glyphs = f.applyChainContextSubst(glyphs, subtableOff)
		case 7:
			glyphs = f.applyExtensionSubst(glyphs, subtableOff)
		case 8:
			glyphs = f.applyReverseChainSubst(glyphs, subtableOff)
		}
	}
	return glyphs
}

// applySingleSubst applies a GSUB type 1 (single substitution) subtable.
// Format 1: replace glyph ID by adding a delta.
// Format 2: replace glyph ID from an array indexed by coverage index.
func (f *Font) applySingleSubst(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))

	switch format {
	case 1:
		// Format 1: delta substitution.
		deltaGlyphID := int16(readU16BE(f.data, subtableOff+4))
		for i := range glyphs {
			covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
			if covIdx >= 0 {
				// "Addition of deltaGlyphID is modulo 65536."
				glyphs[i].ID = uint16(int32(glyphs[i].ID) + int32(deltaGlyphID))
				glyphs[i].Flags |= GlyphFlagGeneratedByGSUB
			}
		}
	case 2:
		// Format 2: substitution array.
		glyphCount := int(readU16BE(f.data, subtableOff+4))
		if subtableOff+6+glyphCount*2 > len(f.data) {
			return glyphs
		}
		for i := range glyphs {
			covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
			if covIdx >= 0 && covIdx < glyphCount {
				glyphs[i].ID = readU16BE(f.data, subtableOff+6+covIdx*2)
				glyphs[i].Flags |= GlyphFlagGeneratedByGSUB
			}
		}
	}
	return glyphs
}

// applyMultipleSubst applies a GSUB type 2 (multiple substitution) subtable.
// Replaces one glyph with a sequence of glyphs (1:N).
func (f *Font) applyMultipleSubst(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return glyphs
	}
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	seqCount := int(readU16BE(f.data, subtableOff+4))
	if subtableOff+6+seqCount*2 > len(f.data) {
		return glyphs
	}

	i := 0
	for i < len(glyphs) {
		covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
		if covIdx < 0 || covIdx >= seqCount {
			i++
			continue
		}
		seqOff := subtableOff + int(readU16BE(f.data, subtableOff+6+covIdx*2))
		if seqOff+2 > len(f.data) {
			i++
			continue
		}
		substCount := int(readU16BE(f.data, seqOff))
		if substCount == 0 {
			// Deletion: remove the glyph.
			glyphs = append(glyphs[:i], glyphs[i+1:]...)
			continue
		}
		if seqOff+2+substCount*2 > len(f.data) {
			i++
			continue
		}
		// Replace current glyph with first substitute.
		origGlyph := glyphs[i]
		glyphs[i].ID = readU16BE(f.data, seqOff+2)
		glyphs[i].Flags |= GlyphFlagMultipleSubstitution | GlyphFlagFirstInMultiple | GlyphFlagGeneratedByGSUB

		if substCount > 1 {
			// Insert remaining substitutes after position i.
			newGlyphs := make([]Glyph, substCount-1)
			for j := range newGlyphs {
				newGlyphs[j] = origGlyph
				newGlyphs[j].ID = readU16BE(f.data, seqOff+2+(j+1)*2)
				newGlyphs[j].Flags |= GlyphFlagMultipleSubstitution | GlyphFlagGeneratedByGSUB
			}
			// Insert after i: glyphs[:i+1] + newGlyphs + glyphs[i+1:]
			glyphs = append(glyphs[:i+1], append(newGlyphs, glyphs[i+1:]...)...)
			i += substCount
		} else {
			i++
		}
	}
	return glyphs
}

// applyAlternateSubst applies a GSUB type 3 (alternate substitution) subtable.
// altIndex selects which alternate to use (0-based). If out of range, uses index 0.
func (f *Font) applyAlternateSubst(glyphs []Glyph, subtableOff int, altIndex int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return glyphs
	}
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	altSetCount := int(readU16BE(f.data, subtableOff+4))
	if subtableOff+6+altSetCount*2 > len(f.data) {
		return glyphs
	}

	for i := range glyphs {
		covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
		if covIdx < 0 || covIdx >= altSetCount {
			continue
		}
		altSetOff := subtableOff + int(readU16BE(f.data, subtableOff+6+covIdx*2))
		if altSetOff+2 > len(f.data) {
			continue
		}
		altCount := int(readU16BE(f.data, altSetOff))
		if altCount == 0 || altSetOff+2+altCount*2 > len(f.data) {
			continue
		}
		idx := altIndex
		if idx >= altCount {
			idx = 0
		}
		glyphs[i].ID = readU16BE(f.data, altSetOff+2+idx*2)
		glyphs[i].Flags |= GlyphFlagGeneratedByGSUB
	}
	return glyphs
}

// gsubLookupOffset resolves a lookup index to an absolute offset in the font data.
// Returns -1 if the index is out of range.
func (f *Font) gsubLookupOffset(lookupIndex uint16) int {
	te := f.tables[tableGsub]
	if te.length == 0 {
		return -1
	}
	base := int(te.offset)
	if base+10 > len(f.data) {
		return -1
	}
	lookupListOff := base + int(readU16BE(f.data, base+8))
	if lookupListOff+2 > len(f.data) {
		return -1
	}
	lookupCount := int(readU16BE(f.data, lookupListOff))
	if int(lookupIndex) >= lookupCount {
		return -1
	}
	off := lookupListOff + 2 + int(lookupIndex)*2
	if off+2 > len(f.data) {
		return -1
	}
	return lookupListOff + int(readU16BE(f.data, off))
}

// applyExtensionSubst applies a GSUB type 7 (extension substitution) subtable.
// This is just an indirection: it reads the real lookup type and offset, then dispatches.
func (f *Font) applyExtensionSubst(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+8 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return glyphs
	}
	extLookupType := readU16BE(f.data, subtableOff+2)
	extOffset := int(readU32BE(f.data, subtableOff+4))
	realSubtableOff := subtableOff + extOffset

	switch extLookupType {
	case 1:
		glyphs = f.applySingleSubst(glyphs, realSubtableOff)
	case 2:
		glyphs = f.applyMultipleSubst(glyphs, realSubtableOff)
	case 3:
		glyphs = f.applyAlternateSubst(glyphs, realSubtableOff, 0)
	case 4:
		glyphs = f.applyLigatureSubtable(glyphs, realSubtableOff)
	case 5:
		glyphs = f.applyContextSubst(glyphs, realSubtableOff)
	case 6:
		glyphs = f.applyChainContextSubst(glyphs, realSubtableOff)
	case 8:
		glyphs = f.applyReverseChainSubst(glyphs, realSubtableOff)
	}
	return glyphs
}

// sequenceLookupRecord represents an OpenType SequenceLookupRecord.
type sequenceLookupRecord struct {
	sequenceIndex   uint16
	lookupListIndex uint16
}

// applySequenceLookups applies nested lookups from SequenceLookupRecords to matched glyphs.
// matchStart is the index of the first matched glyph. matchLen is the number of matched input glyphs.
func (f *Font) applySequenceLookups(glyphs []Glyph, matchStart int, records []sequenceLookupRecord) []Glyph {
	for _, rec := range records {
		pos := matchStart + int(rec.sequenceIndex)
		if pos >= len(glyphs) {
			continue
		}
		lookupOff := f.gsubLookupOffset(rec.lookupListIndex)
		if lookupOff < 0 || lookupOff+6 > len(f.data) {
			continue
		}
		lookupType := readU16BE(f.data, lookupOff)
		subtableCount := int(readU16BE(f.data, lookupOff+4))
		if lookupOff+6+subtableCount*2 > len(f.data) {
			continue
		}
		// Apply each subtable of the nested lookup to the single glyph at pos.
		for si := 0; si < subtableCount; si++ {
			stOff := lookupOff + int(readU16BE(f.data, lookupOff+6+si*2))
			switch lookupType {
			case 1:
				// Single substitution — apply only to glyph at pos.
				if stOff+6 > len(f.data) {
					continue
				}
				format := readU16BE(f.data, stOff)
				coverageOff := stOff + int(readU16BE(f.data, stOff+2))
				covIdx := f.coverageIndex(coverageOff, glyphs[pos].ID)
				if covIdx < 0 {
					continue
				}
				switch format {
				case 1:
					delta := int16(readU16BE(f.data, stOff+4))
					glyphs[pos].ID = uint16(int32(glyphs[pos].ID) + int32(delta))
					glyphs[pos].Flags |= GlyphFlagGeneratedByGSUB
				case 2:
					glyphCount := int(readU16BE(f.data, stOff+4))
					if covIdx < glyphCount && stOff+6+covIdx*2+2 <= len(f.data) {
						glyphs[pos].ID = readU16BE(f.data, stOff+6+covIdx*2)
						glyphs[pos].Flags |= GlyphFlagGeneratedByGSUB
					}
				}
			case 4:
				// Ligature — apply at pos within the remaining slice.
				glyphs = f.applyLigatureSubtable(glyphs, stOff)
			}
		}
	}
	return glyphs
}

// readSequenceLookupRecords reads n SequenceLookupRecords starting at off.
func (f *Font) readSequenceLookupRecords(off int, n int) []sequenceLookupRecord {
	if n <= 0 || off+n*4 > len(f.data) {
		return nil
	}
	records := make([]sequenceLookupRecord, n)
	for i := range records {
		records[i].sequenceIndex = readU16BE(f.data, off+i*4)
		records[i].lookupListIndex = readU16BE(f.data, off+i*4+2)
	}
	return records
}

// applyContextSubst applies a GSUB type 5 (context substitution) subtable.
// Currently implements format 3 (coverage-based).
func (f *Font) applyContextSubst(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	switch format {
	case 3:
		return f.applyContextSubstFormat3(glyphs, subtableOff)
	}
	return glyphs
}

// applyContextSubstFormat3 implements context substitution format 3 (coverage-based).
// Layout: u16 format=3, u16 glyphCount, u16 seqLookupCount, u16[glyphCount] coverageOffsets,
//
//	{u16 seqIdx, u16 lookupIdx}[seqLookupCount]
func (f *Font) applyContextSubstFormat3(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	glyphCount := int(readU16BE(f.data, subtableOff+2))
	seqLookupCount := int(readU16BE(f.data, subtableOff+4))
	if glyphCount < 1 {
		return glyphs
	}
	coveragesOff := subtableOff + 6
	if coveragesOff+glyphCount*2+seqLookupCount*4 > len(f.data) {
		return glyphs
	}
	recordsOff := coveragesOff + glyphCount*2

	i := 0
	for i <= len(glyphs)-glyphCount {
		// Check all input glyphs against their respective coverages.
		matched := true
		for gi := 0; gi < glyphCount; gi++ {
			covOff := subtableOff + int(readU16BE(f.data, coveragesOff+gi*2))
			if f.coverageIndex(covOff, glyphs[i+gi].ID) < 0 {
				matched = false
				break
			}
		}
		if !matched {
			i++
			continue
		}
		records := f.readSequenceLookupRecords(recordsOff, seqLookupCount)
		glyphs = f.applySequenceLookups(glyphs, i, records)
		i += glyphCount
	}
	return glyphs
}

// applyChainContextSubst applies a GSUB type 6 (chaining context substitution) subtable.
// Currently implements format 3 (coverage-based).
func (f *Font) applyChainContextSubst(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+4 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	switch format {
	case 3:
		return f.applyChainContextSubstFormat3(glyphs, subtableOff)
	}
	return glyphs
}

// applyChainContextSubstFormat3 implements chaining context substitution format 3.
// Variable-length layout:
//
//	u16 format=3
//	u16 backtrackGlyphCount
//	u16[backtrackGlyphCount] backtrackCoverageOffsets
//	u16 inputGlyphCount
//	u16[inputGlyphCount] inputCoverageOffsets
//	u16 lookaheadGlyphCount
//	u16[lookaheadGlyphCount] lookaheadCoverageOffsets
//	u16 seqLookupCount
//	{u16 seqIdx, u16 lookupIdx}[seqLookupCount]
func (f *Font) applyChainContextSubstFormat3(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+4 > len(f.data) {
		return glyphs
	}
	off := subtableOff + 2
	// Backtrack coverages.
	backtrackCount := int(readU16BE(f.data, off))
	off += 2
	backtrackCovsOff := off
	off += backtrackCount * 2
	if off+2 > len(f.data) {
		return glyphs
	}
	// Input coverages.
	inputCount := int(readU16BE(f.data, off))
	off += 2
	inputCovsOff := off
	off += inputCount * 2
	if off+2 > len(f.data) {
		return glyphs
	}
	// Lookahead coverages.
	lookaheadCount := int(readU16BE(f.data, off))
	off += 2
	lookaheadCovsOff := off
	off += lookaheadCount * 2
	if off+2 > len(f.data) {
		return glyphs
	}
	// Sequence lookup records.
	seqLookupCount := int(readU16BE(f.data, off))
	off += 2
	recordsOff := off
	if recordsOff+seqLookupCount*4 > len(f.data) {
		return glyphs
	}
	if inputCount < 1 {
		return glyphs
	}

	for i := 0; i <= len(glyphs)-inputCount; i++ {
		// Check backtrack: glyphs before i, in reverse order.
		if i < backtrackCount {
			continue
		}
		backtrackOK := true
		for bi := 0; bi < backtrackCount; bi++ {
			covOff := subtableOff + int(readU16BE(f.data, backtrackCovsOff+bi*2))
			// Backtrack[0] matches glyph immediately before input, etc.
			if f.coverageIndex(covOff, glyphs[i-1-bi].ID) < 0 {
				backtrackOK = false
				break
			}
		}
		if !backtrackOK {
			continue
		}

		// Check input sequence.
		inputOK := true
		for gi := 0; gi < inputCount; gi++ {
			if i+gi >= len(glyphs) {
				inputOK = false
				break
			}
			covOff := subtableOff + int(readU16BE(f.data, inputCovsOff+gi*2))
			if f.coverageIndex(covOff, glyphs[i+gi].ID) < 0 {
				inputOK = false
				break
			}
		}
		if !inputOK {
			continue
		}

		// Check lookahead: glyphs after input sequence.
		lookaheadStart := i + inputCount
		lookaheadOK := true
		for li := 0; li < lookaheadCount; li++ {
			if lookaheadStart+li >= len(glyphs) {
				lookaheadOK = false
				break
			}
			covOff := subtableOff + int(readU16BE(f.data, lookaheadCovsOff+li*2))
			if f.coverageIndex(covOff, glyphs[lookaheadStart+li].ID) < 0 {
				lookaheadOK = false
				break
			}
		}
		if !lookaheadOK {
			continue
		}

		// All matched — apply sequence lookups.
		records := f.readSequenceLookupRecords(recordsOff, seqLookupCount)
		glyphs = f.applySequenceLookups(glyphs, i, records)
		i += inputCount - 1 // advance past matched input
	}
	return glyphs
}

// applyReverseChainSubst applies a GSUB type 8 (reverse chaining context single substitution).
// Processes glyphs in reverse order (right to left).
// Layout:
//
//	u16 format=1, u16 coverageOffset, u16 backtrackGlyphCount,
//	u16[backtrackCount] backtrackCoverageOffsets,
//	u16 lookaheadGlyphCount, u16[lookaheadCount] lookaheadCoverageOffsets,
//	u16 glyphCount, u16[glyphCount] substituteGlyphIDs
func (f *Font) applyReverseChainSubst(glyphs []Glyph, subtableOff int) []Glyph {
	if subtableOff+6 > len(f.data) {
		return glyphs
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return glyphs
	}
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	off := subtableOff + 4

	// Backtrack coverages.
	backtrackCount := int(readU16BE(f.data, off))
	off += 2
	backtrackCovsOff := off
	off += backtrackCount * 2
	if off+2 > len(f.data) {
		return glyphs
	}
	// Lookahead coverages.
	lookaheadCount := int(readU16BE(f.data, off))
	off += 2
	lookaheadCovsOff := off
	off += lookaheadCount * 2
	if off+2 > len(f.data) {
		return glyphs
	}
	// Substitute glyph IDs.
	substGlyphCount := int(readU16BE(f.data, off))
	off += 2
	substOff := off
	if substOff+substGlyphCount*2 > len(f.data) {
		return glyphs
	}

	// Process in reverse order.
	for i := len(glyphs) - 1; i >= 0; i-- {
		covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
		if covIdx < 0 || covIdx >= substGlyphCount {
			continue
		}

		// Check backtrack: glyphs before i, in reverse.
		if i < backtrackCount {
			continue
		}
		backtrackOK := true
		for bi := 0; bi < backtrackCount; bi++ {
			covOff := subtableOff + int(readU16BE(f.data, backtrackCovsOff+bi*2))
			if f.coverageIndex(covOff, glyphs[i-1-bi].ID) < 0 {
				backtrackOK = false
				break
			}
		}
		if !backtrackOK {
			continue
		}

		// Check lookahead: glyphs after i.
		lookaheadOK := true
		for li := 0; li < lookaheadCount; li++ {
			if i+1+li >= len(glyphs) {
				lookaheadOK = false
				break
			}
			covOff := subtableOff + int(readU16BE(f.data, lookaheadCovsOff+li*2))
			if f.coverageIndex(covOff, glyphs[i+1+li].ID) < 0 {
				lookaheadOK = false
				break
			}
		}
		if !lookaheadOK {
			continue
		}

		// Substitute.
		glyphs[i].ID = readU16BE(f.data, substOff+covIdx*2)
		glyphs[i].Flags |= GlyphFlagGeneratedByGSUB
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

// GPOS ValueFormat bitmask constants.
const (
	valueFormatXPlacement       = 1 << 0
	valueFormatYPlacement       = 1 << 1
	valueFormatXAdvance         = 1 << 2
	valueFormatYAdvance         = 1 << 3
	valueFormatXPlacementDevice = 1 << 4
	valueFormatYPlacementDevice = 1 << 5
	valueFormatXAdvanceDevice   = 1 << 6
	valueFormatYAdvanceDevice   = 1 << 7
)

// valueRecordSize returns the number of uint16 fields in a value record for the given format.
func valueRecordSize(format uint16) int {
	n := 0
	for format != 0 {
		n += int(format & 1)
		format >>= 1
	}
	return n
}

// unpackValueRecord reads a GPOS ValueRecord and applies it to a glyph.
func unpackValueRecord(data []byte, off int, format uint16, g *Glyph) {
	if format == 0 {
		return
	}
	at := off
	if format&valueFormatXPlacement != 0 {
		if at+2 <= len(data) {
			g.OffsetX += int32(readS16BE(data, at))
		}
		at += 2
	}
	if format&valueFormatYPlacement != 0 {
		if at+2 <= len(data) {
			g.OffsetY += int32(readS16BE(data, at))
		}
		at += 2
	}
	if format&valueFormatXAdvance != 0 {
		if at+2 <= len(data) {
			g.AdvanceX += int32(readS16BE(data, at))
		}
		at += 2
	}
	if format&valueFormatYAdvance != 0 {
		if at+2 <= len(data) {
			g.AdvanceY += int32(readS16BE(data, at))
		}
		at += 2
	}
	// Skip device table offsets (4 possible).
	// We don't apply device adjustments for now.
}

// findGPOSFeatureIndices returns all indices in the GPOS FeatureList matching a feature tag.
func (f *Font) findGPOSFeatureIndices(dst []int, tag FeatureTag) []int {
	te := f.tables[tableGpos]
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

// gposFeatureLookups returns the lookup indices for a GPOS feature by feature index.
func (f *Font) gposFeatureLookups(featureIndex int) []uint16 {
	te := f.tables[tableGpos]
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

// defaultGPOSFeatures is the standard set of GPOS features applied in order.
// mark/mkmk must come after kern so that mark positioning accounts for kerning adjustments.
var defaultGPOSFeatures = [...]FeatureTag{
	FeatureTagKern, // Kerning (pair adjustment)
	FeatureTagMark, // Mark-to-base attachment
	FeatureTagMkmk, // Mark-to-mark attachment
}

// applyGPOSFeatures applies GPOS positioning features to glyphs.
// Modifies glyphs in place.
func (f *Font) applyGPOSFeatures(glyphs []Glyph, disabledFeatures map[FeatureTag]bool) {
	te := f.tables[tableGpos]
	if te.length == 0 {
		return
	}
	base := int(te.offset)
	if base+10 > len(f.data) {
		return
	}
	lookupListOff := base + int(readU16BE(f.data, base+8))
	if lookupListOff+2 > len(f.data) {
		return
	}
	lookupCount := int(readU16BE(f.data, lookupListOff))

	var idxBuf [4]int
	for _, tag := range defaultGPOSFeatures {
		if disabledFeatures != nil && disabledFeatures[tag] {
			continue
		}
		featureIndices := f.findGPOSFeatureIndices(idxBuf[:0], tag)
		for _, fi := range featureIndices {
			lookupIndices := f.gposFeatureLookups(fi)
			for _, li := range lookupIndices {
				if int(li) >= lookupCount {
					continue
				}
				lookupOff := lookupListOff + int(readU16BE(f.data, lookupListOff+2+int(li)*2))
				f.applyGPOSLookup(glyphs, lookupOff)
			}
		}
	}
}

// applyGPOSKerning applies GPOS kerning (kern feature, lookup type 2) to glyphs.
// Deprecated: Use applyGPOSFeatures instead. Kept for backward compatibility.
func (f *Font) applyGPOSKerning(glyphs []Glyph, disabledFeatures map[FeatureTag]bool) {
	f.applyGPOSFeatures(glyphs, disabledFeatures)
}

// applyGPOSLookup applies a single GPOS lookup to glyphs.
func (f *Font) applyGPOSLookup(glyphs []Glyph, lookupOff int) {
	if lookupOff+6 > len(f.data) {
		return
	}
	lookupType := readU16BE(f.data, lookupOff)
	subtableCount := int(readU16BE(f.data, lookupOff+4))
	if lookupOff+6+subtableCount*2 > len(f.data) {
		return
	}

	for si := 0; si < subtableCount; si++ {
		subtableOff := lookupOff + int(readU16BE(f.data, lookupOff+6+si*2))
		switch lookupType {
		case 1:
			f.applyGPOSSingleAdjust(glyphs, subtableOff)
		case 2:
			f.applyGPOSPairAdjust(glyphs, subtableOff)
		case 4:
			f.applyGPOSMarkToBase(glyphs, subtableOff)
		case 6:
			f.applyGPOSMarkToMark(glyphs, subtableOff)
		case 9:
			f.applyGPOSExtension(glyphs, subtableOff)
		}
	}
}

// applyGPOSExtension handles GPOS type 9 (extension positioning).
func (f *Font) applyGPOSExtension(glyphs []Glyph, subtableOff int) {
	if subtableOff+8 > len(f.data) {
		return
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return
	}
	extLookupType := readU16BE(f.data, subtableOff+2)
	extOffset := int(readU32BE(f.data, subtableOff+4))
	realOff := subtableOff + extOffset

	switch extLookupType {
	case 1:
		f.applyGPOSSingleAdjust(glyphs, realOff)
	case 2:
		f.applyGPOSPairAdjust(glyphs, realOff)
	case 4:
		f.applyGPOSMarkToBase(glyphs, realOff)
	case 6:
		f.applyGPOSMarkToMark(glyphs, realOff)
	}
}

// readAnchor reads an anchor table at the given absolute offset, returning X and Y.
func readAnchor(data []byte, off int) (x, y int16) {
	if off+6 > len(data) {
		return 0, 0
	}
	// Format 1, 2, and 3 all have X at offset 2, Y at offset 4.
	return readS16BE(data, off+2), readS16BE(data, off+4)
}

// applyGPOSMarkToBase applies GPOS type 4 (mark-to-base attachment).
// For each mark glyph covered by MarkCoverage, finds the preceding base glyph
// covered by BaseCoverage and positions the mark relative to the base using anchor points.
func (f *Font) applyGPOSMarkToBase(glyphs []Glyph, subtableOff int) {
	// MarkToBase subtable (format 1 only):
	//   u16 format (must be 1)
	//   u16 markCoverageOffset
	//   u16 baseCoverageOffset
	//   u16 markClassCount
	//   u16 markArrayOffset
	//   u16 baseArrayOffset
	if subtableOff+12 > len(f.data) {
		return
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return
	}
	markCovOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	baseCovOff := subtableOff + int(readU16BE(f.data, subtableOff+4))
	markClassCount := int(readU16BE(f.data, subtableOff+6))
	markArrayOff := subtableOff + int(readU16BE(f.data, subtableOff+8))
	baseArrayOff := subtableOff + int(readU16BE(f.data, subtableOff+10))

	for i := range glyphs {
		markCovIdx := f.coverageIndex(markCovOff, glyphs[i].ID)
		if markCovIdx < 0 {
			continue
		}

		// Find preceding base glyph (skip marks).
		baseIdx := -1
		var advSinceBaseX, advSinceBaseY int32
		for j := i - 1; j >= 0; j-- {
			cls := f.glyphClassDef(glyphs[j].ID)
			if cls != glyphClassMark {
				advSinceBaseX += glyphs[j].AdvanceX
				advSinceBaseY += glyphs[j].AdvanceY
			}
			if cls == glyphClassBase || cls == glyphClassLigature || cls == glyphClassZero {
				baseCovIdx := f.coverageIndex(baseCovOff, glyphs[j].ID)
				if baseCovIdx >= 0 {
					baseIdx = j
					f.attachMark(glyphs, i, baseIdx, markArrayOff, markCovIdx, baseArrayOff, baseCovIdx, markClassCount, advSinceBaseX, advSinceBaseY)
				}
				break
			}
		}
		_ = baseIdx
	}
}

// applyGPOSMarkToMark applies GPOS type 6 (mark-to-mark attachment).
// Positions a combining mark relative to a preceding mark glyph.
func (f *Font) applyGPOSMarkToMark(glyphs []Glyph, subtableOff int) {
	// Same structure as MarkToBase but with mark1 (base mark) and mark2 (attaching mark).
	if subtableOff+12 > len(f.data) {
		return
	}
	format := readU16BE(f.data, subtableOff)
	if format != 1 {
		return
	}
	mark2CovOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	mark1CovOff := subtableOff + int(readU16BE(f.data, subtableOff+4))
	markClassCount := int(readU16BE(f.data, subtableOff+6))
	mark2ArrayOff := subtableOff + int(readU16BE(f.data, subtableOff+8))
	mark1ArrayOff := subtableOff + int(readU16BE(f.data, subtableOff+10))

	for i := range glyphs {
		mark2CovIdx := f.coverageIndex(mark2CovOff, glyphs[i].ID)
		if mark2CovIdx < 0 {
			continue
		}

		// Find preceding mark glyph (the base mark).
		var advSinceBaseX, advSinceBaseY int32
		for j := i - 1; j >= 0; j-- {
			cls := f.glyphClassDef(glyphs[j].ID)
			advSinceBaseX += glyphs[j].AdvanceX
			advSinceBaseY += glyphs[j].AdvanceY
			if cls == glyphClassMark {
				mark1CovIdx := f.coverageIndex(mark1CovOff, glyphs[j].ID)
				if mark1CovIdx >= 0 {
					f.attachMark(glyphs, i, j, mark2ArrayOff, mark2CovIdx, mark1ArrayOff, mark1CovIdx, markClassCount, advSinceBaseX, advSinceBaseY)
				}
				break
			}
			// Non-mark encountered before finding a base mark: stop.
			break
		}
	}
}

// attachMark positions a mark glyph relative to a base glyph using anchor tables.
// markArrayOff/markCovIdx identify the mark's anchor; baseArrayOff/baseCovIdx/markClassCount
// identify the base's anchor for the mark's class. advSince is the total advance between them.
func (f *Font) attachMark(glyphs []Glyph, markIdx, baseIdx int, markArrayOff, markCovIdx int, baseArrayOff, baseCovIdx, markClassCount int, advSinceX, advSinceY int32) {
	// MarkArray:
	//   u16 markCount
	//   MarkRecord[markCount]:
	//     u16 markClass
	//     u16 markAnchorOffset (from start of MarkArray)
	if markArrayOff+2 > len(f.data) {
		return
	}
	markCount := int(readU16BE(f.data, markArrayOff))
	if markCovIdx >= markCount {
		return
	}
	markRecOff := markArrayOff + 2 + markCovIdx*4
	if markRecOff+4 > len(f.data) {
		return
	}
	markClass := int(readU16BE(f.data, markRecOff))
	markAnchorOff := markArrayOff + int(readU16BE(f.data, markRecOff+2))

	if markClass >= markClassCount {
		return
	}

	// BaseArray:
	//   u16 baseCount
	//   BaseRecord[baseCount]:
	//     u16 anchorOffset[markClassCount] (from start of BaseArray)
	if baseArrayOff+2 > len(f.data) {
		return
	}
	baseCount := int(readU16BE(f.data, baseArrayOff))
	if baseCovIdx >= baseCount {
		return
	}
	baseRecOff := baseArrayOff + 2 + baseCovIdx*markClassCount*2 + markClass*2
	if baseRecOff+2 > len(f.data) {
		return
	}
	baseAnchorRelOff := int(readU16BE(f.data, baseRecOff))
	if baseAnchorRelOff == 0 {
		return // NULL anchor
	}
	baseAnchorOff := baseArrayOff + baseAnchorRelOff

	markAnchorX, markAnchorY := readAnchor(f.data, markAnchorOff)
	baseAnchorX, baseAnchorY := readAnchor(f.data, baseAnchorOff)

	glyphs[markIdx].OffsetX = glyphs[baseIdx].OffsetX - advSinceX + int32(baseAnchorX) - int32(markAnchorX)
	glyphs[markIdx].OffsetY = glyphs[baseIdx].OffsetY - advSinceY + int32(baseAnchorY) - int32(markAnchorY)
	glyphs[markIdx].Flags |= GlyphFlagUsedInGPOS
	glyphs[baseIdx].Flags |= GlyphFlagUsedInGPOS
}

// applyGPOSSingleAdjust applies GPOS type 1 (single adjustment).
func (f *Font) applyGPOSSingleAdjust(glyphs []Glyph, subtableOff int) {
	if subtableOff+6 > len(f.data) {
		return
	}
	format := readU16BE(f.data, subtableOff)
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	valueFormat := readU16BE(f.data, subtableOff+4)
	vrSize := valueRecordSize(valueFormat)

	switch format {
	case 1:
		// Single value record applied to all covered glyphs.
		vrOff := subtableOff + 6
		if vrOff+vrSize*2 > len(f.data) {
			return
		}
		for i := range glyphs {
			if f.coverageIndex(coverageOff, glyphs[i].ID) >= 0 {
				unpackValueRecord(f.data, vrOff, valueFormat, &glyphs[i])
				glyphs[i].Flags |= GlyphFlagUsedInGPOS
			}
		}
	case 2:
		// Per-glyph value records indexed by coverage index.
		recordCount := int(readU16BE(f.data, subtableOff+6))
		vrOff := subtableOff + 8
		if vrOff+recordCount*vrSize*2 > len(f.data) {
			return
		}
		for i := range glyphs {
			covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
			if covIdx >= 0 && covIdx < recordCount {
				unpackValueRecord(f.data, vrOff+covIdx*vrSize*2, valueFormat, &glyphs[i])
				glyphs[i].Flags |= GlyphFlagUsedInGPOS
			}
		}
	}
}

// applyGPOSPairAdjust applies GPOS type 2 (pair adjustment / kerning).
func (f *Font) applyGPOSPairAdjust(glyphs []Glyph, subtableOff int) {
	if subtableOff+10 > len(f.data) {
		return
	}
	format := readU16BE(f.data, subtableOff)
	coverageOff := subtableOff + int(readU16BE(f.data, subtableOff+2))
	valueFormat1 := readU16BE(f.data, subtableOff+4)
	valueFormat2 := readU16BE(f.data, subtableOff+6)
	size1 := valueRecordSize(valueFormat1)
	size2 := valueRecordSize(valueFormat2)

	switch format {
	case 1:
		f.applyPairAdjustFormat1(glyphs, subtableOff, coverageOff, valueFormat1, valueFormat2, size1, size2)
	case 2:
		f.applyPairAdjustFormat2(glyphs, subtableOff, coverageOff, valueFormat1, valueFormat2, size1, size2)
	}
}

// applyPairAdjustFormat1 implements GPOS pair adjustment format 1 (individual pairs).
func (f *Font) applyPairAdjustFormat1(glyphs []Glyph, subtableOff int, coverageOff int, vf1, vf2 uint16, size1, size2 int) {
	setCount := int(readU16BE(f.data, subtableOff+8))
	if subtableOff+10+setCount*2 > len(f.data) {
		return
	}

	for i := 0; i < len(glyphs)-1; i++ {
		covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
		if covIdx < 0 || covIdx >= setCount {
			continue
		}
		pairSetOff := subtableOff + int(readU16BE(f.data, subtableOff+10+covIdx*2))
		if pairSetOff+2 > len(f.data) {
			continue
		}
		pairCount := int(readU16BE(f.data, pairSetOff))
		pairRecordSize := 1 + size1 + size2 // SecondGlyph(1 u16) + vr1 + vr2
		recordsOff := pairSetOff + 2
		if recordsOff+pairCount*pairRecordSize*2 > len(f.data) {
			continue
		}

		nextGlyphID := glyphs[i+1].ID
		// Binary search for the pair.
		lo, hi := 0, pairCount-1
		for lo <= hi {
			mid := (lo + hi) / 2
			recOff := recordsOff + mid*pairRecordSize*2
			secondGlyph := readU16BE(f.data, recOff)
			if nextGlyphID < secondGlyph {
				hi = mid - 1
			} else if nextGlyphID > secondGlyph {
				lo = mid + 1
			} else {
				// Found pair — apply value records.
				vrOff := recOff + 2
				unpackValueRecord(f.data, vrOff, vf1, &glyphs[i])
				glyphs[i].Flags |= GlyphFlagUsedInGPOS
				if vf2 != 0 {
					unpackValueRecord(f.data, vrOff+size1*2, vf2, &glyphs[i+1])
					glyphs[i+1].Flags |= GlyphFlagUsedInGPOS
				}
				break
			}
		}
	}
}

// applyPairAdjustFormat2 implements GPOS pair adjustment format 2 (class-based pairs).
func (f *Font) applyPairAdjustFormat2(glyphs []Glyph, subtableOff int, coverageOff int, vf1, vf2 uint16, size1, size2 int) {
	if subtableOff+16 > len(f.data) {
		return
	}
	classDef1Off := subtableOff + int(readU16BE(f.data, subtableOff+8))
	classDef2Off := subtableOff + int(readU16BE(f.data, subtableOff+10))
	class1Count := int(readU16BE(f.data, subtableOff+12))
	class2Count := int(readU16BE(f.data, subtableOff+14))

	pairRecordSize := size1 + size2
	recordsOff := subtableOff + 16
	totalRecords := class1Count * class2Count * pairRecordSize * 2
	if recordsOff+totalRecords > len(f.data) {
		return
	}

	for i := 0; i < len(glyphs)-1; i++ {
		covIdx := f.coverageIndex(coverageOff, glyphs[i].ID)
		if covIdx < 0 {
			continue
		}
		class1 := int(classDefLookup(f.data, classDef1Off, glyphs[i].ID))
		class2 := int(classDefLookup(f.data, classDef2Off, glyphs[i+1].ID))
		if class1 >= class1Count || class2 >= class2Count {
			continue
		}
		vrOff := recordsOff + (class1*class2Count+class2)*pairRecordSize*2
		unpackValueRecord(f.data, vrOff, vf1, &glyphs[i])
		glyphs[i].Flags |= GlyphFlagUsedInGPOS
		if vf2 != 0 {
			unpackValueRecord(f.data, vrOff+size1*2, vf2, &glyphs[i+1])
			glyphs[i+1].Flags |= GlyphFlagUsedInGPOS
		}
	}
}
