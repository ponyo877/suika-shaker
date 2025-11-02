package assets

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp/v2"
)

const (
	Grape Kind = 1 + iota
	Mandarin
	Apple
	Pear
	Peach
	Pineapple
	Melon
	Watermelon

	Min Kind = Grape
	Max Kind = Watermelon
)

var (
	//go:embed grape.png
	grape_png []byte
	//go:embed mandarin.png
	mandarin_png []byte
	//go:embed apple.png
	apple_png []byte
	//go:embed pear.png
	pear_png []byte
	//go:embed peach.png
	peach_png []byte
	//go:embed pineapple.png
	pineapple_png []byte
	//go:embed melon.png
	melon_png []byte
	//go:embed watermelon.png
	watermelon_png []byte

	assets map[Kind]ImageSet
)

type Kind int

func (k Kind) Next() (hasNext bool, next Kind) {
	if k < Max {
		return true, k + 1
	}
	return false, 0
}

func (k Kind) Score() int {
	return Get(k).Score
}

type ImageSet struct {
	EbitenImage *ebiten.Image
	Image       image.Image
	Scale       float64
	Vectors     []cp.Vector
	Score       int
}

func init() {
	grapePngImage, grapeImage := loadImage(grape_png)
	mandarinPngImage, mandarinImage := loadImage(mandarin_png)
	applePngImage, appleImage := loadImage(apple_png)
	pearPngImage, pearImage := loadImage(pear_png)
	peachPngImage, peachImage := loadImage(peach_png)
	pineapplePngImage, pineappleImage := loadImage(pineapple_png)
	melonPngImage, melonImage := loadImage(melon_png)
	watermelonPngImage, watermelonImage := loadImage(watermelon_png)

	assets = map[Kind]ImageSet{
		Grape:      makeImageSet(grapeImage, grapePngImage, 1.0, 10),
		Mandarin:   makeImageSet(mandarinImage, mandarinPngImage, 1.0, 20),
		Apple:      makeImageSet(appleImage, applePngImage, 1.0, 60),
		Pear:       makeImageSet(pearImage, pearPngImage, 1.2, 70),
		Peach:      makeImageSet(peachImage, peachPngImage, 1.2, 80),
		Pineapple:  makeImageSet(pineappleImage, pineapplePngImage, 1.2, 90),
		Melon:      makeImageSet(melonImage, melonPngImage, 1.2, 100),
		Watermelon: makeImageSet(watermelonImage, watermelonPngImage, 1.2, 110),
	}
}

func loadImage(b []byte) (image.Image, *ebiten.Image) {
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	origImage := ebiten.NewImageFromImage(img)

	s := origImage.Bounds().Size()
	ebitenImage := ebiten.NewImage(s.X, s.Y)
	op := &ebiten.DrawImageOptions{}
	ebitenImage.DrawImage(origImage, op)
	return img, ebitenImage
}

func Get(tp Kind) ImageSet {
	is, ok := assets[tp]
	if !ok {
		log.Fatalf("image %d not found", tp)
	}
	return is
}

func Length() int {
	return len(assets)
}

func Exists(tp Kind) bool {
	return tp >= Min && tp <= Max
}

func ForEach(f func(Kind, ImageSet)) {
	for i, v := range assets {
		f(i, v)
	}
}

func makeImageSet(
	ebitenImage *ebiten.Image,
	image image.Image,
	scale float64,
	score int,
) ImageSet {
	is := ImageSet{
		EbitenImage: ebitenImage,
		Image:       image,
		Scale:       scale,
		Vectors:     makeVector(image, scale),
		Score:       score,
	}
	return is
}

func makeVector(img image.Image, scale float64) []cp.Vector {
	b := img.Bounds()
	bb := cp.BB{L: float64(b.Min.X), B: float64(b.Min.Y), R: float64(b.Max.X), T: float64(b.Max.Y)}

	sampleFunc := func(point cp.Vector) float64 {
		x := point.X
		y := point.Y
		rect := img.Bounds()

		if x < float64(rect.Min.X) || x > float64(rect.Max.X) || y < float64(rect.Min.Y) || y > float64(rect.Max.Y) {
			return 0.0
		}
		_, _, _, a := img.At(int(x), int(y)).RGBA()
		return float64(a) / 0xffff
	}

	lineSet := cp.MarchSoft(bb, 300, 300, 0.5, cp.PolyLineCollectSegment, sampleFunc)

	line := lineSet.Lines[0].SimplifyCurves(.9)
	offset := cp.Vector{X: float64(b.Max.X-b.Min.X) / 2., Y: float64(b.Max.Y-b.Min.Y) / 2.}
	// center the verts on origin
	for i, l := range line.Verts {
		line.Verts[i] = l.Sub(offset).Mult(scale)
	}
	return line.Verts
}
