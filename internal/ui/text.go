package ui

import (
	"bytes"
	_ "embed"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed fonts/Poppins-Bold-subset.ttf
var poppinsBoldTTF []byte

//go:embed fonts/Poppins-Regular-subset.ttf
var poppinsRegularTTF []byte

var (
	poppinsBoldSource    *text.GoTextFaceSource
	poppinsRegularSource *text.GoTextFaceSource
)

func init() {
	var err error
	poppinsBoldSource, err = text.NewGoTextFaceSource(bytes.NewReader(poppinsBoldTTF))
	if err != nil {
		log.Fatal(err)
	}

	poppinsRegularSource, err = text.NewGoTextFaceSource(bytes.NewReader(poppinsRegularTTF))
	if err != nil {
		log.Fatal(err)
	}
}

func DrawTextCentered(screen *ebiten.Image, str string, size float64, x, y float64, clr color.Color, bold bool) {
	source := poppinsRegularSource
	if bold {
		source = poppinsBoldSource
	}

	face := &text.GoTextFace{
		Source: source,
		Size:   size,
	}

	textWidth, textHeight := text.Measure(str, face, 0)

	op := &text.DrawOptions{}
	op.GeoM.Translate(x-textWidth/2, y-textHeight/2)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, str, face, op)
}
