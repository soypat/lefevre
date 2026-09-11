# Feasibility: replacing go-text/typesetting + go-text/render with lefevre

**Deliverable: evaluation only. No code changes proposed.**

## Context

Fyne's entire text stack is `github.com/go-text/typesetting v0.3.4` + `github.com/go-text/render v0.2.1`.
The question is whether `../lefevre` (`github.com/soypat/lefevre`) can replace them, primarily to
reduce binary size, and secondarily to remove C dependencies pulled in by the harfbuzz path.

Decisions taken during this study (user):
- Color emoji may be dropped.
- Font fallback would be restored by adding cmap enumeration to lefevre plus a fyne-side font scanner.

---

## 1. The C premise is wrong — there is no C in the text path

`go-text/typesetting`'s `harfbuzz` package is a **pure-Go port** of harfbuzz (22,757 LOC), not a cgo
binding. Verified by `grep -rl 'import "C"'` over the module cache:

| module | files with `import "C"` |
|---|---|
| `go-text/typesetting@v0.3.4` | 0 |
| `go-text/render@v0.2.1` | 0 |
| `srwiley/rasterx`, `srwiley/oksvg` | 0 |

The only C in a fyne binary comes from GLFW/X11 — `_glfw` (139,712 B data), `_glfwInitX11` (8,528 B),
`keysymtab`, `_glfwDefaultMappings`, etc.; bindiff attributes 223,842 B to `[c]`. None of it is
text-related. **Swapping to lefevre removes zero C.** Both libraries are pure Go with no cgo.

---

## 2. Measured binary size

Target: `go build -tags x11 ./cmd/hello` (Wayland headers unavailable; `-tags x11` is required to build).

| build | bytes |
|---|---|
| unstripped | 32,170,472 |
| **stripped (`-ldflags="-s -w"`)** | **23,939,096** |
| stripped + `-tags no_emoji` | 19,706,296 |

`bindiff -kind section` on the unstripped binary: `.text` 7,569,024 / `.gopclntab` 5,893,524 →
**pclntab ≈ 0.779 × text**. That ratio is used below to project the stripped cost of code, since
`.gopclntab` survives `-s -w` and bindiff cannot attribute it per package.

### 2.1 Where the bytes actually are

`bindiff -kind package profile`, top of the text-relevant set:

```
fyne.io/fyne/v2/theme                                6,539,144
github.com/go-text/typesetting/font/opentype/tables    420,867
github.com/go-text/typesetting/harfbuzz                350,075
github.com/go-text/typesetting/fontscan                190,077
github.com/go-text/typesetting/font                    115,880
github.com/go-text/typesetting/internal/unicodedata     58,397
github.com/srwiley/rasterx                              62,278   (fyne uses directly, stays)
github.com/srwiley/oksvg                                48,882   (reachable only via render)
github.com/go-text/typesetting/font/cff                 36,534
github.com/go-text/typesetting/shaping                  32,500
github.com/go-text/typesetting/language                 25,851
github.com/go-text/typesetting/font/cff/interpreter     17,121
github.com/go-text/typesetting/font/opentype             8,789
github.com/go-text/render                                7,915
github.com/go-text/typesetting/segmenter                 6,728
github.com/go-text/typesetting/di                          382
```

**`fyne.io/fyne/v2/theme` at 6.54 MB dwarfs the whole text stack.** `go tool nm -size` breaks it down:

```
4,232,712  theme..gobytes.1   = theme/font/EmojiOneColor.otf
  471,004  theme..gobytes.5   = NotoSans-BoldItalic.ttf
  470,472  theme..gobytes.4   = NotoSans-Italic.ttf
  455,188  theme..gobytes.2   = NotoSans-Regular.ttf
  455,164  theme..gobytes.3   = NotoSans-Bold.ttf
  341,860  theme..gobytes.6   = DejaVuSansMono-Powerline.ttf
```

That 4.23 MB emoji font is **17.7% of the stripped binary** and is already removable today with
`-tags no_emoji`, with no code change at all (`theme/bundled-emoji.go` / `theme/unbundled-emoji.go`).

### 2.2 go-text vs lefevre, measured

`go tool nm -size`, summing `T`/`t` and `R`/`r`/`D`/`d`/`B`/`b`:

| | .text | data+rodata | proj. pclntab | **total (stripped)** |
|---|---|---|---|---|
| go-text stack as linked in fyne | 988,586 | 304,564 | 770,109 | **2,063,259** |
| `srwiley/oksvg` (via render only) | 47,742 | 1,148 | 37,192 | **86,082** |
| lefevre root + `raster`, fully linked | 164,134 | 275,115 | 127,861 | **567,110** |

lefevre's figure is an **upper bound** — taken from `go test -c` binaries, where nothing is
dead-code-eliminated. Its 275 KB of data is the generated Unicode tables (`scriptExtensionData`
62,720 B, `unicodeFlagsData` 31,104, `bidiClassData` 29,696, `wordBreakClassData` 27,648,
`lineBreakClassData` 26,752, `graphemeBreakClassData` 20,096, `joiningTypeData` 14,848, plus
page-index arrays). Those are unconditionally reachable, so they will not shrink.

### 2.3 Where go-text's code size really lives

Per-subpackage `.text` only:

```
342,008  font/opentype/tables        ← generated binary-table parsers
 95,104  harfbuzz (package funcs)
 35,273  harfbuzz.(*otApplyContext)
 14,581  harfbuzz.(*Buffer)
 13,595  harfbuzz.(*indicShapePlan)
  9,803  harfbuzz.(*aatApplyContext)
  7,555  harfbuzz.(*complexShaperArabic)
 63,500  font
 17,101  fontscan
 12,163  shaping.(*LineWrapper)
 11,854  font/cff/interpreter.(*CharstringReader)
```

Worth noting: **`font/opentype/tables` (342 KB) is larger than the entire shaper (~180 KB).** The cost
is in parsing every OpenType table, not in harfbuzz. lefevre's `font.go` indexes **15 tables**; go-text
parses essentially all of them. That is the same trade as the feature gap in §3 — the bytes and the
capability are the same bytes.

### 2.4 Projected result on the chosen path

Drop emoji, swap to lefevre, add cmap enumeration + a fyne-side font scanner:

```
baseline stripped                          23,939,096
  − EmojiOneColor.otf                      − 4,232,712
  − go-text/typesetting + go-text/render   − 2,063,259
  − srwiley/oksvg                          −    86,082
  + lefevre (root + raster)                +   567,110
  + new fyne glue (est. ~40 KB text)       +   100,000
                                          ─────────────
projected                                  ≈18,224,153   (−23.9%)
```

**Attribution of that −5.71 MB: 4.23 MB (74%) is the emoji font; the lefevre swap contributes
1.48 MB (26%).**

Against the honest baseline — `-tags no_emoji` today, 19,706,296 B — the lefevre swap is
**−1.48 MB, −7.5%**.

---

## 3. What fyne uses, and what lefevre is missing

### 3.1 Fyne's actual dependency surface

go-text appears in only 5 non-test files. The hub is
[internal/painter/font.go](internal/painter/font.go):

- `font.ParseTTF`, `*font.Face`, `Face.NominalGlyph`, `font.Aspect`
- `fontscan.NewFontMap` / `UseSystemFonts` / `SetQuery` / `SetScript` / `ResolveFace` / `ResolveFaceForLang`
- `shaping.HarfbuzzShaper.Shape`, `shaping.Segmenter.Split`, `shaping.Input/Output/Glyph`, `shaping.Fontmap`
- `render.Renderer{FontSize,PixScale,Color}.DrawShapedRunAt` / `DrawStringAt`
- `language.NewLangID`, `language.LookupScript`
- `di.DirectionLTR` — **hardcoded**; fyne does no bidi today

Supporting files: [internal/painter/font_prod.go](internal/painter/font_prod.go) (`UseSystemFonts`),
[internal/painter/font_testhelper.go](internal/painter/font_testhelper.go) (stub),
[internal/test/text.go](internal/test/text.go), and
[widget/richtext.go](widget/richtext.go) — the sole user of `shaping.LineWrapper` + `WrapConfig`,
for ellipsis truncation only (ordinary wrapping is fyne's own `lineBounds`/`splitLines` loop).

Fyne implements `shaping.Fontmap` three times itself (`dynamicFontMap`, `fixedFontMap`,
`FontCacheItem.Fonts`) and does its own fallback chaining, shared-baseline ascent across runs
(`walkString`), `.notdef` run splitting, and VS16/keycap emoji sequence grouping
(`splitEmojiSequences`). **That fyne-side logic survives a swap unchanged** — it is the strongest
argument that the integration is tractable.

### 3.2 Gap matrix

| capability | go-text | lefevre | impact on fyne |
|---|---|---|---|
| TrueType `glyf` outlines | yes | yes | none |
| **CFF / CFF2 (`OTTO`)** | yes | **absent** | **blocking, see §3.3** |
| Cubic bezier segments | yes | **`SegmentOp` has no cubic op** | public-API change to fix |
| **OT-SVG (`SVG `)** | yes | **absent** | emoji dead |
| **COLRv0/v1, CPAL** | parsed (render lacks `GlyphColor` case) | **absent** | emoji dead |
| **CBDT/CBLC, sbix, EBDT** | yes | **absent** | emoji dead |
| cmap formats | 0,2,4,6,12,13,14 | **4 and 12 only** | format-0/6 fonts → all `.notdef`, silently |
| **System font discovery** | `fontscan` | **absent** | must be written in fyne |
| **Font fallback / coverage query** | `fontscan.ResolveFace` | **absent** (`GlyphID` is one-way; lefevre's own doc calls this "the blocker for font fallback") | must add cmap enumeration |
| GSUB lookup types | all 8 | **all 8** | none |
| GPOS lookup types | 1–9 | 1,2,4,5,6,9 — **no 3 (cursive), no 7/8 (contextual)** | minor for Latin; breaks some scripts |
| Arabic shaping | yes | **yes** | none |
| **Indic / USE / Khmer / Myanmar / Thai / Tibetan / Hangul** | real shapers (`indicShapePlan` is 13.6 KB of linked code) | **classified in `scriptProps`, never dispatched** — `shape.go:130` only tests `ShaperArabic` | those scripts mis-render |
| AAT `morx`/`kerx` (macOS fonts) | yes (`aatApplyContext`) | **absent** | macOS system-font regressions |
| UAX#9 bidi | `x/text/unicode/bidi` available | **direction itemization only** — no embedding levels, no L1/L2 reordering | fyne hardcodes LTR today, so no regression |
| UAX#14 line breaking | `segmenter` | **yes, and stronger** — 45 classes, EA splits, 6-level priority, kinsoku | fyne currently doesn't use it at all |
| UAX#29 grapheme/word | yes | **yes** | none |
| Variable fonts (`fvar`/`gvar`) | yes | absent | no current fyne use |
| Legacy `kern` table | yes | absent (GPOS `kern` feature only) | marginal |
| Rasterizer output | `render` → `*image.RGBA` via rasterx, dispatches outline/bitmap/SVG | `raster` → 8-bit alpha mask into `[]byte`+stride, 4× vertical supersample, nonzero winding, **quadratics only** | fyne must write mask→RGBA compositing itself |

### 3.3 CFF is the real blocker, not emoji

On this machine, **252 of 998 system fonts are `OTTO`** (CFF outlines) — 25%. lefevre's
`selectCmap`/outline path gives those `OutlineNone`, and a font whose cmap is format 0 or 6 maps
*every* rune to glyph 0 with no error. Combined with a fyne-side scanner (the chosen fallback path),
fyne would have to filter OTF out of discovery entirely, shrinking the usable system font pool by a
quarter on Linux and far more on macOS, where system `.ttc`s are CFF.

Fixing it is not a patch. `Segment{Op, X, Y, Cx, Cy int16}` has no cubic representation, so CFF
touches: the public `Segment` type, `Font.GlyphOutline`, `raster.ScanlineRasterizer.Rasterize`
(`flattenQuad` only), `raster.GlyphBox`, and `sfnt.Subsetter`. Reference cost from the other side:
go-text spends 36,534 + 17,121 B on `font/cff` + `interpreter`.

### 3.4 Work the swap forces on fyne

1. **Replace `render.Renderer`** — `DrawShapedRunAt` handles outline/bitmap/SVG dispatch, color,
   `PixScale`, and subpixel origin. lefevre's `raster` produces bare alpha masks; fyne writes the
   compositing into `*image.RGBA` (`internal/painter/font.go:203-218`).
2. **Replace `fontscan`** — directory walking per OS, family/aspect matching, a face cache, and
   locale→font resolution for `ResolveFaceForLang`. `fontscan` is 190 KB attributed; a minimal
   fyne version will not be free.
3. **Add cmap enumeration to lefevre** — needed for `dynamicFontMap`'s `NominalGlyph` coverage
   probes. lefevre's own `FONT-LOWLEVELlocal.md` already scopes this as `iter.Seq2[rune, uint16]`
   over the range-structured format 4/12 tables; genuinely cheap.
4. **Replace `shaping.LineWrapper`** in [widget/richtext.go:1301](widget/richtext.go#L1301) — but
   lefevre's `Breaker` (UAX#14) is a better foundation than what fyne uses today.
5. **Metrics mapping** — fyne reads `Output.LineBounds.Ascent` and `LineBounds.LineThickness()`;
   lefevre exposes `VMetrics()` in raw font units. Note `ShapeConfig.Size` is **declared but never
   read** in `shape.go` — all lefevre output is in font units regardless, so fyne scales downstream.
6. **Cross-repo churn** — lefevre is `go 1.25.7`; fyne's `go.mod` says `go 1.22.0`. Adopting it
   raises fyne's floor by three releases.

---

## 4. Recommendation

**Do not swap for binary size.** The measurement does not support the premise:

- There is **no C to remove** — harfbuzz here is pure Go.
- The swap is worth **1.48 MB of a 23.94 MB binary (6.2%)**, and only **7.5%** measured against a
  `no_emoji` baseline.
- **`-tags no_emoji` alone is worth 4.23 MB (17.7%)** — 2.9× the swap's value, zero code, zero
  capability loss beyond the emoji the swap also forfeits.

If binary size is the goal, the ordered wins are:

1. **`-tags no_emoji`** → −4.23 MB. Already exists. Free.
2. **Trim bundled text faces** → NotoSans ×4 is 1.85 MB; DejaVuSansMono adds 342 KB. Subsetting or
   shipping fewer aspects is worth 1–1.5 MB with no API change. (lefevre's `sfnt.Subsetter` is
   genuinely good at this and is usable as a build-time tool **without** adopting it at runtime —
   this is the one place lefevre pays off today.)
3. **Only then** consider the shaper swap, for 1.48 MB and a large capability regression.

**lefevre's genuine strengths** — zero dependencies, the UAX#14/UAX#29 segmentation (a faithful port
of a mature C implementation, better factored than go-text's), complete GSUB coverage, and a clean
subsetter — are real. They are just not what fyne is short of. Its README states "the scope of this
package is fulfilled," which is accurate for its own goals (segmentation + Latin/Arabic + raster) and
misaligned with fyne's, which is rendering arbitrary user text in arbitrary system fonts.

**Conditions under which this becomes worth revisiting:**

- CFF/Type2 lands in lefevre (cubic `SegmentOp` through parser, rasterizer, subsetter), **and**
- cmap enumeration lands (its own doc already plans it), **and**
- Indic/USE dispatch exists, **or** fyne formally scopes itself to Latin/Greek/Cyrillic/Arabic.

Until then the swap trades 6% of binary size for OTF system fonts, color emoji, and every Brahmic
script.

---

## 5. Verification / reproducing these numbers

```sh
SP=/tmp/claude-1000/-home-pato-src-tg-fyne/c6814d13-fbd5-41f0-a718-07da251ef83c/scratchpad

# Wayland headers are absent on this box; -tags x11 is required.
go build -tags x11                    -o $SP/hello_x11     ./cmd/hello
go build -tags x11 -ldflags="-s -w"   -o $SP/hello_strip   ./cmd/hello
go build -tags "x11 no_emoji" -ldflags="-s -w" -o $SP/hello_noemoji ./cmd/hello

# bindiff needs .symtab — profile the UNSTRIPPED binary.
bindiff -kind section profile $SP/hello_x11                  # pclntab/text ratio
bindiff -kind package -top 45 profile $SP/hello_x11          # package attribution
# NB: -kind file fails on this binary ("DWARF format error: short read"); use -kind symbol.

# text vs data split, which bindiff's package view merges
go tool nm -size $SP/hello_x11 | awk '$3~/^[Tt]$/'   | grep go-text | awk '{s+=$2}END{print s}'
go tool nm -size $SP/hello_x11 | awk '$3~/^[RrDdBb]$/' | grep go-text | awk '{s+=$2}END{print s}'

# lefevre upper bound: test binaries, nothing DCE'd
cd ../lefevre && go test -c -o $SP/lef.test . && go test -c -o $SP/lefr.test ./raster
go tool nm -size $SP/lef.test | awk '$3~/^[Tt]$/' | grep soypat/lefevre | awk '{s+=$2}END{print s}'

# cgo check
grep -rl 'import "C"' $(go env GOMODCACHE)/github.com/go-text/   # → empty

# emoji font table directory (OTTO + CFF + SVG)
python3 -c "import struct;d=open('theme/font/EmojiOneColor.otf','rb').read();\
n=struct.unpack('>H',d[4:6])[0];print(d[:4]);\
[print(d[12+16*i:16+16*i].decode('latin1'),struct.unpack('>II',d[20+16*i:28+16*i])[1]) for i in range(n)]"
```

Caveat on the projections: the `.gopclntab` multiplier (0.779) is a whole-binary average. Table-heavy
generated code such as `font/opentype/tables` skews denser in small functions, so the go-text removal
figure is more likely under- than over-stated. Every other number above is directly measured.
