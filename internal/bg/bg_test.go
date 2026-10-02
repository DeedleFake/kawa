package bg_test

import (
	"testing"

	"deedles.dev/kawa/internal/bg"
	"deedles.dev/ximage/geom"
)

func TestPlace(t *testing.T) {
	area := geom.Rt[float64](0, 25, 1920, 1080)
	tests := []struct {
		name  string
		scale bg.Scale
		img   geom.Rect[float64]
		want  geom.Rect[float64]
	}{
		{"stretch", bg.Stretch, geom.Rt[float64](0, 0, 200, 100), area},
		{"center small", bg.Center, geom.Rt[float64](0, 0, 200, 100), geom.Rt(860, 502.5, 1060, 602.5)},
		{"center large", bg.Center, geom.Rt[float64](0, 0, 3840, 2160), geom.Rt(-960, -527.5, 2880, 1632.5)},
		{"fit small", bg.Fit, geom.Rt[float64](0, 0, 200, 100), geom.Rt[float64](0, 0, 200, 100)},
		{"fit wide", bg.Fit, geom.Rt[float64](0, 0, 3840, 1080), geom.Rt(0, 282.5, 1920, 822.5)},
		{"fit as wide as area", bg.Fit, geom.Rt[float64](0, 0, 1920, 100), geom.Rt(0, 502.5, 1920, 602.5)},
		{"fill small", bg.Fill, geom.Rt[float64](0, 0, 200, 100), geom.Rt(0, 72.5, 1920, 1032.5)},
		{"fill tall", bg.Fill, geom.Rt[float64](0, 0, 1000, 2000), geom.Rt(696.25, 25, 1223.75, 1080)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.scale.Place(area, test.img)
			if got != test.want {
				t.Errorf("%v.Place(%v, %v) = %v, want %v", test.scale, area, test.img, got, test.want)
			}
		})
	}
}

func TestText(t *testing.T) {
	for _, s := range []bg.Scale{bg.Stretch, bg.Center, bg.Fit, bg.Fill} {
		text, err := s.MarshalText()
		if err != nil {
			t.Fatalf("marshal %v: %v", s, err)
		}

		var got bg.Scale
		err = got.UnmarshalText(text)
		if err != nil {
			t.Fatalf("unmarshal %q: %v", text, err)
		}
		if got != s {
			t.Errorf("round trip of %v through %q gave %v", s, text, got)
		}
	}

	var s bg.Scale
	err := s.UnmarshalText([]byte("tile"))
	if want := `unknown scaling method: "tile"`; (err == nil) || (err.Error() != want) {
		t.Errorf("unmarshal tile: got error %v, want %v", err, want)
	}
}
