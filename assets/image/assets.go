package assets

import (
	"bytes"
	_ "embed"
	"image"
	_ "golang.org/x/image/webp"
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
	//go:embed grape.webp
	grapeWebP []byte
	//go:embed mandarin.webp
	mandarinWebP []byte
	//go:embed apple.webp
	appleWebP []byte
	//go:embed pear.webp
	pearWebP []byte
	//go:embed peach.webp
	peachWebP []byte
	//go:embed pineapple.webp
	pineappleWebP []byte
	//go:embed melon.webp
	melonWebP []byte
	//go:embed watermelon.webp
	watermelonWebP []byte

	//go:embed speaker.webp
	speakerWebP []byte
	//go:embed muted.webp
	mutedWebP []byte
	//go:embed share.webp
	shareWebP []byte
	//go:embed titlelogo.webp
	titlelogoWebP []byte

	assets map[Kind]ImageSet
	icons  map[IconKind]*ebiten.Image
)

type IconKind int

const (
	Speaker IconKind = iota
	Muted
	Share
	TitleLogo
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
	grapeImg, grapeEbiten := decodeImage(grapeWebP)
	mandarinImg, mandarinEbiten := decodeImage(mandarinWebP)
	appleImg, appleEbiten := decodeImage(appleWebP)
	pearImg, pearEbiten := decodeImage(pearWebP)
	peachImg, peachEbiten := decodeImage(peachWebP)
	pineappleImg, pineappleEbiten := decodeImage(pineappleWebP)
	melonImg, melonEbiten := decodeImage(melonWebP)
	watermelonImg, watermelonEbiten := decodeImage(watermelonWebP)

	assets = map[Kind]ImageSet{
		Grape:      newImageSet(grapeImg, grapeEbiten, 1.0, 10),
		Mandarin:   newImageSet(mandarinImg, mandarinEbiten, 1.0, 20),
		Apple:      newImageSet(appleImg, appleEbiten, 1.0, 60),
		Pear:       newImageSet(pearImg, pearEbiten, 1.0, 70),
		Peach:      newImageSet(peachImg, peachEbiten, 1.0, 80),
		Pineapple:  newImageSet(pineappleImg, pineappleEbiten, 1.0, 90),
		Melon:      newImageSet(melonImg, melonEbiten, 1.0, 100),
		Watermelon: newImageSet(watermelonImg, watermelonEbiten, 1.0, 110),
	}

	_, speakerEbiten := decodeImage(speakerWebP)
	_, mutedEbiten := decodeImage(mutedWebP)
	_, shareEbiten := decodeImage(shareWebP)
	_, titleLogoEbiten := decodeImage(titlelogoWebP)

	icons = map[IconKind]*ebiten.Image{
		Speaker:   speakerEbiten,
		Muted:     mutedEbiten,
		Share:     shareEbiten,
		TitleLogo: titleLogoEbiten,
	}
}

func decodeImage(data []byte) (image.Image, *ebiten.Image) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	return img, ebiten.NewImageFromImage(img)
}

func Get(kind Kind) ImageSet {
	imageSet, ok := assets[kind]
	if !ok {
		log.Fatalf("image %d not found", kind)
	}
	return imageSet
}

func GetIcon(kind IconKind) *ebiten.Image {
	icon, ok := icons[kind]
	if !ok {
		log.Fatalf("icon %d not found", kind)
	}
	return icon
}

func Length() int {
	return len(assets)
}

func Exists(kind Kind) bool {
	return kind >= Min && kind <= Max
}

func ForEach(fn func(Kind, ImageSet)) {
	for k, v := range assets {
		fn(k, v)
	}
}

func newImageSet(img image.Image, ebitenImg *ebiten.Image, scale float64, score int) ImageSet {
	return ImageSet{
		EbitenImage: ebitenImg,
		Image:       img,
		Scale:       scale,
		Vectors:     generateVectors(img, scale),
		Score:       score,
	}
}

func generateVectors(img image.Image, scale float64) []cp.Vector {
	bounds := img.Bounds()
	bb := cp.BB{
		L: float64(bounds.Min.X),
		B: float64(bounds.Min.Y),
		R: float64(bounds.Max.X),
		T: float64(bounds.Max.Y),
	}

	sampleFunc := func(point cp.Vector) float64 {
		x, y := int(point.X), int(point.Y)
		rect := img.Bounds()

		if x < rect.Min.X || x > rect.Max.X || y < rect.Min.Y || y > rect.Max.Y {
			return 0.0
		}
		_, _, _, a := img.At(x, y).RGBA()
		return float64(a) / 0xffff
	}

	lineSet := cp.MarchSoft(bb, 300, 300, 0.5, cp.PolyLineCollectSegment, sampleFunc)
	line := lineSet.Lines[0].SimplifyCurves(.9)

	offset := cp.Vector{
		X: float64(bounds.Max.X-bounds.Min.X) / 2.0,
		Y: float64(bounds.Max.Y-bounds.Min.Y) / 2.0,
	}

	for i, vertex := range line.Verts {
		line.Verts[i] = vertex.Sub(offset).Mult(scale)
	}

	return line.Verts
}
