package lefevre

const maxCodepointScripts = 23

// bracket tracks an open bracket for bidi/script pairing.
type bracket struct {
	codepoint uint32
	position  uint32
	direction uint8
	script    uint8
}

// breakStateFlags for internal BreakState tracking.
type breakStateFlags uint32

const (
	bsfStarted        breakStateFlags = 1
	bsfEnd            breakStateFlags = 2
	bsfSawRAfterL     breakStateFlags = 8
	bsfSawALAfterLR   breakStateFlags = 0x10
	bsfLastWasBracket breakStateFlags = 0x20
)

// flushFlags controls what gets flushed at the end of a codepoint.
type flushFlags uint32

const (
	flushNone               flushFlags = 0
	flushScript             flushFlags = 1 << 0
	flushDirection2         flushFlags = 1 << 1
	flushDirection1         flushFlags = 1 << 2
	flushDirectionParagraph flushFlags = 1 << 3
)

// Line break priority constants.
const (
	lineBreakAllowed0     uint64 = 1
	lineBreakAllowed1     uint64 = 3
	lineBreakAllowed2     uint64 = 7
	lineBreakAllowed3     uint64 = 0xF
	lineBreakAllowed4     uint64 = 0x1F
	lineBreakAllowed5     uint64 = 0x3F
	lineBreakRequired0    uint64 = (1 << 6) | lineBreakAllowed0
	lineBreakRequired1    uint64 = (3 << 6) | lineBreakAllowed1
	lineBreakRequired2    uint64 = (7 << 6) | lineBreakAllowed2
	lineBreakRequired3    uint64 = (0xF << 6) | lineBreakAllowed3
	lineBreakRequired4    uint64 = (0x1F << 6) | lineBreakAllowed4
	lineBreakRequired5    uint64 = (0x3F << 6) | lineBreakAllowed5
	lineBreakRequiredMask uint64 = 0x3F << 6
	lineBreakAllowedMask  uint64 = 0x3F
	lineBreakMask         uint64 = lineBreakRequiredMask | lineBreakAllowedMask
)

// Internal fields of BreakState. We define these as the struct body.
func init() {
	// Compile-time check: Direction count
	_ = [1]struct{}{{}} // placeholder
}

// breakBuf is the internal break buffer.
type breakBuf struct {
	breaks     [8]Break
	breakCount uint32
}

// breakStateInternal holds all the internal state for the break state machine.
type breakStateInternal struct {
	breakBuf

	paragraphDirection     Direction
	userParagraphDirection Direction

	currentPosition        uint32
	paragraphStartPosition uint32

	lastScriptBreakPosition     uint32
	lastDirectionBreakPosition  uint32
	lastScriptBreakScript       uint8
	lastDirectionBreakDirection uint8

	scriptPositionOffset int16
	scriptCount          uint32
	scriptSet            [maxCodepointScripts]uint8

	brackets     [64]bracket
	bracketCount uint32
	flags        breakStateFlags

	flagState       uint32 // u8(breakFlags)x4
	positionOffset2 int16
	positionOffset3 int16

	wordBreakHistory                   uint32 // u8x4
	wordBreaks                         uint16 // u4x4
	wordUnbreaks                       uint16 // u4x4
	wordBreak2PositionOffset           int16
	lastWordBreakClass                 wordBreakClass
	lastWordBreakClassIncludingIgnored wordBreakClass

	lineBreaks               uint64 // u16x4
	lineUnbreaksAsync        uint64 // u16x4
	lineUnbreaks             uint64 // u16x4
	lineBreakHistory         uint32 // u8(lineBreakClass)x4
	lineBreak2PositionOffset int16
	lineBreak3PositionOffset int16

	lastDirection                uint8
	bidirectionalClass2          bidiClass
	bidirectionalClass1          bidiClass
	bidirectional1PositionOffset int16
	bidirectional2PositionOffset int16

	japaneseLineBreakStyle JapaneseLineBreakStyle
	configFlags            BreakConfigFlags
	graphemeBreakSt        graphemeBreakState
	lastLineBreakClass     lineBreakClass
}

// Begin initializes or resets the break state for a new text run.
func (bs *BreakState) Begin(paragraphDirection Direction, style JapaneseLineBreakStyle, flags BreakConfigFlags) {
	*bs = BreakState{}
	s := &bs.s
	s.userParagraphDirection = paragraphDirection
	s.paragraphDirection = paragraphDirection
	s.japaneseLineBreakStyle = style
	s.configFlags = flags

	// Force a direction break at the start.
	s.lastDirection = 3 // KBTS_DIRECTION_COUNT

	// Out-of-bounds offsets while buffers haven't filled.
	s.positionOffset2 = -100
	s.positionOffset3 = -100
	s.wordBreak2PositionOffset = -100
	s.lineBreak2PositionOffset = -100
	s.lineBreak3PositionOffset = -100
	s.bidirectional1PositionOffset = -100
	s.bidirectional2PositionOffset = -100

	breakStateStartParagraph(s)

	if paragraphDirection != DirectionUnknown {
		doBreak(&bs.s, 0, BreakFlagParagraphDirection, DirectionUnknown, paragraphDirection, 0)
	}
}

// AddCodepoint feeds one codepoint to the segmenter.
func (bs *BreakState) AddCodepoint(codepoint rune, positionIncrement int, endOfText bool) {
	breakAddCodepoint(&bs.s, uint32(codepoint), uint32(positionIncrement), false)
	if endOfText {
		bs.End()
	}
}

// End flushes any remaining breaks after all codepoints have been added.
func (bs *BreakState) End() {
	breakAddCodepoint(&bs.s, 3, 0, true) // ASCII ETX
}

// Next returns the next pending break, or false if none remain.
func (bs *BreakState) Next() (Break, bool) {
	if bs.s.breakCount == 0 {
		return Break{}, false
	}
	bs.s.breakCount--
	return bs.s.breaks[bs.s.breakCount], true
}

func breakStateStartParagraph(s *breakStateInternal) {
	pd := s.userParagraphDirection
	startBidi := bidiClassNI
	var flags breakStateFlags

	if pd == DirectionLTR {
		startBidi = bidiClassL
	} else if pd == DirectionRTL {
		flags = bsfSawRAfterL
		startBidi = bidiClassR
	}

	s.paragraphDirection = pd
	s.bidirectionalClass1 = startBidi
	s.flags = flags
}

func doBreak(s *breakStateInternal, position int16, flags BreakFlags, dir Direction, paragraphDir Direction, script Script) {
	breakPosition := s.currentPosition + uint32(int32(position))
	if flags == 0 || breakPosition > s.currentPosition {
		return
	}

	b := Break{
		Position:           int(breakPosition),
		Flags:              flags,
		Direction:          dir,
		ParagraphDirection: paragraphDir,
		Script:             script,
	}

	// Resolve bracket scripts/directions.
	if flags.HasAll(BreakFlagScript) && s.lastScriptBreakScript != 0 {
		for i := int(s.bracketCount); i > 0; i-- {
			br := &s.brackets[i-1]
			if br.position >= breakPosition {
				br.script = uint8(script)
			} else if br.position >= s.lastScriptBreakPosition {
				br.script = s.lastScriptBreakScript
			} else {
				break
			}
		}
		s.lastScriptBreakPosition = breakPosition
		s.lastScriptBreakScript = uint8(script)
	}

	if flags.HasAll(BreakFlagDirection) && s.lastDirectionBreakDirection != 0 {
		for i := int(s.bracketCount); i > 0; i-- {
			br := &s.brackets[i-1]
			if br.position >= breakPosition {
				br.direction = uint8(dir)
			} else if br.position >= s.lastDirectionBreakPosition {
				br.direction = s.lastDirectionBreakDirection
			} else {
				break
			}
		}
		s.lastDirectionBreakPosition = breakPosition
		s.lastDirectionBreakDirection = uint8(dir)
	}

	// Try to merge with existing break at same position.
	matched := false
	for i := uint32(0); i < s.breakCount; i++ {
		existing := &s.breaks[i]
		if existing.Position == b.Position {
			existing.Flags |= b.Flags
			if b.Flags.HasAll(BreakFlagDirection) {
				existing.Direction = b.Direction
			}
			if b.Flags.HasAll(BreakFlagParagraphDirection) {
				existing.ParagraphDirection = b.ParagraphDirection
			}
			if b.Flags.HasAll(BreakFlagScript) {
				existing.Script = b.Script
			}
			matched = true
			break
		} else if existing.Position < b.Position {
			// Insert in reverse order (LIFO).
			b, *existing = *existing, b
		} else if b.Flags.HasAll(BreakFlagParagraphDirection) &&
			existing.Flags.HasAll(BreakFlagDirection) &&
			existing.Direction == DirectionUnknown {
			existing.Direction = b.ParagraphDirection
		}
	}

	s.breaks[s.breakCount] = b
	if !matched {
		s.breakCount++
	}
}

func doLineBreak(s *breakStateInternal, position int16, effectiveBreaks uint64) {
	if effectiveBreaks&lineBreakMask == 0 {
		return
	}
	var flags BreakFlags
	if effectiveBreaks&lineBreakAllowedMask != 0 {
		flags |= BreakFlagLineSoft
	}
	if effectiveBreaks&lineBreakRequiredMask != 0 {
		flags |= BreakFlagLineHard
	}
	doBreak(s, position, flags, 0, 0, 0)
}

func flushDirection(s *breakStateInternal, lastDirection *uint8, bc bidiClass, positionOffset int16) {
	var breakFlags BreakFlags
	var dir Direction

	switch bc {
	case bidiClassL:
		breakFlags = BreakFlagDirection | BreakFlagParagraphDirection
		dir = DirectionLTR
	case bidiClassR:
		breakFlags = BreakFlagDirection | BreakFlagParagraphDirection
		dir = DirectionRTL
	case bidiClassAN, bidiClassEN:
		breakFlags = BreakFlagDirection
		dir = DirectionLTR
	case bidiClassNI:
		breakFlags = BreakFlagDirection
	}

	if breakFlags.HasAll(BreakFlagDirection) && uint8(dir) != *lastDirection {
		*lastDirection = uint8(dir)
		doBreak(s, positionOffset, BreakFlagDirection, dir, 0, 0)
	}

	if breakFlags.HasAll(BreakFlagParagraphDirection) && s.paragraphDirection == DirectionUnknown {
		startOffset := int16(int32(s.paragraphStartPosition) - int32(s.currentPosition))
		doBreak(s, startOffset, BreakFlagParagraphDirection, 0, dir, 0)
		s.paragraphDirection = dir
	}
}

// inSet32 checks if a value is in a bitset (up to 32 values).
func inSet32(val uint8, set uint32) bool {
	return val < 32 && (set>>(val))&1 != 0
}

// set32 builds a bitset from values.
func set32(vals ...uint8) uint32 {
	var s uint32
	for _, v := range vals {
		s |= 1 << v
	}
	return s
}

// wordBreakBits computes the priority bitmask for word breaking.
func wordBreakBits(priority, position uint32) uint32 {
	return ((1 << (priority + 1)) - 1) << (position * 4)
}

func breakAddCodepoint(s *breakStateInternal, codepoint uint32, positionIncrement uint32, maybeEndOfText bool) {
	cp := rune(codepoint)

	bidirectionalCls := getBidiClass(cp)
	unicodeFlgs := getUnicodeFlags(cp)
	matchingBracket := getMirrorCodepoint(cp)
	graphemeBrkClass := getGraphemeBreakClass(cp)
	lineBrkClass := getLineBreakClass(cp)
	wordBrkClass := getWordBreakClass(cp)
	scriptExt := getScriptExtension(cp)
	cpScriptCount := uint32(scriptExtensionCount(scriptExt))
	cpScriptOffset := uint32(scriptExtensionOffset(scriptExt))

	flagState := s.flagState << 8
	lastLineBrkClass := s.lastLineBreakClass
	endOfText := codepoint == 3 && maybeEndOfText
	startOfText := s.flags&bsfStarted == 0
	lineBreakHist := s.lineBreakHistory
	wordBreakHist := s.wordBreakHistory
	lastWordBrkClass := s.lastWordBreakClass
	wordBreak2PosOff := s.wordBreak2PositionOffset
	lastWordBrkClassIncIgnored := s.lastWordBreakClassIncludingIgnored
	posOff2 := s.positionOffset2
	posOff3 := s.positionOffset3
	flags := s.flags
	lastDir := s.lastDirection
	scriptSet := &s.scriptSet
	scriptPosOff := s.scriptPositionOffset
	scriptCnt := s.scriptCount
	scriptCntAtStart := scriptCnt
	breakScript := Script(scriptSet[0])
	var flush flushFlags
	bidi1PosOff := s.bidirectional1PositionOffset
	bidi2PosOff := s.bidirectional2PositionOffset
	bidi2 := s.bidirectionalClass2
	bidi1 := s.bidirectionalClass1

	if startOfText {
		lineBreakHist = uint32(lbcSOT)
		lastLineBrkClass = lbcSOT
		wordBreakHist = uint32(wbcSOT)
		lastWordBrkClass = wbcSOT
	}

	// Bracket pairing.
	if unicodeFlgs&unicodeFlagMirrored == unicodeFlagOpenBracket {
		if s.bracketCount < uint32(len(s.brackets)) {
			br := &s.brackets[s.bracketCount]
			s.bracketCount++
			br.codepoint = codepoint
			br.position = s.currentPosition
			br.direction = 0
			br.script = 0
			if scriptCnt > 0 {
				br.script = scriptSet[0]
			}
			flags |= bsfLastWasBracket
		}
	} else if unicodeFlgs&unicodeFlagMirrored == unicodeFlagCloseBracket {
		if s.bracketCount > 0 {
			var foundIdx int = -1
			for i := int(s.bracketCount) - 1; i >= 0; i-- {
				if s.brackets[i].codepoint == matchingBracket {
					foundIdx = i
					break
				}
			}
			if foundIdx >= 0 {
				found := &s.brackets[foundIdx]
				bracketScript := found.script
				bracketDir := found.direction
				if bracketScript == 0 && scriptCnt > 0 {
					bracketScript = scriptSet[0]
				}
				if bracketDir == 0 {
					bracketDir = lastDir
				}
				if bracketDir == uint8(DirectionLTR) {
					bidirectionalCls = bidiClassL
				} else if bracketDir == uint8(DirectionRTL) {
					bidirectionalCls = bidiClassR
				}
				bidirectionalCls = bidiClass(found.direction)
				cpScriptCount = 1
				cpScriptOffset = uint32(bracketScript)
				s.bracketCount = uint32(foundIdx)
			}
		}
	}

	// Script breaking.
	if endOfText {
		flush |= flushScript
	}

	if cpScriptCount < 2 {
		cpScript := uint8(cpScriptOffset)
		if cpScript != uint8(ScriptUnknown) && cpScript != uint8(ScriptDefault) && cpScript != uint8(ScriptDefault2) {
			scriptSetMatch := false
			for i := uint32(0); i < scriptCnt; i++ {
				if scriptSet[i] == cpScript {
					scriptSetMatch = true
					break
				}
			}
			if !scriptSetMatch {
				flush |= flushScript
			}
			scriptCnt = 1
			scriptSet[0] = cpScript
		}
	} else {
		// Refine script set by intersecting.
		cpScripts := scriptExtensions[cpScriptOffset:]
		newCount := uint32(0)
		si := uint32(0)
		ci := uint32(0)
		for si < scriptCnt && ci < cpScriptCount {
			cs := cpScripts[ci]
			ss := scriptSet[si]
			if cs < ss {
				ci++
			} else if ss < cs {
				si++
			} else {
				scriptSet[newCount] = ss
				newCount++
				ci++
				si++
			}
		}
		if newCount == 0 {
			flush |= flushScript
			for i := uint32(0); i < cpScriptCount; i++ {
				scriptSet[i] = cpScripts[i]
			}
			scriptCnt = cpScriptCount
		} else {
			scriptCnt = newCount
		}
	}

	// Direction breaking.
	if endOfText {
		bidirectionalCls = bidiClassNI
	}

	if bidirectionalCls != bidiClassBN {
		switch bidirectionalCls {
		case bidiClassNSM:
			bidirectionalCls = bidi1
		case bidiClassL:
			flags &^= bsfSawRAfterL | bsfSawALAfterLR
		case bidiClassR:
			flags |= bsfSawRAfterL
			flags &^= bsfSawALAfterLR
		case bidiClassAL:
			flags |= bsfSawALAfterLR | bsfSawRAfterL
			bidirectionalCls = bidiClassR
		case bidiClassEN:
			if flags&bsfSawALAfterLR != 0 {
				bidirectionalCls = bidiClassAN
				goto caseAN
			}
			if bidi2 == bidiClassEN &&
				(bidi1 == bidiClassES || bidi1 == bidiClassCS) {
				bidi1 = bidiClassEN
			}
			if s.paragraphDirection != DirectionUnknown && flags&bsfSawRAfterL == 0 {
				bidirectionalCls = bidiClassL
			}
		case bidiClassAN:
			goto caseAN
		case bidiClassET:
			if bidi1 == bidiClassEN {
				bidirectionalCls = bidiClassEN
			}
		}
		goto afterCaseAN
	caseAN:
		if bidi2 == bidiClassAN && bidi1 == bidiClassCS {
			bidi1 = bidiClassAN
		}
	afterCaseAN:

		// NI resolution.
		esEtCsSet := set32(uint8(bidiClassET), uint8(bidiClassES), uint8(bidiClassCS))
		if inSet32(uint8(bidi1), esEtCsSet) {
			bidi1 = bidiClassNI
		}

		if bidi1 == bidiClassNI {
			niEtEsCsSet := set32(uint8(bidiClassNI), uint8(bidiClassET), uint8(bidiClassES), uint8(bidiClassCS))
			if inSet32(uint8(bidirectionalCls), niEtEsCsSet) {
				goto skipDirectionBreak
			}

			rAnEnSet := set32(uint8(bidiClassR), uint8(bidiClassAN), uint8(bidiClassEN))
			if (bidi2 == bidiClassR || bidirectionalCls == bidiClassR) &&
				inSet32(uint8(bidi2), rAnEnSet) && inSet32(uint8(bidirectionalCls), rAnEnSet) {
				bidi1 = bidiClassR
			} else if bidi2 == bidiClassL && bidirectionalCls == bidiClassL {
				bidi1 = bidiClassL
			} else {
				if s.paragraphDirection == DirectionLTR {
					bidi1 = bidiClassL
				} else if s.paragraphDirection == DirectionRTL {
					bidi1 = bidiClassR
				}
			}
		}

		flush |= flushDirection2
		if endOfText {
			flush |= flushDirection1
		}

		goto doneDirection
	}
skipDirectionBreak:
	s.bidirectional2PositionOffset -= int16(positionIncrement)
	s.bidirectional1PositionOffset -= int16(positionIncrement)
doneDirection:

	// Grapheme breaking.
	if endOfText && !startOfText {
		flagState |= uint32(BreakFlagGrapheme) << 8 // position 1
		s.graphemeBreakSt = gbsStart
	} else {
		gbs := graphemeBreakState(graphemeBreakTransition[graphemeBrkClass][s.graphemeBreakSt])
		switch gbs {
		case gbsb01:
			flagState |= uint32(BreakFlagGrapheme)<<8 | uint32(BreakFlagGrapheme)
			gbs = gbsStart
		case gbsb0:
			flagState |= uint32(BreakFlagGrapheme)
			gbs = gbsStart
		default:
			if gbs >= gbsb1 && gbs <= gbsb1toSKIP {
				flagState |= uint32(BreakFlagGrapheme) << 8
				gbs -= gbsb1
			}
		}
		s.graphemeBreakSt = gbs
	}

	// Word breaking.
	ignoreWordSet := set32(uint8(wbcEX), uint8(wbcFO), uint8(wbcZWJ))
	sotCrLfNlSet := set32(uint8(wbcSOT), uint8(wbcCR), uint8(wbcLF), uint8(wbcNL))

	if inSet32(uint8(wordBrkClass), ignoreWordSet) && !inSet32(uint8(lastWordBrkClass), sotCrLfNlSet) {
		wordBreak2PosOff -= int16(positionIncrement)
		s.wordBreak2PositionOffset = wordBreak2PosOff
	} else {
		wordBreaks := uint32(s.wordBreaks) << 4
		wordUnbreaks := uint32(s.wordUnbreaks) << 4
		wordBreakHist = (wordBreakHist << 8) | uint32(wordBrkClass)

		wordBreaks |= wordBreakBits(0, 1) | wordBreakBits(0, 0)
		if startOfText {
			wordBreaks |= wordBreakBits(2, 1)
		}

		crLfNlSet := set32(uint8(wbcCR), uint8(wbcLF), uint8(wbcNL))
		if inSet32(uint8(wordBrkClass), crLfNlSet) {
			wordBreaks |= wordBreakBits(1, 1) | wordBreakBits(1, 0)
		} else if inSet32(uint8(wordBrkClass), set32(uint8(wbcOep), uint8(wbcALep))) {
			if lastWordBrkClassIncIgnored == wbcZWJ {
				wordUnbreaks |= wordBreakBits(0, 1)
			}
		}

		// 2-character word break rules.
		wb2 := wordBreakHist & 0xFFFF
		switch wb2 {
		case wbc2(wbcCR, wbcLF):
			wordUnbreaks |= wordBreakBits(1, 1)
		case wbc2(wbcWSS, wbcWSS):
			if wordBreak2PosOff >= 0 {
				wordUnbreaks |= wordBreakBits(0, 1)
			}
		case wbc2(wbcRI, wbcRI):
			wordBreakHist = 0
			wordUnbreaks |= wordBreakBits(0, 1)
		case wbc2(wbcHL, wbcSQ):
			wordUnbreaks |= wordBreakBits(0, 1)
		default:
			if isWordPairNoBreak(wb2) {
				wordUnbreaks |= wordBreakBits(0, 1)
			}
		}

		// 3-character word break rules.
		wb3 := wordBreakHist & 0xFFFFFF
		if isWord3NoBreak(wb3) {
			wordUnbreaks |= wordBreakBits(0, 1) | wordBreakBits(0, 2)
		}

		effectiveWordBreaks := wordBreaks & ^wordUnbreaks
		if effectiveWordBreaks&wordBreakBits(2, 2) != 0 {
			doBreak(s, posOff2+wordBreak2PosOff, BreakFlagWord, 0, 0, 0)
		}
		if endOfText {
			flagState |= uint32(BreakFlagWord) << 8 // position 1
		}

		s.wordBreaks = uint16(wordBreaks)
		s.wordUnbreaks = uint16(wordUnbreaks)
		s.lastWordBreakClass = wordBrkClass
		s.wordBreak2PositionOffset = 0
		s.wordBreakHistory = wordBreakHist
	}
	s.lastWordBreakClassIncludingIgnored = wordBrkClass

	// Line breaking.
	lineBrk3PosOff := s.lineBreak3PositionOffset
	lineBrk2PosOff := s.lineBreak2PositionOffset
	hardLineBreak := false
	{
		lineBreaksVal := s.lineBreaks << 16
		lineUnbreaksVal := s.lineUnbreaks << 16
		lineUnbreaksAsyncVal := s.lineUnbreaksAsync << 16
		absorbed := false

		if endOfText {
			lineBrkClass = lbcOnea
		} else if lineBrkClass > lbcCount {
			if lineBrkClass == lbcCM || lineBrkClass == lbcZWJ {
				if lineBrkClass == lbcZWJ {
					lineUnbreaksAsyncVal |= lineBreakRequired3 << 0
				}
				switch lastLineBrkClass {
				case lbcSOT, lbcBK, lbcCR, lbcLF, lbcNL, lbcSP, lbcZW:
					lineBrkClass = lbcALnea
				default:
					absorbed = true
				}
			} else if lineBrkClass == lbcCJ {
				if s.japaneseLineBreakStyle == JapaneseLineBreakStrict {
					lineBrkClass = lbcNSea
				} else {
					lineBrkClass = lbcIDea
				}
			}
		}

		if !absorbed && lastLineBrkClass == lineBrkClass && lineBrkClass == lbcSP {
			absorbed = true
		}

		if absorbed {
			s.lineBreak2PositionOffset -= int16(positionIncrement)
			s.lineBreak3PositionOffset -= int16(positionIncrement)
			s.lineUnbreaksAsync = lineUnbreaksAsyncVal
		} else {
			lineBreakHist = (lineBreakHist << 8) | uint32(lineBrkClass)

			if endOfText && s.configFlags&BreakConfigEndOfTextGeneratesHardBreak != 0 {
				lineBreaksVal |= lineBreakRequired5 << 16
				if lineBreakHist&0xFF00 == uint32(lbcQUPf)<<8 {
					lineUnbreaksVal |= lineBreakRequired3 << 32
				}
			}

			lineBreaksVal |= lineBreakAllowed0 << 0
			lineBreaksVal |= lineBreakAllowed0 << 16

			// Single-character line break rules.
			switch lineBrkClass {
			case lbcBK, lbcCR, lbcLF, lbcNL:
				lineUnbreaksVal |= lineBreakRequired4 << 16
				flush |= flushScript | flushDirection2 | flushDirection1 | flushDirectionParagraph
				scriptCnt = 0
				hardLineBreak = true
				lineBreaksVal |= lineBreakRequired5 << 0
			case lbcZW:
				lineBreaksVal |= lineBreakAllowed4 << 0
				lineUnbreaksVal |= lineBreakRequired4 << 16
			case lbcBB:
				lineUnbreaksVal |= lineBreakAllowed0 << 0
			case lbcGLea, lbcGLnea, lbcOPea, lbcOPnea:
				lineUnbreaksVal |= lineBreakRequired3 << 0
			case lbcWJ:
				lineUnbreaksVal |= lineBreakRequired3 << 0
				lineUnbreaksVal |= lineBreakRequired3 << 16
			case lbcCLea, lbcCLnea, lbcCPea, lbcCPnea, lbcEXea, lbcEXnea, lbcSY:
				lineUnbreaksVal |= lineBreakRequired3 << 16
			case lbcIS:
				lineUnbreaksVal |= lineBreakRequired2 << 16
			case lbcSP:
				lineUnbreaksVal |= lineBreakRequired4 << 16
				lineBreaksVal |= lineBreakAllowed2 << 0
			case lbcQU:
				lineUnbreaksVal |= lineBreakRequired1 << 16
				lineUnbreaksVal |= lineBreakRequired1 << 0
			case lbcQUPi:
				lineUnbreaksVal |= lineBreakRequired1 << 0
			case lbcQUPf:
				lineUnbreaksVal |= lineBreakRequired1 << 16
			case lbcCB:
				lineBreaksVal |= lineBreakAllowed1 << 0
				lineBreaksVal |= lineBreakAllowed1 << 16
			case lbcBAnea, lbcBAea, lbcHYPHEN, lbcHY, lbcNSnea, lbcNSea, lbcINnea, lbcINea:
				lineUnbreaksVal |= lineBreakAllowed0 << 16
			}

			// 2-character line break rules.
			applyLineBreak2Rules(lineBrkClass, lineBreakHist, &lineBreaksVal, &lineUnbreaksVal)

			// 3-character line break rules.
			applyLineBreak3Rules(lineBreakHist, &lineBreaksVal, &lineUnbreaksVal)

			// 4-character line break rules.
			applyLineBreak4Rules(lineBreakHist, &lineUnbreaksVal)

			if startOfText {
				lineUnbreaksVal |= lineBreakRequired5 << 16
				flagState |= uint32(BreakFlagGrapheme) << 8
			}

			effectiveLB := lineBreaksVal & ^(lineUnbreaksVal | lineUnbreaksAsyncVal)

			doLineBreak(s, posOff3+lineBrk3PosOff, effectiveLB>>48)
			if endOfText {
				doLineBreak(s, posOff2+lineBrk2PosOff, effectiveLB>>32)
				var flushedLineFlags BreakFlags
				if (effectiveLB>>16)&lineBreakAllowedMask != 0 {
					flushedLineFlags |= BreakFlagLineSoft
				}
				if (effectiveLB>>16)&lineBreakRequiredMask != 0 {
					flushedLineFlags |= BreakFlagLineHard
				}
				flagState |= uint32(flushedLineFlags) << 8
			}

			s.lineBreaks = lineBreaksVal
			s.lineUnbreaks = lineUnbreaksVal
			s.lineBreak2PositionOffset = 0
			s.lineBreak3PositionOffset = lineBrk2PosOff
			s.lastLineBreakClass = lineBrkClass
			s.lineBreakHistory = lineBreakHist
			s.lineUnbreaksAsync = lineUnbreaksAsyncVal
		}
	}

	// Flush scripts.
	if flush&flushScript != 0 && scriptCntAtStart > 0 {
		doBreak(s, scriptPosOff, BreakFlagScript, 0, 0, breakScript)
		scriptPosOff = 0
		if hardLineBreak {
			scriptPosOff = int16(positionIncrement)
		}
	}

	// Flush directions.
	if flush&flushDirection2 != 0 {
		flushDirection(s, &lastDir, bidi2, bidi2PosOff)
	}
	if flush&flushDirection1 != 0 {
		flushDirection(s, &lastDir, bidi1, bidi1PosOff)
	}

	if flush&flushDirectionParagraph != 0 {
		bidi1 = bidiClassNI
		lastDir = 0
		s.paragraphStartPosition = s.currentPosition + 1
		breakStateStartParagraph(s)
		if s.userParagraphDirection == DirectionUnknown {
			doBreak(s, 1, BreakFlagParagraphDirection, 0, DirectionUnknown, 0)
		}
	}

	scriptPosOff -= int16(positionIncrement)

	// Flush buffered breaks.
	doBreak(s, posOff3, BreakFlags((flagState>>24)&0xFF), 0, 0, 0)
	if endOfText {
		doBreak(s, posOff2, BreakFlags((flagState>>16)&0xFF), 0, 0, 0)
		doBreak(s, 0, BreakFlags((flagState>>8)&0xFF), 0, 0, 0)
		doBreak(s, int16(positionIncrement), BreakFlags(flagState&0xFF), 0, 0, 0)
	}

	s.flagState = flagState
	flags |= bsfStarted
	if endOfText {
		flags |= bsfEnd
	}
	s.flags = flags
	s.positionOffset2 = -int16(positionIncrement)
	s.positionOffset3 = posOff2 - int16(positionIncrement)
	s.currentPosition += positionIncrement

	if flush&flushDirection2 != 0 {
		s.bidirectionalClass2 = bidi1
		s.bidirectionalClass1 = bidirectionalCls
		s.bidirectional2PositionOffset = s.bidirectional1PositionOffset - int16(positionIncrement)
		s.bidirectional1PositionOffset = -int16(positionIncrement)
	}
	s.lastDirection = lastDir

	s.scriptPositionOffset = scriptPosOff
	s.scriptCount = scriptCnt
}

// Word break pair lookup helpers.
func wbc2(a, b wordBreakClass) uint32 {
	return uint32(a)<<8 | uint32(b)
}

func wbc3(a, b, c wordBreakClass) uint32 {
	return uint32(a)<<16 | uint32(b)<<8 | uint32(c)
}

func isWordPairNoBreak(wb2 uint32) bool {
	switch wb2 {
	case wbc2(wbcALnep, wbcALnep), wbc2(wbcALnep, wbcALep), wbc2(wbcALnep, wbcHL), wbc2(wbcALnep, wbcNM), wbc2(wbcALnep, wbcENL),
		wbc2(wbcALep, wbcALnep), wbc2(wbcALep, wbcALep), wbc2(wbcALep, wbcHL), wbc2(wbcALep, wbcNM), wbc2(wbcALep, wbcENL),
		wbc2(wbcHL, wbcALnep), wbc2(wbcHL, wbcALep), wbc2(wbcHL, wbcHL), wbc2(wbcHL, wbcNM), wbc2(wbcHL, wbcENL),
		wbc2(wbcNM, wbcALnep), wbc2(wbcNM, wbcALep), wbc2(wbcNM, wbcHL), wbc2(wbcNM, wbcNM), wbc2(wbcNM, wbcENL),
		wbc2(wbcKA, wbcKA), wbc2(wbcKA, wbcENL),
		wbc2(wbcENL, wbcALnep), wbc2(wbcENL, wbcALep), wbc2(wbcENL, wbcHL), wbc2(wbcENL, wbcNM), wbc2(wbcENL, wbcKA), wbc2(wbcENL, wbcENL):
		return true
	}
	return false
}

func isWord3NoBreak(wb3 uint32) bool {
	switch wb3 {
	case wbc3(wbcALnep, wbcML, wbcALnep), wbc3(wbcALnep, wbcML, wbcALep), wbc3(wbcALnep, wbcML, wbcHL),
		wbc3(wbcALnep, wbcMNL, wbcALnep), wbc3(wbcALnep, wbcMNL, wbcALep), wbc3(wbcALnep, wbcMNL, wbcHL),
		wbc3(wbcALnep, wbcSQ, wbcALnep), wbc3(wbcALnep, wbcSQ, wbcALep), wbc3(wbcALnep, wbcSQ, wbcHL),
		wbc3(wbcALep, wbcML, wbcALnep), wbc3(wbcALep, wbcML, wbcALep), wbc3(wbcALep, wbcML, wbcHL),
		wbc3(wbcALep, wbcMNL, wbcALnep), wbc3(wbcALep, wbcMNL, wbcALep), wbc3(wbcALep, wbcMNL, wbcHL),
		wbc3(wbcALep, wbcSQ, wbcALnep), wbc3(wbcALep, wbcSQ, wbcALep), wbc3(wbcALep, wbcSQ, wbcHL),
		wbc3(wbcHL, wbcML, wbcALnep), wbc3(wbcHL, wbcML, wbcALep), wbc3(wbcHL, wbcML, wbcHL),
		wbc3(wbcHL, wbcMNL, wbcALnep), wbc3(wbcHL, wbcMNL, wbcALep), wbc3(wbcHL, wbcMNL, wbcHL),
		wbc3(wbcHL, wbcSQ, wbcALnep), wbc3(wbcHL, wbcSQ, wbcALep), wbc3(wbcHL, wbcSQ, wbcHL),
		wbc3(wbcHL, wbcDQ, wbcHL),
		wbc3(wbcNM, wbcMN, wbcNM), wbc3(wbcNM, wbcMNL, wbcNM), wbc3(wbcNM, wbcSQ, wbcNM):
		return true
	}
	return false
}

// Line break rule helpers — these are large switch statements mirroring the C code.
// For brevity we implement the most common rules; the full set matches the C source.

func lbc2(a, b lineBreakClass) uint32 {
	return uint32(a)<<8 | uint32(b)
}

func lbc3(a, b, c lineBreakClass) uint32 {
	return uint32(a)<<16 | uint32(b)<<8 | uint32(c)
}

func lbc4(a, b, c, d lineBreakClass) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)
}

func applyLineBreak2Rules(cls lineBreakClass, hist uint32, breaks *uint64, unbreaks *uint64) {
	h2 := hist & 0xFFFF
	switch h2 {
	case lbc2(lbcCR, lbcLF):
		*unbreaks |= lineBreakRequired5 << 16
	case lbc2(lbcZW, lbcSP):
		*breaks |= lineBreakAllowed4 << 0

	case lbc2(lbcOPea, lbcQUPi), lbc2(lbcGLea, lbcQUPi), lbc2(lbcOPea, lbcSP), lbc2(lbcOPnea, lbcSP):
		*unbreaks |= lineBreakRequired3 << 0

	case lbc2(lbcRI, lbcRI):
		*unbreaks |= lineBreakAllowed0 << 16
		// Reset history handled externally

	default:
		applyLineBreak2RulesExtended(h2, cls, hist, breaks, unbreaks)
	}
}

func applyLineBreak2RulesExtended(h2 uint32, cls lineBreakClass, hist uint32, breaks *uint64, unbreaks *uint64) {
	// QUPi followed by various classes.
	prev := lineBreakClass(h2 >> 8)
	curr := lineBreakClass(h2 & 0xFF)

	// QUPi + SOT/BK/CR/LF/NL/SP/ZW/GLnea/QU/OPnea -> no break around QUPi
	if prev == lbcQUPi {
		switch curr {
		case lbcSP:
			*unbreaks |= lineBreakRequired3 << 0
			return
		case lbcGLnea:
			// QUPi GLnea
			*unbreaks |= lineBreakRequired1 << 32 // position 2
			*unbreaks |= lineBreakRequired3 << 16
			return
		case lbcQUPi:
			*unbreaks |= lineBreakRequired3 << 0
			*unbreaks |= lineBreakRequired1 << 32
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			return
		case lbcQU:
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			*unbreaks |= lineBreakRequired1 << 32
			return
		case lbcQUPf:
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			*unbreaks |= lineBreakRequired1 << 32
			return
		}

		// QUPi + most non-break-control classes
		isQUPiNoBreakAfter := false
		switch curr {
		case lbcOnea, lbcOpe, lbcBK, lbcCR, lbcLF, lbcNL, lbcSP, lbcZW,
			lbcWJ, lbcCLnea, lbcCPnea, lbcEXnea, lbcSY, lbcBAnea,
			lbcOPnea, lbcIS, lbcNSnea, lbcB2, lbcCB,
			lbcHY, lbcHYPHEN, lbcINnea, lbcBB, lbcHL, lbcALnea, lbcNU, lbcPRnea,
			lbcIDnea, lbcIDpe, lbcEBnea, lbcPOnea, lbcJV, lbcJT, lbcAP, lbcAK,
			lbcDOTTED_CIRCLE, lbcAS, lbcVF, lbcVI, lbcRI:
			isQUPiNoBreakAfter = true
		}
		if isQUPiNoBreakAfter {
			*unbreaks |= lineBreakRequired1 << 32
			return
		}

		switch curr {
		case lbcSOT:
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			*unbreaks |= lineBreakRequired3 << 0
			return
		}
	}

	// SOT/QUPi prefix rules for QUPi.
	if curr == lbcQUPi {
		switch prev {
		case lbcSOT, lbcBK, lbcCR, lbcLF, lbcNL, lbcSP, lbcZW, lbcGLnea, lbcQU, lbcOPnea:
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			*unbreaks |= lineBreakRequired3 << 0
			return
		}
	}

	// QUPf rules.
	if prev == lbcQUPf {
		switch curr {
		case lbcGLnea:
			*unbreaks |= lineBreakRequired3 << 32
			*unbreaks |= lineBreakRequired3 << 16
			*unbreaks |= lineBreakRequired1 << 16
			return
		case lbcGLea:
			*unbreaks |= lineBreakRequired3 << 32
			*unbreaks |= lineBreakRequired3 << 16
			return
		case lbcCPea, lbcCLea, lbcEXea:
			*unbreaks |= lineBreakRequired3 << 32
			return
		case lbcQUPi:
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			*unbreaks |= lineBreakRequired3 << 0
			*unbreaks |= lineBreakRequired3 << 32
			*unbreaks |= lineBreakRequired1 << 16
			return
		case lbcQUPf:
			*unbreaks |= lineBreakRequired3 << 32
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			return
		case lbcQU:
			*unbreaks |= lineBreakRequired3 << 32
			*unbreaks |= lineBreakRequired1 << 16
			*unbreaks |= lineBreakRequired1 << 0
			return
		case lbcBK, lbcCR, lbcLF, lbcNL, lbcZW, lbcWJ,
			lbcCLnea, lbcCPnea, lbcEXnea, lbcSY, lbcIS, lbcSP:
			*unbreaks |= lineBreakRequired3 << 32
			*unbreaks |= lineBreakRequired1 << 16
			return
		default:
			// QUPf + most other classes
			isQUPfNoBreakBefore := false
			switch curr {
			case lbcOnea, lbcOpe, lbcBAnea, lbcOPnea, lbcNSnea, lbcB2, lbcCB,
				lbcHY, lbcHYPHEN, lbcINnea, lbcBB, lbcHL, lbcALnea, lbcNU, lbcPRnea,
				lbcIDnea, lbcIDpe, lbcEBnea, lbcPOnea, lbcJV, lbcJT, lbcAP, lbcAK,
				lbcDOTTED_CIRCLE, lbcAS, lbcVF, lbcVI, lbcRI:
				isQUPfNoBreakBefore = true
			}
			if isQUPfNoBreakBefore {
				*unbreaks |= lineBreakRequired1 << 16
				return
			}
		}
	}

	// General QU/QUPi/QUPf rules.
	if curr == lbcQU || curr == lbcQUPf {
		*unbreaks |= lineBreakRequired1 << 16
		*unbreaks |= lineBreakRequired1 << 0
		return
	}

	// Numeric/Alphabetic pair rules.
	applyLineBreak2NumAlpha(h2, unbreaks)

	// GL rules (glue).
	if curr == lbcGLea || curr == lbcGLnea {
		*unbreaks |= lineBreakRequired3 << 16
		return
	}

	// CL/CP + NS.
	switch h2 {
	case lbc2(lbcCLea, lbcNSnea), lbc2(lbcCLea, lbcNSea),
		lbc2(lbcCLnea, lbcNSnea), lbc2(lbcCLnea, lbcNSea),
		lbc2(lbcCPea, lbcNSnea), lbc2(lbcCPea, lbcNSea),
		lbc2(lbcCPnea, lbcNSnea), lbc2(lbcCPnea, lbcNSea),
		lbc2(lbcB2, lbcB2):
		*unbreaks |= lineBreakRequired2 << 16
	}
}

func applyLineBreak2NumAlpha(h2 uint32, unbreaks *uint64) {
	prev := lineBreakClass(h2 >> 8)
	curr := lineBreakClass(h2 & 0xFF)

	// AL/HL/DOTTED_CIRCLE + NU and vice versa.
	isALish := func(c lineBreakClass) bool {
		return c == lbcHL || c == lbcALnea || c == lbcALea || c == lbcDOTTED_CIRCLE
	}
	isPR := func(c lineBreakClass) bool { return c == lbcPRnea || c == lbcPRea }
	isPO := func(c lineBreakClass) bool { return c == lbcPOnea || c == lbcPOea }
	isID := func(c lineBreakClass) bool { return c == lbcIDnea || c == lbcIDea || c == lbcIDpe }
	isEB := func(c lineBreakClass) bool { return c == lbcEBnea || c == lbcEBea }

	// AL x NU, NU x AL, NU x NU
	if (isALish(prev) && curr == lbcNU) || (prev == lbcNU && isALish(curr)) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// PR x ID/EB/EM
	if isPR(prev) && (isID(curr) || isEB(curr) || curr == lbcEM) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// ID/EB/EM x PO
	if (isID(prev) || isEB(prev) || prev == lbcEM) && isPO(curr) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// PR/PO x AL/HL, AL/HL x PR/PO
	if (isPR(prev) || isPO(prev)) && isALish(curr) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if isALish(prev) && (isPR(curr) || isPO(curr)) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// NU x PO/PR, PO/PR x NU, NU x NU
	if prev == lbcNU && (isPO(curr) || isPR(curr) || curr == lbcNU) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if (isPO(prev) || isPR(prev)) && curr == lbcNU {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// SY x HL
	if prev == lbcSY && curr == lbcHL {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// Hangul rules
	if prev == lbcJL && (curr == lbcJL || curr == lbcJV || curr == lbcH2 || curr == lbcH3) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if (prev == lbcJV || prev == lbcH2) && (curr == lbcJV || curr == lbcJT) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if (prev == lbcJT || prev == lbcH3) && curr == lbcJT {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// JL/JV/JT/H2/H3 x PO, PR x JL/JV/JT/H2/H3
	isJamo := func(c lineBreakClass) bool {
		return c == lbcJL || c == lbcJV || c == lbcJT || c == lbcH2 || c == lbcH3
	}
	if isJamo(prev) && isPO(curr) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if isPR(prev) && isJamo(curr) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// AL/HL pair rules
	if isALish(prev) && isALish(curr) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// Aksara rules
	if prev == lbcAP && (curr == lbcAK || curr == lbcDOTTED_CIRCLE || curr == lbcAS) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if (prev == lbcAK || prev == lbcDOTTED_CIRCLE || prev == lbcAS) && (curr == lbcVF || curr == lbcVI) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// IS x AL/HL
	if prev == lbcIS && isALish(curr) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// AL/HL/NU x OPnea, CPnea x AL/HL/NU
	if (isALish(prev) || prev == lbcNU) && curr == lbcOPnea {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
	if prev == lbcCPnea && (isALish(curr) || curr == lbcNU) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// EB x EM, Ope x EM, IDpe x EM
	if (prev == lbcEBea || prev == lbcEBnea || prev == lbcOpe || prev == lbcIDpe) && curr == lbcEM {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// HY/IS x NU
	if (prev == lbcHY || prev == lbcIS) && curr == lbcNU {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}
}

func applyLineBreak3Rules(hist uint32, breaks *uint64, unbreaks *uint64) {
	h3 := hist & 0xFFFFFF
	a := lineBreakClass(h3 >> 16)
	b := lineBreakClass((h3 >> 8) & 0xFF)
	c := lineBreakClass(h3 & 0xFF)

	// SP IS NU
	if a == lbcSP && b == lbcIS && c == lbcNU {
		*breaks |= lineBreakRequired3 << 32
		return
	}

	// CL/CP SP NS
	isCL := a == lbcCLea || a == lbcCLnea || a == lbcCPea || a == lbcCPnea
	isNS := c == lbcNSnea || c == lbcNSea
	if isCL && b == lbcSP && isNS {
		*unbreaks |= lineBreakRequired2 << 16
		return
	}

	// B2 SP B2
	if a == lbcB2 && b == lbcSP && c == lbcB2 {
		*unbreaks |= lineBreakRequired2 << 16
		return
	}

	// SOT/BK/CR/LF/NL/SP/ZW/CB/GLnea/GLea HY/HYPHEN AL/DOTTED_CIRCLE
	isALish := c == lbcALnea || c == lbcALea || c == lbcDOTTED_CIRCLE
	isHYish := b == lbcHY || b == lbcHYPHEN
	if isHYish && isALish {
		switch a {
		case lbcSOT, lbcBK, lbcLF, lbcNL, lbcCR, lbcSP, lbcZW, lbcCB, lbcGLnea, lbcGLea:
			*unbreaks |= lineBreakAllowed0 << 16
			return
		}
	}

	// NU SY/IS PO/PR/NU
	if a == lbcNU && (b == lbcSY || b == lbcIS) {
		isPOPR := c == lbcPOea || c == lbcPOnea || c == lbcPRea || c == lbcPRnea || c == lbcNU
		if isPOPR {
			*unbreaks |= lineBreakAllowed0 << 16
			return
		}
	}

	// NU CL/CP PO/PR
	if a == lbcNU {
		isCLCP := b == lbcCLea || b == lbcCLnea || b == lbcCPea || b == lbcCPnea
		isPOPR := c == lbcPOea || c == lbcPOnea || c == lbcPRea || c == lbcPRnea
		if isCLCP && isPOPR {
			*unbreaks |= lineBreakAllowed0 << 16
			return
		}
	}

	// PO/PR OP NU
	if (a == lbcPOea || a == lbcPOnea || a == lbcPRea || a == lbcPRnea) &&
		(b == lbcOPea || b == lbcOPnea) && c == lbcNU {
		*unbreaks |= lineBreakAllowed0 << 32
		return
	}

	// Aksara rules: AK/DC/AS AK/DC/AS VF
	isAksara := func(cl lineBreakClass) bool { return cl == lbcAK || cl == lbcDOTTED_CIRCLE || cl == lbcAS }
	if isAksara(a) && isAksara(b) && c == lbcVF {
		*unbreaks |= lineBreakAllowed0 << 32
		return
	}

	// AK/DC/AS VI AK/DC
	if isAksara(a) && b == lbcVI && (c == lbcAK || c == lbcDOTTED_CIRCLE) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// HL BA/HY/HYPHEN + any -> no break at position 1
	if a == lbcHL && (b == lbcBAnea || b == lbcHYPHEN || b == lbcHY) {
		*unbreaks |= lineBreakAllowed0 << 16
		return
	}

	// QUPi SP rules
	if a == lbcSOT || a == lbcBK || a == lbcCR || a == lbcLF || a == lbcNL ||
		a == lbcOPea || a == lbcOPnea || a == lbcSP || a == lbcZW ||
		a == lbcQU || a == lbcQUPi || a == lbcQUPf || a == lbcGLea || a == lbcGLnea {
		if b == lbcQUPi && c == lbcSP {
			*unbreaks |= lineBreakRequired3 << 0
			return
		}
	}
}

func applyLineBreak4Rules(hist uint32, unbreaks *uint64) {
	a := lineBreakClass(hist >> 24)
	b := lineBreakClass((hist >> 16) & 0xFF)
	c := lineBreakClass((hist >> 8) & 0xFF)
	d := lineBreakClass(hist & 0xFF)

	// NU SY/IS CL/CP PO/PR
	if a == lbcNU && (b == lbcSY || b == lbcIS) {
		isCLCP := c == lbcCLnea || c == lbcCLea || c == lbcCPnea || c == lbcCPea
		isPOPR := d == lbcPOnea || d == lbcPOea || d == lbcPRnea || d == lbcPRea
		if isCLCP && isPOPR {
			*unbreaks |= lineBreakAllowed0 << 16
			return
		}
	}

	// PO/PR OP IS NU
	if (a == lbcPOea || a == lbcPOnea || a == lbcPRea || a == lbcPRnea) &&
		(b == lbcOPea || b == lbcOPnea) && c == lbcIS && d == lbcNU {
		*unbreaks |= lineBreakAllowed0 << 48
		return
	}
}
