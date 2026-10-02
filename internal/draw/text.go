package draw

import (
	"fmt"
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var (
	fontOptions = opentype.FaceOptions{
		Size: 14,
		DPI:  72,
	}

	gomonoFont *sfnt.Font
	gomonoFace font.Face
)

func init() {
	var err error
	gomonoFont, err = opentype.Parse(gomono.TTF)
	if err != nil {
		panic(fmt.Errorf("parse font: %w", err))
	}

	gomonoFace, err = opentype.NewFace(gomonoFont, &fontOptions)
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}
}

// Text draws str in src on a new transparent image that is as wide as
// the text and as tall as the font's size.
func Text(src image.Image, str string) *image.NRGBA {
	fdraw := font.Drawer{
		Src:  src,
		Face: gomonoFace,
		Dot:  fixed.P(0, int(fontOptions.Size)),
	}

	extents, _ := fdraw.BoundString(str)
	buf := image.NewNRGBA(image.Rect(
		0,
		0,
		(extents.Max.X - extents.Min.X).Floor(),
		int(fontOptions.Size),
	))
	fdraw.Dst = buf
	fdraw.DrawString(str)

	return buf
}
