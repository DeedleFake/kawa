// Package output parses the configuration of outputs.
package output

import (
	"strconv"
	"strings"
)

// Transform is a rotation and flip of an output, in the order of the
// wl_output transform enum.
type Transform int

const (
	Normal Transform = iota
	Rotate90
	Rotate180
	Rotate270
	Flipped
	Flipped90
	Flipped180
	Flipped270
)

var transforms = map[string]Transform{
	"normal":      Normal,
	"0":           Normal,
	"90":          Rotate90,
	"180":         Rotate180,
	"270":         Rotate270,
	"flipped":     Flipped,
	"flipped-90":  Flipped90,
	"flipped-180": Flipped180,
	"flipped-270": Flipped270,
}

// Config configures the output named Name.
type Config struct {
	Name string
	// X and Y place the output in the layout. If both are -1, it is
	// placed automatically.
	X, Y int
	// Width and Height pick the output's mode of that size. If either
	// is 0 or the output has no such mode, its preferred mode is used.
	Width, Height int
	// Scale is the output's scale. 0 leaves it alone.
	Scale     float32
	Transform Transform
}

// Parse parses a comma-separated list of configs, each of the form
// name:x:y[:width:height][:scale][:transform].
func Parse(s string) []Config {
	if s == "" {
		return nil
	}

	// TODO: Handle errors.
	var configs []Config
	for config := range strings.SplitSeq(s, ",") {
		parts := strings.Split(config, ":")
		c := Config{Name: parts[0]}
		c.X, _ = strconv.Atoi(parts[1])
		c.Y, _ = strconv.Atoi(parts[2])
		if len(parts) >= 5 {
			c.Width, _ = strconv.Atoi(parts[3])
			c.Height, _ = strconv.Atoi(parts[4])
		}
		if len(parts) >= 6 {
			scale, _ := strconv.ParseFloat(parts[5], 32)
			c.Scale = float32(scale)
		}
		if len(parts) >= 7 {
			c.Transform = transforms[parts[6]]
		}
		configs = append(configs, c)
	}
	return configs
}
