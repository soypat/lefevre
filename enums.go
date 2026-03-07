package lefevre

// Script identifies a Unicode script (e.g., Latin, Arabic, Devanagari).
type Script uint8

const (
	ScriptUnknown Script = iota
	ScriptAdlam
	ScriptAhom
	ScriptAnatolianHieroglyphs
	ScriptArabic
	ScriptArmenian
	ScriptAvestan
	ScriptBalinese
	ScriptBamum
	ScriptBassaVah
	ScriptBatak
	ScriptBengali
	ScriptBhaiksuki
	ScriptBopomofo
	ScriptBrahmi
	ScriptBuginese
	ScriptBuhid
	ScriptCanadianSyllabics
	ScriptCarian
	ScriptCaucasianAlbanian
	ScriptChakma
	ScriptCham
	ScriptCherokee
	ScriptChorasmian
	ScriptCJKIdeographic
	ScriptCoptic
	ScriptCypriotSyllabary
	ScriptCyproMinoan
	ScriptCyrillic
	ScriptDefault
	ScriptDefault2
	ScriptDeseret
	ScriptDevanagari
	ScriptDivesAkuru
	ScriptDogra
	ScriptDuployan
	ScriptEgyptianHieroglyphs
	ScriptElbasan
	ScriptElymaic
	ScriptEthiopic
	ScriptGaray
	ScriptGeorgian
	ScriptGlagolitic
	ScriptGothic
	ScriptGrantha
	ScriptGreek
	ScriptGujarati
	ScriptGunjalaGondi
	ScriptGurmukhi
	ScriptGurungKhema
	ScriptHangul
	ScriptHanifiRohingya
	ScriptHanunoo
	ScriptHatran
	ScriptHebrew
	ScriptHiragana
	ScriptImperialAramaic
	ScriptInscriptionalPahlavi
	ScriptInscriptionalParthian
	ScriptJavanese
	ScriptKaithi
	ScriptKannada
	ScriptKatakana
	ScriptKawi
	ScriptKayahLi
	ScriptKharoshthi
	ScriptKhitanSmallScript
	ScriptKhmer
	ScriptKhojki
	ScriptKhudawadi
	ScriptKiratRai
	ScriptLao
	ScriptLatin
	ScriptLepcha
	ScriptLimbu
	ScriptLinearA
	ScriptLinearB
	ScriptLisu
	ScriptLycian
	ScriptLydian
	ScriptMahajani
	ScriptMakasar
	ScriptMalayalam
	ScriptMandaic
	ScriptManichaean
	ScriptMarchen
	ScriptMasaramGondi
	ScriptMedefaidrin
	ScriptMeeteiMayek
	ScriptMendeKikakui
	ScriptMeroiticCursive
	ScriptMeroiticHieroglyphs
	ScriptMiao
	ScriptModi
	ScriptMongolian
	ScriptMro
	ScriptMultani
	ScriptMyanmar
	ScriptNabataean
	ScriptNagMundari
	ScriptNandinagari
	ScriptNewa
	ScriptNewTaiLue
	ScriptNKo
	ScriptNushu
	ScriptNyiakengPuachueHmong
	ScriptOgham
	ScriptOlChiki
	ScriptOlOnal
	ScriptOldItalic
	ScriptOldHungarian
	ScriptOldNorthArabian
	ScriptOldPermic
	ScriptOldPersianCuneiform
	ScriptOldSogdian
	ScriptOldSouthArabian
	ScriptOldTurkic
	ScriptOldUyghur
	ScriptOdia
	ScriptOsage
	ScriptOsmanya
	ScriptPahawhHmong
	ScriptPalmyrene
	ScriptPauCinHau
	ScriptPhagsPa
	ScriptPhoenician
	ScriptPsalterPahlavi
	ScriptRejang
	ScriptRunic
	ScriptSamaritan
	ScriptSaurashtra
	ScriptSharada
	ScriptShavian
	ScriptSiddham
	ScriptSignWriting
	ScriptSogdian
	ScriptSinhala
	ScriptSoraSompeng
	ScriptSoyombo
	ScriptSumeroAkkadianCuneiform
	ScriptSundanese
	ScriptSunuwar
	ScriptSylotiNagri
	ScriptSyriac
	ScriptTagalog
	ScriptTagbanwa
	ScriptTaiLe
	ScriptTaiTham
	ScriptTaiViet
	ScriptTakri
	ScriptTamil
	ScriptTangsa
	ScriptTangut
	ScriptTelugu
	ScriptThaana
	ScriptThai
	ScriptTibetan
	ScriptTifinagh
	ScriptTirhuta
	ScriptTodhri
	ScriptToto
	ScriptTuluTigalari
	ScriptUgariticCuneiform
	ScriptVai
	ScriptVithkuqi
	ScriptWancho
	ScriptWarangCiti
	ScriptYezidi
	ScriptYi
	ScriptZanabazarSquare
	scriptCount
)

// String returns the four-character ISO 15924 script tag (e.g., "latn", "arab").
func (s Script) String() string {
	tag := s.Tag()
	if tag == ScriptTagUnknown {
		return "Unknown"
	}
	return string([]byte{byte(tag), byte(tag >> 8), byte(tag >> 16), byte(tag >> 24)})
}

// Direction returns the natural text direction for this script (LTR or RTL).
func (s Script) Direction() Direction {
	if s == ScriptArabic || s == ScriptHebrew {
		return DirectionRTL
	}
	return DirectionLTR
}

// IsComplex reports whether this script requires a complex shaping engine.
func (s Script) IsComplex() bool {
	if s >= scriptCount {
		return false
	}
	return scriptProps[s].Shaper != ShaperDefault
}

// Tag returns the ISO 15924 four-character tag for this script.
func (s Script) Tag() ScriptTag {
	if s >= scriptCount {
		return ScriptTagUnknown
	}
	return scriptProps[s].Tag
}

// Shaper returns the shaping engine required for this script.
func (s Script) Shaper() Shaper {
	if s >= scriptCount {
		return ShaperDefault
	}
	return scriptProps[s].Shaper
}

// ScriptTag is a four-character ISO 15924 script identifier stored as a little-endian uint32.
type ScriptTag uint32

const (
	ScriptTagUnknown                 ScriptTag = ' ' | ' '<<8 | ' '<<16 | ' '<<24
	ScriptTagAdlam                   ScriptTag = 'a' | 'd'<<8 | 'l'<<16 | 'm'<<24
	ScriptTagAhom                    ScriptTag = 'a' | 'h'<<8 | 'o'<<16 | 'm'<<24
	ScriptTagAnatolianHieroglyphs    ScriptTag = 'h' | 'l'<<8 | 'u'<<16 | 'w'<<24
	ScriptTagArabic                  ScriptTag = 'a' | 'r'<<8 | 'a'<<16 | 'b'<<24
	ScriptTagArmenian                ScriptTag = 'a' | 'r'<<8 | 'm'<<16 | 'n'<<24
	ScriptTagAvestan                 ScriptTag = 'a' | 'v'<<8 | 's'<<16 | 't'<<24
	ScriptTagBalinese                ScriptTag = 'b' | 'a'<<8 | 'l'<<16 | 'i'<<24
	ScriptTagBamum                   ScriptTag = 'b' | 'a'<<8 | 'm'<<16 | 'u'<<24
	ScriptTagBassaVah                ScriptTag = 'b' | 'a'<<8 | 's'<<16 | 's'<<24
	ScriptTagBatak                   ScriptTag = 'b' | 'a'<<8 | 't'<<16 | 'k'<<24
	ScriptTagBengali                 ScriptTag = 'b' | 'n'<<8 | 'g'<<16 | '2'<<24
	ScriptTagBhaiksuki               ScriptTag = 'b' | 'h'<<8 | 'k'<<16 | 's'<<24
	ScriptTagBopomofo                ScriptTag = 'b' | 'o'<<8 | 'p'<<16 | 'o'<<24
	ScriptTagBrahmi                  ScriptTag = 'b' | 'r'<<8 | 'a'<<16 | 'h'<<24
	ScriptTagBuginese                ScriptTag = 'b' | 'u'<<8 | 'g'<<16 | 'i'<<24
	ScriptTagBuhid                   ScriptTag = 'b' | 'u'<<8 | 'h'<<16 | 'd'<<24
	ScriptTagCanadianSyllabics       ScriptTag = 'c' | 'a'<<8 | 'n'<<16 | 's'<<24
	ScriptTagCarian                  ScriptTag = 'c' | 'a'<<8 | 'r'<<16 | 'i'<<24
	ScriptTagCaucasianAlbanian       ScriptTag = 'a' | 'g'<<8 | 'h'<<16 | 'b'<<24
	ScriptTagChakma                  ScriptTag = 'c' | 'a'<<8 | 'k'<<16 | 'm'<<24
	ScriptTagCham                    ScriptTag = 'c' | 'h'<<8 | 'a'<<16 | 'm'<<24
	ScriptTagCherokee                ScriptTag = 'c' | 'h'<<8 | 'e'<<16 | 'r'<<24
	ScriptTagChorasmian              ScriptTag = 'c' | 'h'<<8 | 'r'<<16 | 's'<<24
	ScriptTagCJKIdeographic          ScriptTag = 'h' | 'a'<<8 | 'n'<<16 | 'i'<<24
	ScriptTagCoptic                  ScriptTag = 'c' | 'o'<<8 | 'p'<<16 | 't'<<24
	ScriptTagCypriotSyllabary        ScriptTag = 'c' | 'p'<<8 | 'r'<<16 | 't'<<24
	ScriptTagCyproMinoan             ScriptTag = 'c' | 'p'<<8 | 'm'<<16 | 'n'<<24
	ScriptTagCyrillic                ScriptTag = 'c' | 'y'<<8 | 'r'<<16 | 'l'<<24
	ScriptTagDefault                 ScriptTag = 'D' | 'F'<<8 | 'L'<<16 | 'T'<<24
	ScriptTagDefault2                ScriptTag = 'D' | 'F'<<8 | 'L'<<16 | 'T'<<24
	ScriptTagDeseret                 ScriptTag = 'd' | 's'<<8 | 'r'<<16 | 't'<<24
	ScriptTagDevanagari              ScriptTag = 'd' | 'e'<<8 | 'v'<<16 | '2'<<24
	ScriptTagDivesAkuru              ScriptTag = 'd' | 'i'<<8 | 'a'<<16 | 'k'<<24
	ScriptTagDogra                   ScriptTag = 'd' | 'o'<<8 | 'g'<<16 | 'r'<<24
	ScriptTagDuployan                ScriptTag = 'd' | 'u'<<8 | 'p'<<16 | 'l'<<24
	ScriptTagEgyptianHieroglyphs     ScriptTag = 'e' | 'g'<<8 | 'y'<<16 | 'p'<<24
	ScriptTagElbasan                 ScriptTag = 'e' | 'l'<<8 | 'b'<<16 | 'a'<<24
	ScriptTagElymaic                 ScriptTag = 'e' | 'l'<<8 | 'y'<<16 | 'm'<<24
	ScriptTagEthiopic                ScriptTag = 'e' | 't'<<8 | 'h'<<16 | 'i'<<24
	ScriptTagGaray                   ScriptTag = 'g' | 'a'<<8 | 'r'<<16 | 'a'<<24
	ScriptTagGeorgian                ScriptTag = 'g' | 'e'<<8 | 'o'<<16 | 'r'<<24
	ScriptTagGlagolitic              ScriptTag = 'g' | 'l'<<8 | 'a'<<16 | 'g'<<24
	ScriptTagGothic                  ScriptTag = 'g' | 'o'<<8 | 't'<<16 | 'h'<<24
	ScriptTagGrantha                 ScriptTag = 'g' | 'r'<<8 | 'a'<<16 | 'n'<<24
	ScriptTagGreek                   ScriptTag = 'g' | 'r'<<8 | 'e'<<16 | 'k'<<24
	ScriptTagGujarati                ScriptTag = 'g' | 'j'<<8 | 'r'<<16 | '2'<<24
	ScriptTagGunjalaGondi            ScriptTag = 'g' | 'o'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagGurmukhi                ScriptTag = 'g' | 'u'<<8 | 'r'<<16 | '2'<<24
	ScriptTagGurungKhema             ScriptTag = 'g' | 'u'<<8 | 'k'<<16 | 'h'<<24
	ScriptTagHangul                  ScriptTag = 'h' | 'a'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagHanifiRohingya          ScriptTag = 'r' | 'o'<<8 | 'h'<<16 | 'g'<<24
	ScriptTagHanunoo                 ScriptTag = 'h' | 'a'<<8 | 'n'<<16 | 'o'<<24
	ScriptTagHatran                  ScriptTag = 'h' | 'a'<<8 | 't'<<16 | 'r'<<24
	ScriptTagHebrew                  ScriptTag = 'h' | 'e'<<8 | 'b'<<16 | 'r'<<24
	ScriptTagHiragana                ScriptTag = 'k' | 'a'<<8 | 'n'<<16 | 'a'<<24
	ScriptTagImperialAramaic         ScriptTag = 'a' | 'r'<<8 | 'm'<<16 | 'i'<<24
	ScriptTagInscriptionalPahlavi    ScriptTag = 'p' | 'h'<<8 | 'l'<<16 | 'i'<<24
	ScriptTagInscriptionalParthian   ScriptTag = 'p' | 'r'<<8 | 't'<<16 | 'i'<<24
	ScriptTagJavanese                ScriptTag = 'j' | 'a'<<8 | 'v'<<16 | 'a'<<24
	ScriptTagKaithi                  ScriptTag = 'k' | 't'<<8 | 'h'<<16 | 'i'<<24
	ScriptTagKannada                 ScriptTag = 'k' | 'n'<<8 | 'd'<<16 | '2'<<24
	ScriptTagKatakana                ScriptTag = 'k' | 'a'<<8 | 'n'<<16 | 'a'<<24
	ScriptTagKawi                    ScriptTag = 'k' | 'a'<<8 | 'w'<<16 | 'i'<<24
	ScriptTagKayahLi                 ScriptTag = 'k' | 'a'<<8 | 'l'<<16 | 'i'<<24
	ScriptTagKharoshthi              ScriptTag = 'k' | 'h'<<8 | 'a'<<16 | 'r'<<24
	ScriptTagKhitanSmallScript       ScriptTag = 'k' | 'i'<<8 | 't'<<16 | 's'<<24
	ScriptTagKhmer                   ScriptTag = 'k' | 'h'<<8 | 'm'<<16 | 'r'<<24
	ScriptTagKhojki                  ScriptTag = 'k' | 'h'<<8 | 'o'<<16 | 'j'<<24
	ScriptTagKhudawadi               ScriptTag = 's' | 'i'<<8 | 'n'<<16 | 'd'<<24
	ScriptTagKiratRai                ScriptTag = 'k' | 'r'<<8 | 'a'<<16 | 'i'<<24
	ScriptTagLao                     ScriptTag = 'l' | 'a'<<8 | 'o'<<16 | ' '<<24
	ScriptTagLatin                   ScriptTag = 'l' | 'a'<<8 | 't'<<16 | 'n'<<24
	ScriptTagLepcha                  ScriptTag = 'l' | 'e'<<8 | 'p'<<16 | 'c'<<24
	ScriptTagLimbu                   ScriptTag = 'l' | 'i'<<8 | 'm'<<16 | 'b'<<24
	ScriptTagLinearA                 ScriptTag = 'l' | 'i'<<8 | 'n'<<16 | 'a'<<24
	ScriptTagLinearB                 ScriptTag = 'l' | 'i'<<8 | 'n'<<16 | 'b'<<24
	ScriptTagLisu                    ScriptTag = 'l' | 'i'<<8 | 's'<<16 | 'u'<<24
	ScriptTagLycian                  ScriptTag = 'l' | 'y'<<8 | 'c'<<16 | 'i'<<24
	ScriptTagLydian                  ScriptTag = 'l' | 'y'<<8 | 'd'<<16 | 'i'<<24
	ScriptTagMahajani                ScriptTag = 'm' | 'a'<<8 | 'h'<<16 | 'j'<<24
	ScriptTagMakasar                 ScriptTag = 'm' | 'a'<<8 | 'k'<<16 | 'a'<<24
	ScriptTagMalayalam               ScriptTag = 'm' | 'l'<<8 | 'm'<<16 | '2'<<24
	ScriptTagMandaic                 ScriptTag = 'm' | 'a'<<8 | 'n'<<16 | 'd'<<24
	ScriptTagManichaean              ScriptTag = 'm' | 'a'<<8 | 'n'<<16 | 'i'<<24
	ScriptTagMarchen                 ScriptTag = 'm' | 'a'<<8 | 'r'<<16 | 'c'<<24
	ScriptTagMasaramGondi            ScriptTag = 'g' | 'o'<<8 | 'n'<<16 | 'm'<<24
	ScriptTagMedefaidrin             ScriptTag = 'm' | 'e'<<8 | 'd'<<16 | 'f'<<24
	ScriptTagMeeteiMayek             ScriptTag = 'm' | 't'<<8 | 'e'<<16 | 'i'<<24
	ScriptTagMendeKikakui            ScriptTag = 'm' | 'e'<<8 | 'n'<<16 | 'd'<<24
	ScriptTagMeroiticCursive         ScriptTag = 'm' | 'e'<<8 | 'r'<<16 | 'c'<<24
	ScriptTagMeroiticHieroglyphs     ScriptTag = 'm' | 'e'<<8 | 'r'<<16 | 'o'<<24
	ScriptTagMiao                    ScriptTag = 'p' | 'l'<<8 | 'r'<<16 | 'd'<<24
	ScriptTagModi                    ScriptTag = 'm' | 'o'<<8 | 'd'<<16 | 'i'<<24
	ScriptTagMongolian               ScriptTag = 'm' | 'o'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagMro                     ScriptTag = 'm' | 'r'<<8 | 'o'<<16 | 'o'<<24
	ScriptTagMultani                 ScriptTag = 'm' | 'u'<<8 | 'l'<<16 | 't'<<24
	ScriptTagMyanmar                 ScriptTag = 'm' | 'y'<<8 | 'm'<<16 | '2'<<24
	ScriptTagNabataean               ScriptTag = 'n' | 'b'<<8 | 'a'<<16 | 't'<<24
	ScriptTagNagMundari              ScriptTag = 'n' | 'a'<<8 | 'g'<<16 | 'm'<<24
	ScriptTagNandinagari             ScriptTag = 'n' | 'a'<<8 | 'n'<<16 | 'd'<<24
	ScriptTagNewa                    ScriptTag = 'n' | 'e'<<8 | 'w'<<16 | 'a'<<24
	ScriptTagNewTaiLue               ScriptTag = 't' | 'a'<<8 | 'l'<<16 | 'u'<<24
	ScriptTagNKo                     ScriptTag = 'n' | 'k'<<8 | 'o'<<16 | ' '<<24
	ScriptTagNushu                   ScriptTag = 'n' | 's'<<8 | 'h'<<16 | 'u'<<24
	ScriptTagNyiakengPuachueHmong    ScriptTag = 'h' | 'm'<<8 | 'n'<<16 | 'p'<<24
	ScriptTagOgham                   ScriptTag = 'o' | 'g'<<8 | 'a'<<16 | 'm'<<24
	ScriptTagOlChiki                 ScriptTag = 'o' | 'l'<<8 | 'c'<<16 | 'k'<<24
	ScriptTagOlOnal                  ScriptTag = 'o' | 'n'<<8 | 'a'<<16 | 'o'<<24
	ScriptTagOldItalic               ScriptTag = 'i' | 't'<<8 | 'a'<<16 | 'l'<<24
	ScriptTagOldHungarian            ScriptTag = 'h' | 'u'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagOldNorthArabian         ScriptTag = 'n' | 'a'<<8 | 'r'<<16 | 'b'<<24
	ScriptTagOldPermic               ScriptTag = 'p' | 'e'<<8 | 'r'<<16 | 'm'<<24
	ScriptTagOldPersianCuneiform     ScriptTag = 'x' | 'p'<<8 | 'e'<<16 | 'o'<<24
	ScriptTagOldSogdian              ScriptTag = 's' | 'o'<<8 | 'g'<<16 | 'o'<<24
	ScriptTagOldSouthArabian         ScriptTag = 's' | 'a'<<8 | 'r'<<16 | 'b'<<24
	ScriptTagOldTurkic               ScriptTag = 'o' | 'r'<<8 | 'k'<<16 | 'h'<<24
	ScriptTagOldUyghur               ScriptTag = 'o' | 'u'<<8 | 'g'<<16 | 'r'<<24
	ScriptTagOdia                    ScriptTag = 'o' | 'r'<<8 | 'y'<<16 | '2'<<24
	ScriptTagOsage                   ScriptTag = 'o' | 's'<<8 | 'g'<<16 | 'e'<<24
	ScriptTagOsmanya                 ScriptTag = 'o' | 's'<<8 | 'm'<<16 | 'a'<<24
	ScriptTagPahawhHmong             ScriptTag = 'h' | 'm'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagPalmyrene               ScriptTag = 'p' | 'a'<<8 | 'l'<<16 | 'm'<<24
	ScriptTagPauCinHau               ScriptTag = 'p' | 'a'<<8 | 'u'<<16 | 'c'<<24
	ScriptTagPhagsPa                 ScriptTag = 'p' | 'h'<<8 | 'a'<<16 | 'g'<<24
	ScriptTagPhoenician              ScriptTag = 'p' | 'h'<<8 | 'n'<<16 | 'x'<<24
	ScriptTagPsalterPahlavi          ScriptTag = 'p' | 'h'<<8 | 'l'<<16 | 'p'<<24
	ScriptTagRejang                  ScriptTag = 'r' | 'j'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagRunic                   ScriptTag = 'r' | 'u'<<8 | 'n'<<16 | 'r'<<24
	ScriptTagSamaritan               ScriptTag = 's' | 'a'<<8 | 'm'<<16 | 'r'<<24
	ScriptTagSaurashtra              ScriptTag = 's' | 'a'<<8 | 'u'<<16 | 'r'<<24
	ScriptTagSharada                 ScriptTag = 's' | 'h'<<8 | 'r'<<16 | 'd'<<24
	ScriptTagShavian                 ScriptTag = 's' | 'h'<<8 | 'a'<<16 | 'w'<<24
	ScriptTagSiddham                 ScriptTag = 's' | 'i'<<8 | 'd'<<16 | 'd'<<24
	ScriptTagSignWriting             ScriptTag = 's' | 'g'<<8 | 'n'<<16 | 'w'<<24
	ScriptTagSogdian                 ScriptTag = 's' | 'o'<<8 | 'g'<<16 | 'd'<<24
	ScriptTagSinhala                 ScriptTag = 's' | 'i'<<8 | 'n'<<16 | 'h'<<24
	ScriptTagSoraSompeng             ScriptTag = 's' | 'o'<<8 | 'r'<<16 | 'a'<<24
	ScriptTagSoyombo                 ScriptTag = 's' | 'o'<<8 | 'y'<<16 | 'o'<<24
	ScriptTagSumeroAkkadianCuneiform ScriptTag = 'x' | 's'<<8 | 'u'<<16 | 'x'<<24
	ScriptTagSundanese               ScriptTag = 's' | 'u'<<8 | 'n'<<16 | 'd'<<24
	ScriptTagSunuwar                 ScriptTag = 's' | 'u'<<8 | 'n'<<16 | 'u'<<24
	ScriptTagSylotiNagri             ScriptTag = 's' | 'y'<<8 | 'l'<<16 | 'o'<<24
	ScriptTagSyriac                  ScriptTag = 's' | 'y'<<8 | 'r'<<16 | 'c'<<24
	ScriptTagTagalog                 ScriptTag = 't' | 'g'<<8 | 'l'<<16 | 'g'<<24
	ScriptTagTagbanwa                ScriptTag = 't' | 'a'<<8 | 'g'<<16 | 'b'<<24
	ScriptTagTaiLe                   ScriptTag = 't' | 'a'<<8 | 'l'<<16 | 'e'<<24
	ScriptTagTaiTham                 ScriptTag = 'l' | 'a'<<8 | 'n'<<16 | 'a'<<24
	ScriptTagTaiViet                 ScriptTag = 't' | 'a'<<8 | 'v'<<16 | 't'<<24
	ScriptTagTakri                   ScriptTag = 't' | 'a'<<8 | 'k'<<16 | 'r'<<24
	ScriptTagTamil                   ScriptTag = 't' | 'm'<<8 | 'l'<<16 | '2'<<24
	ScriptTagTangsa                  ScriptTag = 't' | 'n'<<8 | 's'<<16 | 'a'<<24
	ScriptTagTangut                  ScriptTag = 't' | 'a'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagTelugu                  ScriptTag = 't' | 'e'<<8 | 'l'<<16 | '2'<<24
	ScriptTagThaana                  ScriptTag = 't' | 'h'<<8 | 'a'<<16 | 'a'<<24
	ScriptTagThai                    ScriptTag = 't' | 'h'<<8 | 'a'<<16 | 'i'<<24
	ScriptTagTibetan                 ScriptTag = 't' | 'i'<<8 | 'b'<<16 | 't'<<24
	ScriptTagTifinagh                ScriptTag = 't' | 'f'<<8 | 'n'<<16 | 'g'<<24
	ScriptTagTirhuta                 ScriptTag = 't' | 'i'<<8 | 'r'<<16 | 'h'<<24
	ScriptTagTodhri                  ScriptTag = 't' | 'o'<<8 | 'd'<<16 | 'r'<<24
	ScriptTagToto                    ScriptTag = 't' | 'o'<<8 | 't'<<16 | 'o'<<24
	ScriptTagTuluTigalari            ScriptTag = 't' | 'u'<<8 | 't'<<16 | 'g'<<24
	ScriptTagUgariticCuneiform       ScriptTag = 'u' | 'g'<<8 | 'a'<<16 | 'r'<<24
	ScriptTagVai                     ScriptTag = 'v' | 'a'<<8 | 'i'<<16 | ' '<<24
	ScriptTagVithkuqi                ScriptTag = 'v' | 'i'<<8 | 't'<<16 | 'h'<<24
	ScriptTagWancho                  ScriptTag = 'w' | 'c'<<8 | 'h'<<16 | 'o'<<24
	ScriptTagWarangCiti              ScriptTag = 'w' | 'a'<<8 | 'r'<<16 | 'a'<<24
	ScriptTagYezidi                  ScriptTag = 'y' | 'e'<<8 | 'z'<<16 | 'i'<<24
	ScriptTagYi                      ScriptTag = 'y' | 'i'<<8 | ' '<<16 | ' '<<24
	ScriptTagZanabazarSquare         ScriptTag = 'z' | 'a'<<8 | 'n'<<16 | 'b'<<24
)

type scriptProperties struct {
	Tag    ScriptTag
	Shaper Shaper
}

var scriptProps = [scriptCount]scriptProperties{
	{ScriptTagUnknown, ShaperDefault},
	{ScriptTagAdlam, ShaperUSE},
	{ScriptTagAhom, ShaperUSE},
	{ScriptTagAnatolianHieroglyphs, ShaperDefault},
	{ScriptTagArabic, ShaperArabic},
	{ScriptTagArmenian, ShaperDefault},
	{ScriptTagAvestan, ShaperUSE},
	{ScriptTagBalinese, ShaperUSE},
	{ScriptTagBamum, ShaperDefault},
	{ScriptTagBassaVah, ShaperUSE},
	{ScriptTagBatak, ShaperUSE},
	{ScriptTagBengali, ShaperIndic},
	{ScriptTagBhaiksuki, ShaperUSE},
	{ScriptTagBopomofo, ShaperDefault},
	{ScriptTagBrahmi, ShaperUSE},
	{ScriptTagBuginese, ShaperUSE},
	{ScriptTagBuhid, ShaperUSE},
	{ScriptTagCanadianSyllabics, ShaperDefault},
	{ScriptTagCarian, ShaperDefault},
	{ScriptTagCaucasianAlbanian, ShaperDefault},
	{ScriptTagChakma, ShaperUSE},
	{ScriptTagCham, ShaperUSE},
	{ScriptTagCherokee, ShaperDefault},
	{ScriptTagChorasmian, ShaperUSE},
	{ScriptTagCJKIdeographic, ShaperDefault},
	{ScriptTagCoptic, ShaperUSE},
	{ScriptTagCypriotSyllabary, ShaperDefault},
	{ScriptTagCyproMinoan, ShaperUSE},
	{ScriptTagCyrillic, ShaperDefault},
	{ScriptTagDefault, ShaperDefault},
	{ScriptTagDefault2, ShaperDefault},
	{ScriptTagDeseret, ShaperDefault},
	{ScriptTagDevanagari, ShaperIndic},
	{ScriptTagDivesAkuru, ShaperUSE},
	{ScriptTagDogra, ShaperUSE},
	{ScriptTagDuployan, ShaperUSE},
	{ScriptTagEgyptianHieroglyphs, ShaperUSE},
	{ScriptTagElbasan, ShaperDefault},
	{ScriptTagElymaic, ShaperUSE},
	{ScriptTagEthiopic, ShaperDefault},
	{ScriptTagGaray, ShaperUSE},
	{ScriptTagGeorgian, ShaperDefault},
	{ScriptTagGlagolitic, ShaperUSE},
	{ScriptTagGothic, ShaperDefault},
	{ScriptTagGrantha, ShaperUSE},
	{ScriptTagGreek, ShaperDefault},
	{ScriptTagGujarati, ShaperIndic},
	{ScriptTagGunjalaGondi, ShaperUSE},
	{ScriptTagGurmukhi, ShaperIndic},
	{ScriptTagGurungKhema, ShaperUSE},
	{ScriptTagHangul, ShaperHangul},
	{ScriptTagHanifiRohingya, ShaperUSE},
	{ScriptTagHanunoo, ShaperUSE},
	{ScriptTagHatran, ShaperUSE},
	{ScriptTagHebrew, ShaperHebrew},
	{ScriptTagHiragana, ShaperDefault},
	{ScriptTagImperialAramaic, ShaperUSE},
	{ScriptTagInscriptionalPahlavi, ShaperDefault},
	{ScriptTagInscriptionalParthian, ShaperDefault},
	{ScriptTagJavanese, ShaperUSE},
	{ScriptTagKaithi, ShaperUSE},
	{ScriptTagKannada, ShaperIndic},
	{ScriptTagKatakana, ShaperDefault},
	{ScriptTagKawi, ShaperUSE},
	{ScriptTagKayahLi, ShaperUSE},
	{ScriptTagKharoshthi, ShaperUSE},
	{ScriptTagKhitanSmallScript, ShaperUSE},
	{ScriptTagKhmer, ShaperKhmer},
	{ScriptTagKhojki, ShaperUSE},
	{ScriptTagKhudawadi, ShaperUSE},
	{ScriptTagKiratRai, ShaperUSE},
	{ScriptTagLao, ShaperDefault},
	{ScriptTagLatin, ShaperDefault},
	{ScriptTagLepcha, ShaperUSE},
	{ScriptTagLimbu, ShaperUSE},
	{ScriptTagLinearA, ShaperDefault},
	{ScriptTagLinearB, ShaperDefault},
	{ScriptTagLisu, ShaperUSE},
	{ScriptTagLycian, ShaperUSE},
	{ScriptTagLydian, ShaperDefault},
	{ScriptTagMahajani, ShaperUSE},
	{ScriptTagMakasar, ShaperUSE},
	{ScriptTagMalayalam, ShaperIndic},
	{ScriptTagMandaic, ShaperUSE},
	{ScriptTagManichaean, ShaperUSE},
	{ScriptTagMarchen, ShaperUSE},
	{ScriptTagMasaramGondi, ShaperUSE},
	{ScriptTagMedefaidrin, ShaperUSE},
	{ScriptTagMeeteiMayek, ShaperUSE},
	{ScriptTagMendeKikakui, ShaperUSE},
	{ScriptTagMeroiticCursive, ShaperUSE},
	{ScriptTagMeroiticHieroglyphs, ShaperUSE},
	{ScriptTagMiao, ShaperUSE},
	{ScriptTagModi, ShaperUSE},
	{ScriptTagMongolian, ShaperUSE},
	{ScriptTagMro, ShaperDefault},
	{ScriptTagMultani, ShaperUSE},
	{ScriptTagMyanmar, ShaperMyanmar},
	{ScriptTagNabataean, ShaperDefault},
	{ScriptTagNagMundari, ShaperUSE},
	{ScriptTagNandinagari, ShaperUSE},
	{ScriptTagNewa, ShaperUSE},
	{ScriptTagNewTaiLue, ShaperUSE},
	{ScriptTagNKo, ShaperUSE},
	{ScriptTagNushu, ShaperDefault},
	{ScriptTagNyiakengPuachueHmong, ShaperUSE},
	{ScriptTagOgham, ShaperDefault},
	{ScriptTagOlChiki, ShaperUSE},
	{ScriptTagOlOnal, ShaperUSE},
	{ScriptTagOldItalic, ShaperDefault},
	{ScriptTagOldHungarian, ShaperDefault},
	{ScriptTagOldNorthArabian, ShaperUSE},
	{ScriptTagOldPermic, ShaperDefault},
	{ScriptTagOldPersianCuneiform, ShaperUSE},
	{ScriptTagOldSogdian, ShaperUSE},
	{ScriptTagOldSouthArabian, ShaperDefault},
	{ScriptTagOldTurkic, ShaperDefault},
	{ScriptTagOldUyghur, ShaperUSE},
	{ScriptTagOdia, ShaperIndic},
	{ScriptTagOsage, ShaperUSE},
	{ScriptTagOsmanya, ShaperDefault},
	{ScriptTagPahawhHmong, ShaperUSE},
	{ScriptTagPalmyrene, ShaperDefault},
	{ScriptTagPauCinHau, ShaperDefault},
	{ScriptTagPhagsPa, ShaperUSE},
	{ScriptTagPhoenician, ShaperDefault},
	{ScriptTagPsalterPahlavi, ShaperUSE},
	{ScriptTagRejang, ShaperUSE},
	{ScriptTagRunic, ShaperDefault},
	{ScriptTagSamaritan, ShaperDefault},
	{ScriptTagSaurashtra, ShaperUSE},
	{ScriptTagSharada, ShaperUSE},
	{ScriptTagShavian, ShaperDefault},
	{ScriptTagSiddham, ShaperUSE},
	{ScriptTagSignWriting, ShaperUSE},
	{ScriptTagSogdian, ShaperUSE},
	{ScriptTagSinhala, ShaperUSE},
	{ScriptTagSoraSompeng, ShaperUSE},
	{ScriptTagSoyombo, ShaperUSE},
	{ScriptTagSumeroAkkadianCuneiform, ShaperUSE},
	{ScriptTagSundanese, ShaperUSE},
	{ScriptTagSunuwar, ShaperUSE},
	{ScriptTagSylotiNagri, ShaperUSE},
	{ScriptTagSyriac, ShaperArabic},
	{ScriptTagTagalog, ShaperUSE},
	{ScriptTagTagbanwa, ShaperUSE},
	{ScriptTagTaiLe, ShaperUSE},
	{ScriptTagTaiTham, ShaperUSE},
	{ScriptTagTaiViet, ShaperUSE},
	{ScriptTagTakri, ShaperUSE},
	{ScriptTagTamil, ShaperIndic},
	{ScriptTagTangsa, ShaperUSE},
	{ScriptTagTangut, ShaperUSE},
	{ScriptTagTelugu, ShaperIndic},
	{ScriptTagThaana, ShaperDefault},
	{ScriptTagThai, ShaperDefault},
	{ScriptTagTibetan, ShaperTibetan},
	{ScriptTagTifinagh, ShaperUSE},
	{ScriptTagTirhuta, ShaperUSE},
	{ScriptTagTodhri, ShaperUSE},
	{ScriptTagToto, ShaperUSE},
	{ScriptTagTuluTigalari, ShaperUSE},
	{ScriptTagUgariticCuneiform, ShaperDefault},
	{ScriptTagVai, ShaperDefault},
	{ScriptTagVithkuqi, ShaperUSE},
	{ScriptTagWancho, ShaperUSE},
	{ScriptTagWarangCiti, ShaperUSE},
	{ScriptTagYezidi, ShaperUSE},
	{ScriptTagYi, ShaperDefault},
	{ScriptTagZanabazarSquare, ShaperUSE},
}

// FeatureTag is an OpenType feature tag stored as a little-endian uint32 (e.g., "liga", "kern").
type FeatureTag uint32

const (
	FeatureTagUnregistered FeatureTag = 0
	FeatureTagIsol         FeatureTag = 'i' | 's'<<8 | 'o'<<16 | 'l'<<24
	FeatureTagFina         FeatureTag = 'f' | 'i'<<8 | 'n'<<16 | 'a'<<24
	FeatureTagFin2         FeatureTag = 'f' | 'i'<<8 | 'n'<<16 | '2'<<24
	FeatureTagFin3         FeatureTag = 'f' | 'i'<<8 | 'n'<<16 | '3'<<24
	FeatureTagMedi         FeatureTag = 'm' | 'e'<<8 | 'd'<<16 | 'i'<<24
	FeatureTagMed2         FeatureTag = 'm' | 'e'<<8 | 'd'<<16 | '2'<<24
	FeatureTagInit         FeatureTag = 'i' | 'n'<<8 | 'i'<<16 | 't'<<24
	FeatureTagLjmo         FeatureTag = 'l' | 'j'<<8 | 'm'<<16 | 'o'<<24
	FeatureTagVjmo         FeatureTag = 'v' | 'j'<<8 | 'm'<<16 | 'o'<<24
	FeatureTagTjmo         FeatureTag = 't' | 'j'<<8 | 'm'<<16 | 'o'<<24
	FeatureTagRphf         FeatureTag = 'r' | 'p'<<8 | 'h'<<16 | 'f'<<24
	FeatureTagBlwf         FeatureTag = 'b' | 'l'<<8 | 'w'<<16 | 'f'<<24
	FeatureTagHalf         FeatureTag = 'h' | 'a'<<8 | 'l'<<16 | 'f'<<24
	FeatureTagPstf         FeatureTag = 'p' | 's'<<8 | 't'<<16 | 'f'<<24
	FeatureTagAbvf         FeatureTag = 'a' | 'b'<<8 | 'v'<<16 | 'f'<<24
	FeatureTagPref         FeatureTag = 'p' | 'r'<<8 | 'e'<<16 | 'f'<<24
	FeatureTagNumr         FeatureTag = 'n' | 'u'<<8 | 'm'<<16 | 'r'<<24
	FeatureTagFrac         FeatureTag = 'f' | 'r'<<8 | 'a'<<16 | 'c'<<24
	FeatureTagDnom         FeatureTag = 'd' | 'n'<<8 | 'o'<<16 | 'm'<<24
	FeatureTagCfar         FeatureTag = 'c' | 'f'<<8 | 'a'<<16 | 'r'<<24
	FeatureTagAalt         FeatureTag = 'a' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagAbvm         FeatureTag = 'a' | 'b'<<8 | 'v'<<16 | 'm'<<24
	FeatureTagAbvs         FeatureTag = 'a' | 'b'<<8 | 'v'<<16 | 's'<<24
	FeatureTagAfrc         FeatureTag = 'a' | 'f'<<8 | 'r'<<16 | 'c'<<24
	FeatureTagAkhn         FeatureTag = 'a' | 'k'<<8 | 'h'<<16 | 'n'<<24
	FeatureTagApkn         FeatureTag = 'a' | 'p'<<8 | 'k'<<16 | 'n'<<24
	FeatureTagBlwm         FeatureTag = 'b' | 'l'<<8 | 'w'<<16 | 'm'<<24
	FeatureTagBlws         FeatureTag = 'b' | 'l'<<8 | 'w'<<16 | 's'<<24
	FeatureTagCalt         FeatureTag = 'c' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagCase         FeatureTag = 'c' | 'a'<<8 | 's'<<16 | 'e'<<24
	FeatureTagCcmp         FeatureTag = 'c' | 'c'<<8 | 'm'<<16 | 'p'<<24
	FeatureTagChws         FeatureTag = 'c' | 'h'<<8 | 'w'<<16 | 's'<<24
	FeatureTagCjct         FeatureTag = 'c' | 'j'<<8 | 'c'<<16 | 't'<<24
	FeatureTagClig         FeatureTag = 'c' | 'l'<<8 | 'i'<<16 | 'g'<<24
	FeatureTagCpct         FeatureTag = 'c' | 'p'<<8 | 'c'<<16 | 't'<<24
	FeatureTagCpsp         FeatureTag = 'c' | 'p'<<8 | 's'<<16 | 'p'<<24
	FeatureTagCswh         FeatureTag = 'c' | 's'<<8 | 'w'<<16 | 'h'<<24
	FeatureTagCurs         FeatureTag = 'c' | 'u'<<8 | 'r'<<16 | 's'<<24
	FeatureTagCv01         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '1'<<24
	FeatureTagCv02         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '2'<<24
	FeatureTagCv03         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '3'<<24
	FeatureTagCv04         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '4'<<24
	FeatureTagCv05         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '5'<<24
	FeatureTagCv06         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '6'<<24
	FeatureTagCv07         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '7'<<24
	FeatureTagCv08         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '8'<<24
	FeatureTagCv09         FeatureTag = 'c' | 'v'<<8 | '0'<<16 | '9'<<24
	FeatureTagCv10         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '0'<<24
	FeatureTagCv11         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '1'<<24
	FeatureTagCv12         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '2'<<24
	FeatureTagCv13         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '3'<<24
	FeatureTagCv14         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '4'<<24
	FeatureTagCv15         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '5'<<24
	FeatureTagCv16         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '6'<<24
	FeatureTagCv17         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '7'<<24
	FeatureTagCv18         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '8'<<24
	FeatureTagCv19         FeatureTag = 'c' | 'v'<<8 | '1'<<16 | '9'<<24
	FeatureTagCv20         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '0'<<24
	FeatureTagCv21         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '1'<<24
	FeatureTagCv22         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '2'<<24
	FeatureTagCv23         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '3'<<24
	FeatureTagCv24         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '4'<<24
	FeatureTagCv25         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '5'<<24
	FeatureTagCv26         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '6'<<24
	FeatureTagCv27         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '7'<<24
	FeatureTagCv28         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '8'<<24
	FeatureTagCv29         FeatureTag = 'c' | 'v'<<8 | '2'<<16 | '9'<<24
	FeatureTagCv30         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '0'<<24
	FeatureTagCv31         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '1'<<24
	FeatureTagCv32         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '2'<<24
	FeatureTagCv33         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '3'<<24
	FeatureTagCv34         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '4'<<24
	FeatureTagCv35         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '5'<<24
	FeatureTagCv36         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '6'<<24
	FeatureTagCv37         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '7'<<24
	FeatureTagCv38         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '8'<<24
	FeatureTagCv39         FeatureTag = 'c' | 'v'<<8 | '3'<<16 | '9'<<24
	FeatureTagCv40         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '0'<<24
	FeatureTagCv41         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '1'<<24
	FeatureTagCv42         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '2'<<24
	FeatureTagCv43         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '3'<<24
	FeatureTagCv44         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '4'<<24
	FeatureTagCv45         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '5'<<24
	FeatureTagCv46         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '6'<<24
	FeatureTagCv47         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '7'<<24
	FeatureTagCv48         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '8'<<24
	FeatureTagCv49         FeatureTag = 'c' | 'v'<<8 | '4'<<16 | '9'<<24
	FeatureTagCv50         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '0'<<24
	FeatureTagCv51         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '1'<<24
	FeatureTagCv52         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '2'<<24
	FeatureTagCv53         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '3'<<24
	FeatureTagCv54         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '4'<<24
	FeatureTagCv55         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '5'<<24
	FeatureTagCv56         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '6'<<24
	FeatureTagCv57         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '7'<<24
	FeatureTagCv58         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '8'<<24
	FeatureTagCv59         FeatureTag = 'c' | 'v'<<8 | '5'<<16 | '9'<<24
	FeatureTagCv60         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '0'<<24
	FeatureTagCv61         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '1'<<24
	FeatureTagCv62         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '2'<<24
	FeatureTagCv63         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '3'<<24
	FeatureTagCv64         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '4'<<24
	FeatureTagCv65         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '5'<<24
	FeatureTagCv66         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '6'<<24
	FeatureTagCv67         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '7'<<24
	FeatureTagCv68         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '8'<<24
	FeatureTagCv69         FeatureTag = 'c' | 'v'<<8 | '6'<<16 | '9'<<24
	FeatureTagCv70         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '0'<<24
	FeatureTagCv71         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '1'<<24
	FeatureTagCv72         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '2'<<24
	FeatureTagCv73         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '3'<<24
	FeatureTagCv74         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '4'<<24
	FeatureTagCv75         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '5'<<24
	FeatureTagCv76         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '6'<<24
	FeatureTagCv77         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '7'<<24
	FeatureTagCv78         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '8'<<24
	FeatureTagCv79         FeatureTag = 'c' | 'v'<<8 | '7'<<16 | '9'<<24
	FeatureTagCv80         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '0'<<24
	FeatureTagCv81         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '1'<<24
	FeatureTagCv82         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '2'<<24
	FeatureTagCv83         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '3'<<24
	FeatureTagCv84         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '4'<<24
	FeatureTagCv85         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '5'<<24
	FeatureTagCv86         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '6'<<24
	FeatureTagCv87         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '7'<<24
	FeatureTagCv88         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '8'<<24
	FeatureTagCv89         FeatureTag = 'c' | 'v'<<8 | '8'<<16 | '9'<<24
	FeatureTagCv90         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '0'<<24
	FeatureTagCv91         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '1'<<24
	FeatureTagCv92         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '2'<<24
	FeatureTagCv93         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '3'<<24
	FeatureTagCv94         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '4'<<24
	FeatureTagCv95         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '5'<<24
	FeatureTagCv96         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '6'<<24
	FeatureTagCv97         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '7'<<24
	FeatureTagCv98         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '8'<<24
	FeatureTagCv99         FeatureTag = 'c' | 'v'<<8 | '9'<<16 | '9'<<24
	FeatureTagC2pc         FeatureTag = 'c' | '2'<<8 | 'p'<<16 | 'c'<<24
	FeatureTagC2sc         FeatureTag = 'c' | '2'<<8 | 's'<<16 | 'c'<<24
	FeatureTagDist         FeatureTag = 'd' | 'i'<<8 | 's'<<16 | 't'<<24
	FeatureTagDlig         FeatureTag = 'd' | 'l'<<8 | 'i'<<16 | 'g'<<24
	FeatureTagDtls         FeatureTag = 'd' | 't'<<8 | 'l'<<16 | 's'<<24
	FeatureTagExpt         FeatureTag = 'e' | 'x'<<8 | 'p'<<16 | 't'<<24
	FeatureTagFalt         FeatureTag = 'f' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagFlac         FeatureTag = 'f' | 'l'<<8 | 'a'<<16 | 'c'<<24
	FeatureTagFwid         FeatureTag = 'f' | 'w'<<8 | 'i'<<16 | 'd'<<24
	FeatureTagHaln         FeatureTag = 'h' | 'a'<<8 | 'l'<<16 | 'n'<<24
	FeatureTagHalt         FeatureTag = 'h' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagHist         FeatureTag = 'h' | 'i'<<8 | 's'<<16 | 't'<<24
	FeatureTagHkna         FeatureTag = 'h' | 'k'<<8 | 'n'<<16 | 'a'<<24
	FeatureTagHlig         FeatureTag = 'h' | 'l'<<8 | 'i'<<16 | 'g'<<24
	FeatureTagHngl         FeatureTag = 'h' | 'n'<<8 | 'g'<<16 | 'l'<<24
	FeatureTagHojo         FeatureTag = 'h' | 'o'<<8 | 'j'<<16 | 'o'<<24
	FeatureTagHwid         FeatureTag = 'h' | 'w'<<8 | 'i'<<16 | 'd'<<24
	FeatureTagItal         FeatureTag = 'i' | 't'<<8 | 'a'<<16 | 'l'<<24
	FeatureTagJalt         FeatureTag = 'j' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagJp78         FeatureTag = 'j' | 'p'<<8 | '7'<<16 | '8'<<24
	FeatureTagJp83         FeatureTag = 'j' | 'p'<<8 | '8'<<16 | '3'<<24
	FeatureTagJp90         FeatureTag = 'j' | 'p'<<8 | '9'<<16 | '0'<<24
	FeatureTagJp04         FeatureTag = 'j' | 'p'<<8 | '0'<<16 | '4'<<24
	FeatureTagKern         FeatureTag = 'k' | 'e'<<8 | 'r'<<16 | 'n'<<24
	FeatureTagLfbd         FeatureTag = 'l' | 'f'<<8 | 'b'<<16 | 'd'<<24
	FeatureTagLiga         FeatureTag = 'l' | 'i'<<8 | 'g'<<16 | 'a'<<24
	FeatureTagLnum         FeatureTag = 'l' | 'n'<<8 | 'u'<<16 | 'm'<<24
	FeatureTagLocl         FeatureTag = 'l' | 'o'<<8 | 'c'<<16 | 'l'<<24
	FeatureTagLtra         FeatureTag = 'l' | 't'<<8 | 'r'<<16 | 'a'<<24
	FeatureTagLtrm         FeatureTag = 'l' | 't'<<8 | 'r'<<16 | 'm'<<24
	FeatureTagMark         FeatureTag = 'm' | 'a'<<8 | 'r'<<16 | 'k'<<24
	FeatureTagMgrk         FeatureTag = 'm' | 'g'<<8 | 'r'<<16 | 'k'<<24
	FeatureTagMkmk         FeatureTag = 'm' | 'k'<<8 | 'm'<<16 | 'k'<<24
	FeatureTagMset         FeatureTag = 'm' | 's'<<8 | 'e'<<16 | 't'<<24
	FeatureTagNalt         FeatureTag = 'n' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagNlck         FeatureTag = 'n' | 'l'<<8 | 'c'<<16 | 'k'<<24
	FeatureTagNukt         FeatureTag = 'n' | 'u'<<8 | 'k'<<16 | 't'<<24
	FeatureTagOnum         FeatureTag = 'o' | 'n'<<8 | 'u'<<16 | 'm'<<24
	FeatureTagOpbd         FeatureTag = 'o' | 'p'<<8 | 'b'<<16 | 'd'<<24
	FeatureTagOrdn         FeatureTag = 'o' | 'r'<<8 | 'd'<<16 | 'n'<<24
	FeatureTagOrnm         FeatureTag = 'o' | 'r'<<8 | 'n'<<16 | 'm'<<24
	FeatureTagPalt         FeatureTag = 'p' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagPcap         FeatureTag = 'p' | 'c'<<8 | 'a'<<16 | 'p'<<24
	FeatureTagPkna         FeatureTag = 'p' | 'k'<<8 | 'n'<<16 | 'a'<<24
	FeatureTagPnum         FeatureTag = 'p' | 'n'<<8 | 'u'<<16 | 'm'<<24
	FeatureTagPres         FeatureTag = 'p' | 'r'<<8 | 'e'<<16 | 's'<<24
	FeatureTagPsts         FeatureTag = 'p' | 's'<<8 | 't'<<16 | 's'<<24
	FeatureTagPwid         FeatureTag = 'p' | 'w'<<8 | 'i'<<16 | 'd'<<24
	FeatureTagQwid         FeatureTag = 'q' | 'w'<<8 | 'i'<<16 | 'd'<<24
	FeatureTagRand         FeatureTag = 'r' | 'a'<<8 | 'n'<<16 | 'd'<<24
	FeatureTagRclt         FeatureTag = 'r' | 'c'<<8 | 'l'<<16 | 't'<<24
	FeatureTagRkrf         FeatureTag = 'r' | 'k'<<8 | 'r'<<16 | 'f'<<24
	FeatureTagRlig         FeatureTag = 'r' | 'l'<<8 | 'i'<<16 | 'g'<<24
	FeatureTagRtbd         FeatureTag = 'r' | 't'<<8 | 'b'<<16 | 'd'<<24
	FeatureTagRtla         FeatureTag = 'r' | 't'<<8 | 'l'<<16 | 'a'<<24
	FeatureTagRtlm         FeatureTag = 'r' | 't'<<8 | 'l'<<16 | 'm'<<24
	FeatureTagRuby         FeatureTag = 'r' | 'u'<<8 | 'b'<<16 | 'y'<<24
	FeatureTagRvrn         FeatureTag = 'r' | 'v'<<8 | 'r'<<16 | 'n'<<24
	FeatureTagSalt         FeatureTag = 's' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagSinf         FeatureTag = 's' | 'i'<<8 | 'n'<<16 | 'f'<<24
	FeatureTagSize         FeatureTag = 's' | 'i'<<8 | 'z'<<16 | 'e'<<24
	FeatureTagSmcp         FeatureTag = 's' | 'm'<<8 | 'c'<<16 | 'p'<<24
	FeatureTagSmpl         FeatureTag = 's' | 'm'<<8 | 'p'<<16 | 'l'<<24
	FeatureTagSs01         FeatureTag = 's' | 's'<<8 | '0'<<16 | '1'<<24
	FeatureTagSs02         FeatureTag = 's' | 's'<<8 | '0'<<16 | '2'<<24
	FeatureTagSs03         FeatureTag = 's' | 's'<<8 | '0'<<16 | '3'<<24
	FeatureTagSs04         FeatureTag = 's' | 's'<<8 | '0'<<16 | '4'<<24
	FeatureTagSs05         FeatureTag = 's' | 's'<<8 | '0'<<16 | '5'<<24
	FeatureTagSs06         FeatureTag = 's' | 's'<<8 | '0'<<16 | '6'<<24
	FeatureTagSs07         FeatureTag = 's' | 's'<<8 | '0'<<16 | '7'<<24
	FeatureTagSs08         FeatureTag = 's' | 's'<<8 | '0'<<16 | '8'<<24
	FeatureTagSs09         FeatureTag = 's' | 's'<<8 | '0'<<16 | '9'<<24
	FeatureTagSs10         FeatureTag = 's' | 's'<<8 | '1'<<16 | '0'<<24
	FeatureTagSs11         FeatureTag = 's' | 's'<<8 | '1'<<16 | '1'<<24
	FeatureTagSs12         FeatureTag = 's' | 's'<<8 | '1'<<16 | '2'<<24
	FeatureTagSs13         FeatureTag = 's' | 's'<<8 | '1'<<16 | '3'<<24
	FeatureTagSs14         FeatureTag = 's' | 's'<<8 | '1'<<16 | '4'<<24
	FeatureTagSs15         FeatureTag = 's' | 's'<<8 | '1'<<16 | '5'<<24
	FeatureTagSs16         FeatureTag = 's' | 's'<<8 | '1'<<16 | '6'<<24
	FeatureTagSs17         FeatureTag = 's' | 's'<<8 | '1'<<16 | '7'<<24
	FeatureTagSs18         FeatureTag = 's' | 's'<<8 | '1'<<16 | '8'<<24
	FeatureTagSs19         FeatureTag = 's' | 's'<<8 | '1'<<16 | '9'<<24
	FeatureTagSs20         FeatureTag = 's' | 's'<<8 | '2'<<16 | '0'<<24
	FeatureTagSsty         FeatureTag = 's' | 's'<<8 | 't'<<16 | 'y'<<24
	FeatureTagStch         FeatureTag = 's' | 't'<<8 | 'c'<<16 | 'h'<<24
	FeatureTagSubs         FeatureTag = 's' | 'u'<<8 | 'b'<<16 | 's'<<24
	FeatureTagSups         FeatureTag = 's' | 'u'<<8 | 'p'<<16 | 's'<<24
	FeatureTagSwsh         FeatureTag = 's' | 'w'<<8 | 's'<<16 | 'h'<<24
	FeatureTagTest         FeatureTag = 't' | 'e'<<8 | 's'<<16 | 't'<<24
	FeatureTagTitl         FeatureTag = 't' | 'i'<<8 | 't'<<16 | 'l'<<24
	FeatureTagTnam         FeatureTag = 't' | 'n'<<8 | 'a'<<16 | 'm'<<24
	FeatureTagTnum         FeatureTag = 't' | 'n'<<8 | 'u'<<16 | 'm'<<24
	FeatureTagTrad         FeatureTag = 't' | 'r'<<8 | 'a'<<16 | 'd'<<24
	FeatureTagTwid         FeatureTag = 't' | 'w'<<8 | 'i'<<16 | 'd'<<24
	FeatureTagUnic         FeatureTag = 'u' | 'n'<<8 | 'i'<<16 | 'c'<<24
	FeatureTagValt         FeatureTag = 'v' | 'a'<<8 | 'l'<<16 | 't'<<24
	FeatureTagVapk         FeatureTag = 'v' | 'a'<<8 | 'p'<<16 | 'k'<<24
	FeatureTagVatu         FeatureTag = 'v' | 'a'<<8 | 't'<<16 | 'u'<<24
	FeatureTagVchw         FeatureTag = 'v' | 'c'<<8 | 'h'<<16 | 'w'<<24
	FeatureTagVert         FeatureTag = 'v' | 'e'<<8 | 'r'<<16 | 't'<<24
	FeatureTagVhal         FeatureTag = 'v' | 'h'<<8 | 'a'<<16 | 'l'<<24
	FeatureTagVkna         FeatureTag = 'v' | 'k'<<8 | 'n'<<16 | 'a'<<24
	FeatureTagVkrn         FeatureTag = 'v' | 'k'<<8 | 'r'<<16 | 'n'<<24
	FeatureTagVpal         FeatureTag = 'v' | 'p'<<8 | 'a'<<16 | 'l'<<24
	FeatureTagVrt2         FeatureTag = 'v' | 'r'<<8 | 't'<<16 | '2'<<24
	FeatureTagVrtr         FeatureTag = 'v' | 'r'<<8 | 't'<<16 | 'r'<<24
	FeatureTagZero         FeatureTag = 'z' | 'e'<<8 | 'r'<<16 | 'o'<<24
)
