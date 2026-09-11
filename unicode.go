package lefevre

// Unicode property enums and lookup functions.
// These mirror the C enums from kb_text_shape.h.

// bidiClass represents a Unicode Bidirectional class.
type bidiClass uint8

const (
	bidiClassNI  bidiClass = iota // Neutral/Isolate
	bidiClassBN                   // Boundary Neutral (formatting, ignored)
	bidiClassL                    // Left-to-Right
	bidiClassR                    // Right-to-Left
	bidiClassNSM                  // Non-Spacing Mark
	bidiClassAL                   // Arabic Letter
	bidiClassAN                   // Arabic Number
	bidiClassEN                   // European Number
	bidiClassES                   // European Separator
	bidiClassET                   // European Terminator
	bidiClassCS                   // Common Separator
	bidiClassCount
)

// lineBreakClass represents a Unicode Line Break class.
type lineBreakClass uint8

const (
	lbcOnea          lineBreakClass = iota // Other non-East Asian
	lbcOea                                 // Other East Asian
	lbcOpe                                 // Other pictographic/emoji
	lbcBK                                  // Mandatory Break
	lbcCR                                  // Carriage Return
	lbcLF                                  // Line Feed
	lbcNL                                  // Next Line
	lbcSP                                  // Space
	lbcZW                                  // Zero Width Space
	lbcWJ                                  // Word Joiner
	lbcGLnea                               // Glue non-EA
	lbcGLea                                // Glue EA
	lbcCLnea                               // Close non-EA
	lbcCLea                                // Close EA
	lbcCPnea                               // Close Parenthesis non-EA
	lbcCPea                                // Close Parenthesis EA
	lbcEXnea                               // Exclamation non-EA
	lbcEXea                                // Exclamation EA
	lbcSY                                  // Symbols allowing break after
	lbcBAnea                               // Break After non-EA
	lbcBAea                                // Break After EA
	lbcOPnea                               // Open Punctuation non-EA
	lbcOPea                                // Open Punctuation EA
	lbcQU                                  // Quotation
	lbcQUPi                                // Quotation Initial
	lbcQUPf                                // Quotation Final
	lbcIS                                  // Infix Separator
	lbcNSnea                               // Nonstarter non-EA
	lbcNSea                                // Nonstarter EA
	lbcB2                                  // Break Opportunity Before and After
	lbcCB                                  // Contingent Break
	lbcHY                                  // Hyphen
	lbcHYPHEN                              // Hyphen (U+2010)
	lbcINnea                               // Inseparable non-EA
	lbcINea                                // Inseparable EA
	lbcBB                                  // Break Before
	lbcHL                                  // Hebrew Letter
	lbcALnea                               // Alphabetic non-EA
	lbcALea                                // Alphabetic EA
	lbcNU                                  // Numeric
	lbcPRnea                               // Prefix non-EA
	lbcPRea                                // Prefix EA
	lbcIDnea                               // Ideographic non-EA
	lbcIDea                                // Ideographic EA
	lbcIDpe                                // Ideographic pictographic/emoji
	lbcEBnea                               // Emoji Base non-EA
	lbcEBea                                // Emoji Base EA
	lbcEM                                  // Emoji Modifier
	lbcPOnea                               // Postfix non-EA
	lbcPOea                                // Postfix EA
	lbcJL                                  // Jamo Leading
	lbcJV                                  // Jamo Vowel
	lbcJT                                  // Jamo Trailing
	lbcH2                                  // Hangul LV
	lbcH3                                  // Hangul LVT
	lbcAP                                  // Aksara Pre-Base
	lbcAK                                  // Aksara
	lbcDOTTED_CIRCLE                       // Dotted Circle
	lbcAS                                  // Aksara Start
	lbcVF                                  // Virama Final
	lbcVI                                  // Virama
	lbcRI                                  // Regional Indicator

	lbcCount lineBreakClass = 62

	lbcCM  lineBreakClass = 63 // Combining Mark
	lbcZWJ lineBreakClass = 64 // Zero Width Joiner
	lbcCJ  lineBreakClass = 65 // Conditional Japanese Starter
	lbcSOT lineBreakClass = 66 // Start of Text
	lbcEOT lineBreakClass = 67 // End of Text
)

// wordBreakClass represents a Unicode Word Break class.
type wordBreakClass uint8

const (
	wbcOnep  wordBreakClass = iota // Other non-emoji pictographic
	wbcOep                         // Other emoji pictographic
	wbcCR                          // Carriage Return
	wbcLF                          // Line Feed
	wbcNL                          // Newline
	wbcEX                          // Extend
	wbcZWJ                         // Zero Width Joiner
	wbcRI                          // Regional Indicator
	wbcFO                          // Format
	wbcKA                          // Katakana
	wbcHL                          // Hebrew Letter
	wbcALnep                       // ALetter non-emoji-pictographic
	wbcALep                        // ALetter emoji-pictographic
	wbcSQ                          // Single Quote
	wbcDQ                          // Double Quote
	wbcMNL                         // MidNumLet
	wbcML                          // MidLetter
	wbcMN                          // MidNum
	wbcNM                          // Numeric
	wbcENL                         // ExtendNumLet
	wbcWSS                         // WSegSpace

	wbcSOT wordBreakClass = 21 // Start of Text
)

// graphemeBreakClass represents a Unicode Grapheme Break class.
type graphemeBreakClass uint8

const (
	gbcDefault graphemeBreakClass = iota
	gbcCR
	gbcLF
	gbcControl
	gbcExtend
	gbcZWJ
	gbcSpacingMark
	gbcL
	gbcV
	gbcLV
	gbcLVT
	gbcT
	gbcPrepend
	gbcIndicConsonant
	gbcIndicExtend
	gbcIndicLinker
	gbcExtendedPictographic
	gbcRI

	gbcCount graphemeBreakClass = 18
)

// graphemeBreakState for the grapheme break state machine.
type graphemeBreakState uint8

const (
	gbsStart                      graphemeBreakState = 0
	gbsCR                         graphemeBreakState = 1
	gbsL                          graphemeBreakState = 2
	gbsLVxV                       graphemeBreakState = 3
	gbsLVTxT                      graphemeBreakState = 4
	gbsIndicConsonantxIndicLinker graphemeBreakState = 5
	gbsIndicExtendr               graphemeBreakState = 6
	gbsIndicExtendLinkerr         graphemeBreakState = 7
	gbsExtendedPictographic       graphemeBreakState = 8
	gbsExtendR                    graphemeBreakState = 9
	gbsExtendR_ZWJ                graphemeBreakState = 10
	gbsRI                         graphemeBreakState = 11
	gbsSKIP                       graphemeBreakState = 12
	gbsCount                      graphemeBreakState = 13

	gbsb0                             graphemeBreakState = 14
	gbsb01                            graphemeBreakState = 15
	gbsb1                             graphemeBreakState = 16
	gbsb1toCR                         graphemeBreakState = 17
	gbsb1toL                          graphemeBreakState = 18
	gbsb1toLVxV                       graphemeBreakState = 19
	gbsb1toLVTxT                      graphemeBreakState = 20
	gbsb1toIndicConsonantxIndicLinker graphemeBreakState = 21
	gbsPADDING0                       graphemeBreakState = 22
	gbsPADDING1                       graphemeBreakState = 23
	gbsb1toExtendedPictographic       graphemeBreakState = 24
	gbsPADDING2                       graphemeBreakState = 25
	gbsPADDING3                       graphemeBreakState = 26
	gbsb1toRI                         graphemeBreakState = 27
	gbsb1toSKIP                       graphemeBreakState = 28
)

// joiningType represents the Unicode Arabic Joining_Type property.
type joiningType uint8

const (
	joiningTypeNone        joiningType = 0 // Non_Joining
	joiningTypeLeft        joiningType = 1 // Left_Joining
	joiningTypeDual        joiningType = 2 // Dual_Joining
	joiningTypeForce       joiningType = 3 // Join_Causing (e.g., ZWJ)
	joiningTypeRight       joiningType = 4 // Right_Joining
	joiningTypeTransparent joiningType = 5 // Transparent (marks)
)

func (u *UnicodeTables) joiningType(cp rune) joiningType {
	return joiningType(u.joining.at(uint32(cp)))
}

// unicodeFlag bits from kbts_unicode_flags.
const (
	unicodeFlagModifierCombiningMark uint8 = 1 << 0
	unicodeFlagDefaultIgnorable      uint8 = 1 << 1
	unicodeFlagOpenBracket           uint8 = 1 << 2
	unicodeFlagCloseBracket          uint8 = 1 << 3
	unicodeFlagPartOfWord            uint8 = 1 << 4
	unicodeFlagDecimalDigit          uint8 = 1 << 5
	unicodeFlagNonSpacingMark        uint8 = 1 << 6
	unicodeFlagMirrored              uint8 = unicodeFlagOpenBracket | unicodeFlagCloseBracket
)

// pageTable is a two-level page table over the Unicode codespace: pages maps
// a codepoint's high bits to a page of 1<<shift entries in data.
//
// The zero value answers zero for every codepoint, because maxCP of 0 makes the
// first comparison reject everything. A table a set leaves out therefore needs
// no nil check at its call site and costs no branch of its own.
type pageTable[T uint8 | uint16 | uint32] struct {
	pages    []uint8
	data     []T
	maxCP    uint32 // exclusive; at or above this the table answers zero
	highPage uint32 // data offset for codepoints past the page-index array
	shift    uint8  // 7 for 128-entry pages, 5 for mirroring
}

func (t *pageTable[T]) at(u uint32) T {
	if u >= t.maxCP {
		var zero T
		return zero
	}
	page := t.highPage
	if p := u >> t.shift; p < uint32(len(t.pages)) {
		page = uint32(t.pages[p]) << t.shift
	}
	return t.data[page|(u&(uint32(1)<<t.shift-1))]
}

// UnicodeTables is the character-property data the segmenter and shaper read.
// Restricting it to the codepoints a program actually renders is the difference
// between roughly 273 KB of tables and roughly 1.4 KB, which is what decides
// whether shaping fits on a microcontroller.
//
// Obtain one from [UnicodeFull] or [UnicodeASCII]. Those are functions, and
// must stay functions: a package-level variable holding a UnicodeTables forces
// a package init that references every table, and the linker then retains all
// of them even for a program that only ever asks for the smallest set. Adding
// one such variable anywhere in this package silently costs every user 273 KB.
//
// The zero value covers nothing and answers zero for every property.
type UnicodeTables struct {
	// MaxRune is the highest codepoint this set describes, inclusive.
	MaxRune rune

	grapheme pageTable[uint8]
	line     pageTable[uint8]
	word     pageTable[uint8]
	bidi     pageTable[uint8]
	flags    pageTable[uint8]
	joining  pageTable[uint8]
	script   pageTable[uint16]
	mirror   pageTable[uint32]

	// scriptExt holds the multi-script extension lists script indexes into.
	// A set whose range reaches no codepoint with more than one script leaves
	// it nil, since nothing can index it.
	scriptExt []uint8
}

// Covered reports whether cp is within the range this set describes. Runes
// outside it read as zero for every property rather than as their true value,
// so a caller shaping arbitrary text must substitute before looking them up.
func (u *UnicodeTables) Covered(cp rune) bool {
	return uint32(cp) <= uint32(u.MaxRune)
}

// Lookup methods using two-level tables.

func (u *UnicodeTables) graphemeBreakClass(cp rune) graphemeBreakClass {
	return graphemeBreakClass(u.grapheme.at(uint32(cp)))
}

func (u *UnicodeTables) lineBreakClass(cp rune) lineBreakClass {
	return lineBreakClass(u.line.at(uint32(cp)))
}

func (u *UnicodeTables) wordBreakClass(cp rune) wordBreakClass {
	return wordBreakClass(u.word.at(uint32(cp)))
}

func (u *UnicodeTables) bidiClass(cp rune) bidiClass {
	return bidiClass(u.bidi.at(uint32(cp)))
}

func (u *UnicodeTables) unicodeFlags(cp rune) uint8 {
	return u.flags.at(uint32(cp))
}

func (u *UnicodeTables) scriptExtension(cp rune) uint16 {
	return u.script.at(uint32(cp))
}

func (u *UnicodeTables) mirrorCodepoint(cp rune) uint32 {
	return u.mirror.at(uint32(cp))
}

// scriptExtensionList returns the count scripts starting at off, or nil if the
// list cannot supply them. A table set restricted to a range with no
// multi-script codepoint carries no list at all, so the nil case is normal
// rather than a sign of damage.
func scriptExtensionList(ext []uint8, off, count uint32) []uint8 {
	if off > uint32(len(ext)) || count > uint32(len(ext))-off {
		return nil
	}
	return ext[off:]
}

func scriptExtensionCount(ext uint16) int {
	return int(ext & 0x1f)
}

func scriptExtensionOffset(ext uint16) int {
	return int(ext >> 5)
}
