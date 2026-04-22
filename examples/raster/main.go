// Example raster demonstrates CPU glyph rasterization with and without
// antialiasing, writing the result to a PNG image.
//
// Usage:
//
//	go run ./examples/raster testdata/DejaVuSans.ttf raster.png
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"

	kb "github.com/soypat/lefevre"
	"github.com/soypat/lefevre/raster"
)

type Flags struct {
	FontPath  string
	Text      string
	OutputPNG io.Writer
}

func main() {

	var (
		flags      Flags
		flagOutPNG string
	)
	flag.StringVar(&flags.FontPath, "font", "", "Path to font to use")
	flag.StringVar(&flags.Text, "text", "lefevre rastering!", "Text to render")
	flag.StringVar(&flagOutPNG, "o", "raster.png", "Output PNG raster file")
	flag.Parse()
	var buf bytes.Buffer
	flags.OutputPNG = &buf
	err := run(flags)
	if err != nil {
		flag.Usage()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = os.WriteFile(flagOutPNG, buf.Bytes(), 0777)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to write png file:", err)
		os.Exit(1)
	}
}

func run(args Flags) error {
	if args.FontPath == "" {
		return errors.New("need font path")
	}
	data, err := os.ReadFile(args.FontPath)
	if err != nil {
		return err
	}
	font, err := kb.FontFromMemory(data, 0)
	if err != nil {
		return err
	}

	const fontSize = 72
	scale := float32(fontSize) / float32(font.Info().UnitsPerEm)

	textNoAA := "NO AA: " + args.Text
	textAA := "   AA: " + args.Text
	glyphsNoAA := glyphIDs([]rune(textNoAA), font)
	glyphsAA := glyphIDs([]rune(textAA), font)
	glyphs := append(append([]uint16(nil), glyphsNoAA...), glyphsAA...)

	const atlasW, atlasH = 2048, 256
	atlas := make([]byte, atlasW*atlasH)
	placements := make([]raster.PackedGlyph, len(glyphs))
	cfg := raster.PackConfig{Font: font, Scale: scale, Padding: 1}
	if err := cfg.BakeAtlas(&raster.ScanlineRasterizer{}, glyphs, atlas, atlasW, atlasH, placements); err != nil {
		fmt.Fprintln(os.Stderr, "atlas bake error:", err)
		os.Exit(1)
	}

	const width, height = 1400, 320
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillBackground(img, color.NRGBA{R: 20, G: 20, B: 30, A: 255})

	baselineNoAA := 120
	drawText(img, atlas, atlasW, placements[:len(glyphsNoAA)], glyphsNoAA, font, scale, 40, baselineNoAA, false)
	baselineAA := 240
	drawText(img, atlas, atlasW, placements[len(glyphsNoAA):], glyphsAA, font, scale, 40, baselineAA, true)

	if err := png.Encode(args.OutputPNG, img); err != nil {
		return fmt.Errorf("encoding png: %w", err)
	}
	return nil
}

func glyphIDs(runes []rune, font *kb.Font) []uint16 {
	ids := make([]uint16, len(runes))
	for i, r := range runes {
		ids[i] = font.GlyphID(r)
	}
	return ids
}

func fillBackground(img *image.NRGBA, col color.NRGBA) {
	for y := 0; y < img.Rect.Dy(); y++ {
		off := y * img.Stride
		for x := 0; x < img.Rect.Dx(); x++ {
			img.Pix[off+0] = col.R
			img.Pix[off+1] = col.G
			img.Pix[off+2] = col.B
			img.Pix[off+3] = col.A
			off += 4
		}
	}
}

func drawText(img *image.NRGBA, atlas []byte, atlasW int, placements []raster.PackedGlyph, glyphs []uint16, font *kb.Font, scale float32, startX, baseline int, aa bool) {
	penX := float32(startX)
	penY := float32(baseline)
	for i, gid := range glyphs {
		p := placements[i]
		dstX := int(penX) + p.Xoff
		dstY := int(penY) + p.Yoff
		drawGlyph(img, atlas, atlasW, p, dstX, dstY, color.NRGBA{R: 240, G: 240, B: 240, A: 255}, aa)
		penX += float32(font.GlyphAdvance(gid)) * scale
	}
}

func drawGlyph(img *image.NRGBA, atlas []byte, atlasW int, p raster.PackedGlyph, dstX, dstY int, col color.NRGBA, aa bool) {
	if p.W == 0 || p.H == 0 {
		return
	}
	for gy := 0; gy < p.H; gy++ {
		y := dstY + gy
		if y < img.Rect.Min.Y || y >= img.Rect.Max.Y {
			continue
		}
		dstRow := img.Pix[(y-img.Rect.Min.Y)*img.Stride:]
		srcRow := atlas[(p.Y+gy)*atlasW+p.X:]
		for gx := 0; gx < p.W; gx++ {
			x := dstX + gx
			if x < img.Rect.Min.X || x >= img.Rect.Max.X {
				continue
			}
			a := srcRow[gx]
			if a == 0 {
				continue
			}
			if !aa {
				if a < 128 {
					continue
				}
				a = 255
			}
			off := x * 4
			dstRow[off+0] = uint8((uint16(col.R)*uint16(a) + 127) / 255)
			dstRow[off+1] = uint8((uint16(col.G)*uint16(a) + 127) / 255)
			dstRow[off+2] = uint8((uint16(col.B)*uint16(a) + 127) / 255)
			dstRow[off+3] = 255
		}
	}
}
