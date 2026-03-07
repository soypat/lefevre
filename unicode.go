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

func getJoiningType(cp rune) joiningType {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 918016 {
		page = uint32(joiningTypePageIndices[u/128]) * 128
	} else {
		page = 0
	}
	return joiningType(joiningTypeData[page|(u&127)])
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

// Lookup functions using two-level tables.

func getGraphemeBreakClass(cp rune) graphemeBreakClass {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 921600 {
		page = uint32(graphemeBreakClassPageIndices[u/128]) * 128
	} else {
		page = 256
	}
	return graphemeBreakClass(graphemeBreakClassData[page|(u&127)])
}

func getLineBreakClass(cp rune) lineBreakClass {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 918016 {
		page = uint32(lineBreakClassPageIndices[u/128]) * 128
	} else {
		page = 256
	}
	return lineBreakClass(lineBreakClassData[page|(u&127)])
}

func getWordBreakClass(cp rune) wordBreakClass {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 918016 {
		page = uint32(wordBreakClassPageIndices[u/128]) * 128
	} else {
		page = 7296
	}
	return wordBreakClass(wordBreakClassData[page|(u&127)])
}

func getBidiClass(cp rune) bidiClass {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 1048704 {
		page = uint32(bidiClassPageIndices[u/128]) * 128
	} else {
		page = 7296
	}
	return bidiClass(bidiClassData[page|(u&127)])
}

func getUnicodeFlags(cp rune) uint8 {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 1048704 {
		page = uint32(unicodeFlagsPageIndices[u/128]) * 128
	} else {
		page = 256
	}
	return unicodeFlagsData[page|(u&127)]
}

func getScriptExtension(cp rune) uint16 {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 918016 {
		page = uint32(scriptExtensionPageIndices[u/128]) * 128
	} else {
		page = 7808
	}
	return scriptExtensionData[page|(u&127)]
}

func getMirrorCodepoint(cp rune) uint32 {
	u := uint32(cp)
	if u >= 1114110 {
		return 0
	}
	var page uint32
	if u < 65408 {
		page = uint32(mirrorCodepointPageIndices[u/32]) * 32
	} else {
		page = 0
	}
	return mirrorCodepointData[page|(u&31)]
}

func scriptExtensionCount(ext uint16) int {
	return int(ext & 0x1f)
}

func scriptExtensionOffset(ext uint16) int {
	return int(ext >> 5)
}
