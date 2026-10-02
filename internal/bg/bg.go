// Package bg places a background image in the area that it covers.
package bg

import (
	"fmt"
	"slices"

	"deedles.dev/ximage/geom"
)

// Scale is a way to fit a background image to an area.
type Scale int

const (
	// Stretch covers the area exactly, whatever the image's aspect
	// ratio.
	Stretch Scale = iota
	// Center keeps the image's size and centers it in the area.
	Center
	// Fit leaves an image that is smaller than the area on both axes
	// as it is, and places any other image like Fill.
	Fit
	// Fill scales the image to the largest size with the same aspect
	// ratio that fits in the area and centers it.
	Fill
)

var names = [...]string{
	Stretch: "stretch",
	Center:  "center",
	Fit:     "fit",
	Fill:    "fill",
}

// Place returns where to draw an image with the bounds img in area.
func (s Scale) Place(area, img geom.Rect[float64]) geom.Rect[float64] {
	switch s {
	case Stretch:
		return area
	case Center:
		return img.CenterAt(area.Center())
	case Fit:
		if (img.Dx() < area.Dx()) && (img.Dy() < area.Dy()) {
			return img
		}
		return Fill.Place(area, img)
	case Fill:
		return Center.Place(area, img.FitTo(area.Size()))
	default:
		panic(fmt.Errorf("invalid scaling method: %v", s))
	}
}

func (s Scale) String() string {
	if (s < 0) || (int(s) >= len(names)) {
		return fmt.Sprintf("Scale(%d)", int(s))
	}
	return names[s]
}

func (s Scale) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Scale) UnmarshalText(text []byte) error {
	i := slices.Index(names[:], string(text))
	if i < 0 {
		return fmt.Errorf("unknown scaling method: %q", text)
	}
	*s = Scale(i)
	return nil
}
