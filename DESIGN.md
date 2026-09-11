# lefevre design notes: table gating, alpha compositing, CFF

> Companion to [fyne.md](fyne.md). Every number here is measured, not estimated;
> reproduction commands are in §5.


## Context

[fyne.md](fyne.md) evaluated lefevre against go-text and surfaced three gaps worth closing on
their own merits, independent of whether the fyne swap happens. This document records what I
measured, corrects two claims in that evaluation, and plans the work.

---

## 0. Correction to fyne.md: the binary-size premise is wrong

fyne.md:93-99 reports lefevre costs 567 KB, and that its 275 KB of Unicode tables are
"unconditionally reachable, so they will not shrink". Both come from `go test -c` binaries,
where nothing is dead-code-eliminated. Measured against real linked binaries instead
(baseline: an empty Go program that reads a file, 1,675,426 B stripped):

| program | stripped | lefevre cost | lefevre data symbols |
|---|---|---|---|
| `LoadBytes` + `GlyphID` | 1,724,578 | +49,152 | 16 B |
| + `GlyphOutline` + `raster` | 1,736,866 | +61,440 | 16 B |
| + `ShapeSimple` | 2,085,026 | +409,600 | 275,115 B |

The tables are eliminated **completely** when shaping is unreferenced. The `.gopclntab`
multiplier fyne.md applies to lefevre is also inapplicable for the same reason.

**Consequence for the original question:** compile-time elimination of unused tables on desktop
is already solved by the linker and needs no API. fyne.md §2.4 and §4 should be revised — the
swap's cost side is overstated by roughly 500 KB.

**Consequence for the real question:** the case that is *not* solved is the ESP32 one. Shaping
ASCII costs the full 275 KB today. Of the 360 KB that `ShapeSimple` adds, **275 KB (76%) is
table data and only ~85 KB is code**. That ratio is what makes §1 worth doing.

### On "pre-indexing"

There is less here than fyne.md:121 implies. What it calls indexing 15 tables is
[font.go:97](font.go#L97) `tableIndex`, a fourcc→slot switch feeding a `[15]tableEntry` array of
offset/length pairs — 120 bytes and one switch. It is not where go-text's 342 KB goes; that is in
table *parsers*. There is no `func(dst []byte, runes []rune) error` abstraction to win there.

The genuinely useful batch API adjacent to that instinct is **cmap enumeration**, which
[FONT-LOWLEVELlocal.md](FONT-LOWLEVELlocal.md) already scopes as "the blocker for font fallback".
Both supported cmap formats are range-structured, so the allocation-free form is cheap:

```go
type CmapRange struct { First, Last rune; FirstGlyph uint16 }
func (f *Font) AppendCmapRanges(dst []CmapRange) []CmapRange
func (f *Font) AppendGlyphIDs(dst []uint16, runes []rune) []uint16
```

Not planned below — out of scope for this request — but it is the thing worth building if
fallback ever matters.

---

## 1. Configurable Unicode coverage (the ESP32 case)

### 1.1 The DCE rule, and why it is a trap

Every generated table in [unicode_tables.go](unicode_tables.go) is a two-level page table with
128-rune pages (32 for mirroring), and **ASCII is exactly page 0**. So an ASCII table set is
~129 bytes per table instead of ~35 KB.

Exposing those sets is subject to a rule I established by measurement:

| shape | 200 KB unreferenced array |
|---|---|
| `var TablesFull = UnicodeTables{data: big[:]}` | **retained** |
| separate package, `func Tables() UnicodeTables` | dropped |
| same package, `func Full()/func ASCII()`, no package vars | dropped |

A package-level var holding slices of arrays is **not** statically initialised — it generates a
package `init`, and the inittask then retains *every* table set, including ones nobody
references. Therefore:

> **Table sets must be functions returning a value. Never package-level vars.**
> One `var DefaultTables = UnicodeFull()` added anywhere in package lefevre silently costs every
> user 275 KB. This must be a comment on the type, and §4's size test is the only real guard.

Both variants may live in `package lefevre`; no subpackage split is needed (verified).

A corollary worth exploiting: **the rule extends to convenience wrappers.** Any entry point that
is *itself* elidable may keep a `UnicodeFull()` default at no cost to the ASCII user. This saves
`BreakState.Begin` and `GuessTextProperties` from being API breaks. It does **not** save
`Breaker.AppendBreak`, which is on both tiers' path.

### 1.2 Types

The `u < K` bound in every current lookup is exactly `u>>shift < len(pageIndices)`
(921600/128 = 7200 = `len(graphemeBreakClassPageIndices)`; 918016/128 = 7172; 1048704/128 = 8193;
65408/32 = 2044). The constants are derivable, so one generic type reproduces all 8 lookups
bit-identically:

```go
// pageTable is a two-level page table over the Unicode codespace. The zero
// value answers zero for every codepoint, so a table a set omits needs no nil
// check anywhere.
type pageTable[T uint8 | uint16 | uint32] struct {
	pages    []uint8
	data     []T
	maxCP    uint32 // exclusive; at or above this the table answers zero
	highPage uint32 // data offset for codepoints past len(pages)<<shift
	shift    uint8  // 7 for 128-entry pages, 5 for mirroring
}

// UnicodeTables is the character-property data the segmenter and shaper consult.
//
// Always obtained from a function ([UnicodeFull], [UnicodeASCII]) and never
// exposed as a package variable: a package-level var of this type forces a
// package init referencing every table, and the linker then retains all
// ~273 KB even for a program that only asks for the ASCII set.
//
// The zero value covers nothing; every property answers zero.
type UnicodeTables struct {
	MaxRune rune // highest codepoint described, inclusive; 0 means empty
	grapheme, line, word, bidi, flags, joining pageTable[uint8]
	script                                     pageTable[uint16]
	mirror                                     pageTable[uint32]
	scriptExt                                  []uint8
}

func UnicodeFull() UnicodeTables   // ~273 KB, U+0000..U+10FFFD
func UnicodeASCII() UnicodeTables  // ~1.4 KB, U+0000..U+007F
func (u *UnicodeTables) Covered(cp rune) bool
```

The 8 free funcs in [unicode.go:203-327](unicode.go#L203-L327) become unexported methods. No
exported name changes.

### 1.3 What the ASCII tier actually contains

I verified each table's ASCII page directly rather than assuming:

| table | ASCII content | tier cost |
|---|---|---|
| grapheme, line, word, bidi, flags | populated | 129 B each |
| `scriptExtension` | all values have `count == 1` | 257 B |
| `mirrorCodepoint` | **8 nonzero: `()`, `<>`, `[]`, `{}`** | 516 B |
| `joiningType` | **entirely zero** | omitted |
| `scriptExtensions` list | never reached — the `count >= 2` branch at [segmentation.go:440](segmentation.go#L440) cannot fire | omitted |

**The mirror table must be kept.** Dropping it — the intuitive call, since mirroring reads like a
bidi concern — would silently break bracket pairing for ASCII, which is precisely the ASCII
user's common case. Tier-specific total ≈ **1,418 B**, against ~273 KB.

Omitted tables cost nothing to check: with `maxCP == 0` the table's *existing* first compare
returns zero. There is no nil check, and omission wins on both size and speed.

`graphemeBreakTransition` (234 B) and `scriptProps` (1,360 B) stay package arrays — indexed by
class and script, not by codepoint, so they are tier-independent.

### 1.4 Threading

There are only **9 call sites**, which makes this far smaller than it looks — seven consecutive
at [segmentation.go:330-336](segmentation.go#L330-L336) inside `breakAddCodepoint`, and one at
[font.go:923](font.go#L923) inside `assignJoiningForms`. Two injection points, everything else
plumbing.

`breakStateInternal` ([segmentation.go:67](segmentation.go#L67)) takes a `*UnicodeTables` —
a pointer, not a value: the struct is ~550 B and `BreakState.Begin` does `*bs = BreakState{}`,
so a value would have to be saved and restored on every `Begin`. `breakAddCodepoint` hoists it
once, so the indirection is one load per codepoint, not seven.

```go
// New, and Begin becomes an elidable wrapper that supplies the full set.
func (bs *BreakState) BeginUnicode(u *UnicodeTables, paragraphDirection Direction,
	style JapaneseLineBreakStyle, flags BreakConfigFlags)
```

**Measured cost of the indirection:** 1.52 ns/op vs 1.17 ns/op (+30%) on x86-64. Part of that is
per-lookup bounds checking, which narrows once the `Shaper` guarantees coverage before lookup.
Negligible beside GSUB/GPOS, but it is a real regression for desktop full-Unicode users and
should be benchmarked rather than assumed.

### 1.5 The `Shaper`

```go
var sh lefevre.Shaper
sh.Configure(lefevre.Config{
    Font:        &font,
    Unicode:     lefevre.UnicodeASCII(),
    Replacement: '?',   // 0 means U+FFFD; -1 drops uncovered runes
})
for { runs = sh.Shape(text) }   // zero allocation after warm-up
```

This is the requested shape, and it is load-bearing for three reasons a field on `Breaker`
cannot address:

1. **Substitution consistency.** `ShapeSimple` walks the text twice — once in `AppendBreak`
   ([shape.go:199](shape.go#L199)) and once in `Shape`'s rune loop
   ([shape.go:162](shape.go#L162)). Replacing uncovered runes in only one desynchronises
   `Break.Position` from `Glyph.UserID`. A `Shaper` decodes once and feeds both.
2. **Buffer ownership** for §1.6.
3. Ergonomics.

Coverage is one compare per rune (`uint32(r) > uint32(u.MaxRune)`) in that single decode loop —
not inside the 8 lookups, which would be 8 redundant compares and would still leave the desync.
Substitution is rune-for-rune so `Break.Position` and `Glyph.UserID` stay 1:1 with the source.
`Shaper.Substituted() int` reports the count; a counter rather than an error, because
substituting is the requested behaviour, not a failure.

> **`'?'` and U+FFFD are not interchangeable here.** Under an ASCII set U+FFFD is *itself*
> uncovered, so it would be replaced and then map to `.notdef`. A set whose `MaxRune < 0xFFFD`
> must use `'?'`; `Configure` should reject the combination rather than render boxes.

Memory safety is independent of all of this: `pageTable.maxCP` is the same compare that exists
today, so even a hand-built or truncated `UnicodeTables` cannot index out of bounds.

### 1.6 Allocations to remove

`Shape` allocates on every call. Also — found by reading, and larger than the `shape.go` set —
so does the OpenType layout code, per feature, per run.

| site | what | replacement |
|---|---|---|
| [shape.go:118](shape.go#L118) | `map[FeatureTag]bool` per call — violates [agents.md](agents.md)'s slices-over-maps rule | `[]FeatureTag`, linear scan (counts ≤ 8) |
| [shape.go:84](shape.go#L84) | `heapBounds` >16-boundary fallback | reused buffer; deletes the duplicated second scan and the `getBound` closure |
| [shape.go:161](shape.go#L161) | `glyphs` re-allocated per run via `glyphs = nil` | `scratch` + `store` buffers |
| [shape.go:199](shape.go#L199) | `[]byte(text)` copies the whole string | `AppendBreakString`, or drive `BreakState` from `utf8.DecodeRuneInString` |
| [shape.go:197](shape.go#L197) | `var b Breaker` — ~1.3 KB zeroed per call | owned `Breaker` |
| [font.go:837](font.go#L837), [font.go:1961](font.go#L1961) | `gsub`/`gposFeatureLookups` allocate **per matching feature, per run** — 6 GSUB + 3 GPOS defaults | `dst`-appending signatures fed from a `shapeCtx` |
| [font.go:1430](font.go#L1430) | `readSequenceLookupRecords` per contextual subtable — every `calt`/`rclt` | `shapeCtx.records` with **stack discipline**; contextual lookups re-enter, so a flat buffer would be clobbered |
| [font.go:1252](font.go#L1252) | multiple-subst: `make` **plus** a nested `append(append(...))` | grow in place and `copy` |
| [font.go:406](font.go#L406) | `LoadBytes` allocates **9×** per call (eager `readNameTable` UTF-16 decode) | make the name-table parse opt-in |

A `shapeCtx{disabled, lookups, records}` carrier threads through the unexported layout functions
rather than adding three parameters to ten signatures.

One subtlety: appending a finished run's glyphs to `store` may reallocate and invalidate earlier
`Run.Glyphs`. Record `(lo, hi)` spans during the loop and set `runs[i].Glyphs = store[lo:hi:hi]`
*after* it; the three-index slice stops a caller's `append` writing into the next run.

### 1.7 API breakage

| change | severity |
|---|---|
| `Breaker` gains exported `Unicode` field | **behavioural break** — zero-value `Breaker` no longer segments |
| `Breaker` gains `Err()`; `AppendBreak` may fail | additive, but it must still consume all input or [segmentation_test.go:194](segmentation_test.go#L194)'s loop spins forever |
| `ShapeConfig` gains `Unicode` | **break, Arabic only** — a zero set leaves glyphs isolated |
| `BreakState.Begin`, `GuessTextProperties` | **no break** (elidable-wrapper corollary) |
| 8 `getX` funcs → methods; layout function signatures | no break (unexported) |
| `UnicodeTables`, `UnicodeFull`, `UnicodeASCII`, `Shaper`, `Config`, `NewBreaker`, `BeginUnicode` | additive |

**There is no source-compatible path for `Breaker`.** An implicit `UnicodeFull()` default inside
`AppendBreak` is reachable from both tiers and kills DCE. Make the failure loud instead: sticky
error, input consumed, no breaks emitted, documented on the type.

### 1.8 Generator

[cmd/gentables/main.go](cmd/gentables/main.go) is restructured around table *pairs* and *tiers*,
so adding a middle tier (Latin-1, Latin+punctuation) is one line in a `tiers` slice. Per
(spec, tier): take the index entries the tier needs, collect distinct source pages in
first-appearance order, remap, emit — and **emit nothing if every entry is zero**. Generate the
`UnicodeFull`/`UnicodeASCII` constructors too; they are boilerplate that must stay in sync with
the tier list, and hand-maintaining them is exactly where a middle tier would break.

Keep the full tier's identifiers unchanged so the regeneration diff is verifiable: full output
should be byte-identical to today's file modulo formatting.

A tier with internal holes is nearly free, because the page-index array is already the sparse
map: uncovered pages all point at one shared zero page, costing 1 byte per 128 codepoints.

---

## 2. Alpha rastering

The mask [Rasterize](raster/scanline.go) produces is already right: linear area coverage, no
gamma, no premultiplication, nonzero winding. The problems are downstream, where lefevre
currently owns nothing — so every failure mode lands on the caller.

### 2.1 Fixes to existing code

- [raster/scanline.go:124](raster/scanline.go#L124) `byte(c * 255)` **truncates**, biasing every
  antialiased pixel ~0.5/255 dark. Change to `byte(c*255 + 0.5)`. **Verified: passes the existing
  raster suite unchanged**, including `_InteriorSolid` (requires exactly 255) and
  `_NonIntegerRectArea` (±1.5 of 49).
- [examples/raster/main.go:150-164](examples/raster/main.go#L150-L164) writes `color × coverage`
  with `A=255` into an `*image.NRGBA`. `NRGBA` is *straight* alpha, so this stores premultiplied
  values in a straight-alpha image and discards the background — it composites against black, in
  gamma space. It is the exact double-multiply failure the video describes, shipped as the
  example people will copy. Rewrite it.

### 2.2 New API in `raster`

The video's conclusion is that the format should have carried one mandatory tag saying whether
data is premultiplied. Encode that as a type so it cannot be defaulted or forgotten:

```go
// AlphaMode names the relationship between color channels and alpha. It cannot
// be detected from the data and guessing wrong fails silently: straight data
// treated as premultiplied haloes every edge; premultiplied data treated as
// straight darkens edges into the background. Nothing here infers it.
type AlphaMode uint8

const (
	// AlphaPremultiplied: channels already scaled by alpha. The only
	// representation closed under filtering, so it is what a GPU sampling a
	// glyph atlas, or any code resizing a composite, requires.
	AlphaPremultiplied AlphaMode = iota
	AlphaStraight
)

// BlendSpace selects the space coverage is applied in. Text is where this shows
// most: blending encoded sRGB makes light-on-dark text bolder and dark-on-light
// text thinner than the outline says.
type BlendSpace uint8

const (
	BlendLinear  BlendSpace = iota // to linear light, blend, re-encode
	BlendEncoded                   // blend sRGB directly: wrong, cheaper, universal
)

type Compositor struct {
	Mode  AlphaMode
	Space BlendSpace
}

// DrawCoverage composites a solid color through a coverage mask into 8-bit RGBA
// dst, which must already be in c.Mode's convention. Allocates nothing.
func (c *Compositor) DrawCoverage(dst []byte, dstStride, dstX, dstY int,
	cov []byte, covStride, w, h int, r, g, b, a uint8)

func Premultiply(pix []byte)

// Unpremultiply is the inverse. It cannot recover precision Premultiply
// discarded, so a round trip is lossy at low alpha — the degradation
// compositing apps introduce when they "scare an image straight".
func Unpremultiply(pix []byte)
```

**Complexity: low.** ~200-250 lines plus two generated LUTs (sRGB→linear 256×uint16, 512 B;
linear→sRGB 4096×uint8, 4 KB) from the existing `cmd/gentables`. The LUTs are reached only from
compositor function bodies, so they eliminate at function granularity — §1.1's init hazard does
not apply, because nothing holds them in a package var.

Signature style follows the package's existing `[]byte` + explicit stride convention rather than
introducing an image type, keeping `raster` dependency-free. Documentation must state that Go's
`image.RGBA` is premultiplied and `image.NRGBA` is straight — that mapping is what callers get
wrong, as the example proves.

---

## 3. CFF / CFF2

Per your direction: implement **CFF1**, structure for a CFF2 fork, record CFF2 incompatibilities
as comments where they will bite.

### 3.1 Exported API diff — smaller than fyne.md suggests

fyne.md:205 calls this "not a patch". The rasterizing half is in fact nearly API-neutral:
`GlyphOutline` already dispatches on `OutlineFormat`, so OTTO fonts simply start returning
outlines. The only public change is that `Segment` learns a cubic:

```go
const (
	SegmentMoveTo SegmentOp = iota
	SegmentLineTo
	SegmentQuadTo
	SegmentClose
	SegmentCubeTo // appended, not inserted, so SegmentClose keeps its value
)

type Segment struct {
	Op       SegmentOp
	X, Y     int16 // Endpoint (font units)
	Cx, Cy   int16 // First control point (QuadTo, CubeTo)
	Cx2, Cy2 int16 // Second control point (CubeTo only)
}
```

`Segment` grows 10 → 14 bytes. This is what [piudf](../piudf) actually wants: its `Path.QuadTo`
degree-elevates to cubic immediately because PDF has no quadratic operator, so a CFF cubic maps
onto `Path.CubicTo` losslessly.

### 3.2 Work

1. **Structural parse** — header, Name/TopDICT/String/GlobalSubr INDEXes, Top DICT (CharStrings,
   Private, charset, ROS, FDArray, FDSelect), Private DICT (Subrs, nominal/default widthX).
   Offsets only, stored on `Font`, no allocation. Small enough to run at load.
2. **Type2 charstring interpreter** — the bulk, ~600-900 lines. Path and arithmetic operators,
   subroutine calls with the 107/1131/32768 bias, stem counting to size `hintmask`, the four flex
   operators, `endchar` with `seac`, and CID `FDSelect` format 0/3 selecting per-glyph
   Private/local subrs. A 48-entry operand stack and 10-deep call stack, both arrays; appends
   into the caller's `dst []Segment`. Zero heap.
   **Keep it reachable only from `glyphOutline`** so metrics-only and shaping-only users still
   eliminate it — enforced by §4's size test.
3. **Raster** — `flattenCubic` beside `flattenQuad`, same flatness test and De Casteljau split.
   `GlyphBox` unaffected.
4. **sfnt** — [sfnt/write.go:74](sfnt/write.go#L74) hardcodes sfntVersion `0x00010000`; derive it,
   emitting `OTTO` when the table set contains `CFF `/`CFF2`.
   [sfnt/subset.go:47](sfnt/subset.go#L47) returns `ErrNoOutlines` for CFF. Phase one: copy the
   `CFF ` table verbatim and add it to `required`. Glyph ids already never shift, so PDF
   embedding via piudf works immediately, with no charstring size reduction. Rewriting the
   CharStrings INDEX to drop glyphs is a separate, later change.

### 3.3 CFF2 incompatibilities, to be written as comments

- CFF1 encodes an optional glyph width as a leading operand on the first stack-clearing operator;
  **CFF2 removes width entirely** (advances come from `hmtx`). The interpreter needs a
  charstring-type branch here — this is the one that silently corrupts every glyph if missed.
- CFF2 adds `blend` (16) and `vsindex` (15); operands become deltas against an ItemVariationStore,
  requiring `fvar`/`avar` resolution.
- CFF2 has no `endchar`, no `seac`, no Encoding, no charset.
- CFF2 INDEXes use a 32-bit count, not 16-bit.
- CFF2's operand stack is 513 deep, not 48.

### 3.4 Test material

No OTTO font in [testdata/](testdata/). This machine has 252 (vs 741 TrueType, 5 collections),
but they should not be vendored without a licence check. Use a small synthesised CFF for unit
tests plus an optional pass over system fonts that skips when absent.

---

## 4. Ordered implementation

Tests first, behavioural, against exported API only ([agents.md](agents.md)).

| phase | work | acceptance |
|---|---|---|
| 0 | Write the tests in §5 against the intended API; record today's `ShapeSimple` alloc baseline | compile-fails until phase 2 |
| 1 | Generator restructure (§1.8), regenerate | **full-tier diff is empty modulo formatting**; existing suite untouched |
| 2 | `pageTable`, `UnicodeTables`, tier constructors; lookups→methods; thread through the 9 call sites | existing suite passes on full tables; ASCII-agrees-with-full goes green |
| 3 | `Shaper`, `Config`, `shapeCtx`; single decode loop with substitution | substitution + Arabic tests green; **size test green** |
| 4 | Allocation removal — `font.go` items first (they dominate and `shapeCtx` is a prerequisite), then `shape.go` | `AllocsPerRun == 0` |
| 5 | Raster rounding fix; `Compositor` + LUTs; rewrite the raster example | compositing tests green |
| 6 | CFF structural parse → interpreter → `flattenCubic` → sfnt OTTO | CFF outline tests green; size test still green |

CFF is last because it is the most self-contained: it touches `Segment`, the outline path and
sfnt, but none of §1's machinery. It can also be lifted out and done first if OTTO support is
more urgent than ESP32 size.

---

## 5. Verification

- `go test ./...` unchanged throughout. The raster rounding change is already verified against it.
- **Size regression test** — the only real guard on §1.1, since the failure it catches is
  invisible everywhere else: the code still works, it just ships a quarter megabyte. Build a
  fixture, run `go tool nm`, assert no full-tier array is linked. Skip under `-short` or when
  `go` is absent.

  Two things make this test easy to write so that it can never fail, which is worse than not
  having it. Both are worth keeping in mind if it is ever rewritten:
  its symbols are fully qualified (`github.com/soypat/lefevre.bidiClassData`), and every
  full-tier name is a *prefix* of its ASCII counterpart (`bidiClassDataASCII`) — so a substring
  or loose-suffix match silently never fires. Match the last whitespace-separated field exactly,
  and assert the ASCII symbols *are* present, or the test also passes against a binary that
  linked no tables at all. Confirm it by introducing the bug it guards against
  (`var x = UnicodeFull()`) and watching it fail.

  Targets, relative to the empty-program baseline:

  | tier | measured, over baseline |
  |---|---|
  | load + cmap | +49,152 |
  | + outline + raster | +61,440 |
  | **ASCII shaping** | **+147,456** |
  | full shaping | +425,984 |

  **Restricting the tables to ASCII saves 278,528 B — a 65% cut.** `go tool nm` confirms every
  full-tier array is absent from the ASCII binary; only the 1,408 B of ASCII tables remain,
  alongside the tier-independent `scriptProps` (1,360 B) and `graphemeBreakTransition` (234 B).

  Threading the tables cost existing full-Unicode users 1,879 B of code (60,562 → 62,441 B of
  `.text`); the rest of the change between +409,600 and +425,984 is section-alignment
  granularity. The generic `pageTable` methods inline completely — no `pageTable` symbol
  survives — so the +30% microbenchmark figure above overstates the cost in the real call path.

- **Allocation** via `testing.AllocsPerRun`: `Shaper.Shape` → 0 for Latin LTR, Arabic RTL and
  mixed text; `LoadBytes` 9 → 0 with the name-table opt-out; `GlyphOutline` is already 0 and must
  stay 0 for CFF.
- **Behavioural, ASCII tier**: every rune U+0000–U+007F yields identical breaks under ASCII and
  full tables; `"(a)"` / `"[a]"` pair correctly (pins the mirror-table finding); `"café"` under
  ASCII replaces one rune with positions unchanged and `Substituted() == 1`; Arabic under ASCII
  does not panic; zero-value `Breaker` consumes all input and reports an error without looping.
- **CFF**: rasterize the same glyph from TrueType and OTTO cuts of one face and compare coverage;
  exercise `seac` accents and a CID-keyed font specifically.
- **Compositing**: a 50%-coverage pixel over a known background gives the linear-light answer
  under `BlendLinear`; `Premultiply`/`Unpremultiply` round-trip within quantisation at high
  alpha; a mismatched `AlphaMode` is impossible to express rather than merely discouraged.

### Reproducing the measurements

```sh
# Tier sizes: build each fixture against a replace-directive module.
go build -ldflags="-s -w" -o /tmp/t ./<tier> && stat -c%s /tmp/t
go build -o /tmp/tu ./<tier> && go tool nm -size /tmp/tu | grep lefevre | sort -rn -k2

# The DCE rule: var-vs-func, same 200 KB array.
go tool nm -size /tmp/tu | grep -E 'inittask|\.init$'   # a package init means DCE is dead

# ASCII page contents, per table: decode page 0 of unicode_tables.go and count nonzero.
# Font format census.
find /usr/share/fonts \( -name '*.otf' -o -name '*.ttf' -o -name '*.ttc' \) \
  -exec sh -c 'head -c4 "$1" | xxd -p' _ {} \; | sort | uniq -c | sort -rn
```
