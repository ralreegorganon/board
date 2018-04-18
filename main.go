package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

var input = flag.String("input", ".", "Input directory")
var output = flag.String("output", ".", "Output directory")
var width = 612
var height = 792
var tileSize = 256
var dpi = 72.0
var size = 32.0
var spacing = 1.0
var textFont *truetype.Font
var columns = int(math.Floor(float64(width) / float64(tileSize)))
var rows = int(math.Floor(float64(height) / float64(tileSize)))
var tilesPerPage = columns * rows
var columnGap = 10
var rowGap = 10
var borderSize = 5
var textHeightReserved = 50
var textPadY = 5
var textPadX = 10

func init() {
	fontBytes, err := Asset("TimesNewRoman.ttf")
	if err != nil {
		panic(err)
	}

	textFont, err = freetype.ParseFont(fontBytes)
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	files := []string{}

	err := filepath.Walk(*input, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fc := len(files)
	pages := int(math.Ceil(float64(fc) / float64(tilesPerPage)))

	for p := 0; p < pages; p++ {
		layout := image.NewRGBA(image.Rect(0, 0, width, height))
		ctx := freetype.NewContext()
		ctx.SetDPI(dpi)
		ctx.SetFont(textFont)
		ctx.SetFontSize(size)
		ctx.SetClip(layout.Bounds())
		ctx.SetDst(layout)
		ctx.SetSrc(image.Black)
		ctx.SetHinting(font.HintingNone)

		d := &font.Drawer{
			Dst: layout,
			Src: image.Black,
			Face: truetype.NewFace(textFont, &truetype.Options{
				Size:    size,
				DPI:     dpi,
				Hinting: font.HintingNone,
			}),
		}

		for r := 0; r < rows; r++ {
			for c := 0; c < columns; c++ {
				fi := p*rows*columns + r*columns + c
				if fi >= fc {
					break
				}

				eif, err := os.Open(files[fi])
				if err != nil {
					log.Fatal(err)
				}
				defer eif.Close()

				img, _, err := image.Decode(eif)
				if err != nil {
					log.Fatal(err)
				}

				resized := imaging.Fit(img, tileSize-borderSize*2, tileSize-borderSize*2-textHeightReserved, imaging.Lanczos)
				resizedBounds := resized.Bounds()
				resizedWidth := resizedBounds.Dx()
				resizedHeight := resizedBounds.Dy()

				centeringXOffset := int(math.Floor(float64(tileSize-resizedWidth)/2.0)) - borderSize
				centeringYOffset := int(math.Floor(float64(tileSize-resizedHeight-textHeightReserved)/2.0)) - borderSize

				borderRect := image.Rect(c*tileSize+c*columnGap, r*tileSize+r*rowGap, c*tileSize+tileSize+c*columnGap, r*tileSize+tileSize+r*rowGap)
				whiteFillRect := image.Rect(c*tileSize+borderSize+c*columnGap, r*tileSize+borderSize+r*rowGap, c*tileSize+tileSize-borderSize+c*columnGap, r*tileSize+tileSize-borderSize+r*rowGap)
				//imageOnlyRect := image.Rect(c*tileSize+borderSize+c*columnGap, r*tileSize+borderSize+r*rowGap, c*tileSize+tileSize-borderSize+c*columnGap, r*tileSize+tileSize-borderSize+r*rowGap-textHeightReserved)
				resizedRect := image.Rect(c*tileSize+borderSize+c*columnGap+centeringXOffset, r*tileSize+borderSize+r*rowGap+centeringYOffset, c*tileSize+tileSize-borderSize+c*columnGap-centeringXOffset, r*tileSize+tileSize-borderSize+r*rowGap-textHeightReserved-centeringYOffset)

				draw.Draw(layout, borderRect, image.Black, image.ZP, draw.Src)
				draw.Draw(layout, whiteFillRect, image.White, image.ZP, draw.Src)
				//draw.Draw(layout, imageOnlyRect, image.NewUniform(color.RGBA{0, 255, 0, 255}), resizedRect.Min, draw.Src)
				draw.Draw(layout, resizedRect, resized, image.ZP, draw.Src)

				basename := filepath.Base(files[fi])
				text := strings.TrimSuffix(basename, filepath.Ext(basename))

				advance := d.MeasureString(text)
				//fmt.Printf("%v %v\n", files[fi], advance)
				max := fixed.I(225)

				if advance > max {
					parts := strings.Split(text, " ")
					line := ""
					mod := 1.2
					textPadYAlt := -5
					ctx.SetFontSize(size / mod)
					x := c*tileSize + borderSize + c*columnGap + textPadX
					y := r*tileSize + tileSize - borderSize + r*rowGap - textHeightReserved + textPadYAlt + int(ctx.PointToFixed(size/mod)>>6)
					pt := freetype.Pt(x, y)

					for _, t := range parts {
						if len(line)+len(t) < 23 {
							line = line + t + " "
						} else {
							ctx.DrawString(line, pt)
							line = t + " "
							pt.Y += ctx.PointToFixed(size / mod)
						}
					}
					ctx.DrawString(line, pt)
					ctx.SetFontSize(size)
				} else {
					pt := freetype.Pt(c*tileSize+borderSize+c*columnGap+textPadX, r*tileSize+tileSize-borderSize+r*rowGap-textHeightReserved+textPadY+int(ctx.PointToFixed(size)>>6))
					ctx.DrawString(text, pt)
				}
			}
		}

		filename := filepath.Join(*output, fmt.Sprintf("board-%v.png", p))
		outFile, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer outFile.Close()

		b := bufio.NewWriter(outFile)
		err = png.Encode(b, layout)
		if err != nil {
			log.Fatal(err)
		}

		err = b.Flush()
		if err != nil {
			log.Fatal(err)
		}
	}
}
