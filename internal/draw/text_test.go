package draw_test

import (
	"image"
	"image/color"
	"testing"

	"deedles.dev/kawa/internal/draw"
)

func TestText(t *testing.T) {
	img := draw.Text(image.NewUniform(color.NRGBA{0x3D, 0x7D, 0x42, 0xFF}), "Log Out")
	if b := img.Bounds(); (b.Min != image.Point{}) || (b.Dy() != 14) || (b.Dx() < 50) {
		t.Fatalf("bounds = %v, want 0,0 at the corner, at least 50 wide, and 14 tall", b)
	}

	var opaque int
	for i := 0; i < len(img.Pix); i += 4 {
		px := img.Pix[i : i+4]
		if px[3] != 0xFF {
			continue
		}
		opaque++
		if (px[0] != 0x3D) || (px[1] != 0x7D) || (px[2] != 0x42) {
			t.Fatalf("opaque pixel %v is not the source color", px)
		}
	}
	if opaque == 0 {
		t.Fatal("no opaque pixels")
	}
}

func TestTextEmpty(t *testing.T) {
	img := draw.Text(image.White, "")
	if b := img.Bounds(); b != image.Rect(0, 0, 0, 14) {
		t.Errorf("bounds = %v, want (0,0)-(0,14)", b)
	}
}

func TestTextWidth(t *testing.T) {
	short := draw.Text(image.White, "New").Bounds().Dx()
	long := draw.Text(image.White, "NewNew").Bounds().Dx()
	if long <= short {
		t.Errorf("NewNew is %d wide, New is %d", long, short)
	}
}
