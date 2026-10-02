package output_test

import (
	"reflect"
	"testing"

	"deedles.dev/kawa/internal/output"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want []output.Config
	}{
		{"", nil},
		{"X11-1:0:0", []output.Config{{Name: "X11-1"}}},
		{"DP-1:-1:-1", []output.Config{{Name: "DP-1", X: -1, Y: -1}}},
		{"DP-1:1920:0:2560:1440", []output.Config{{Name: "DP-1", X: 1920, Width: 2560, Height: 1440}}},
		{"DP-1:0:0:1920", []output.Config{{Name: "DP-1"}}},
		{"DP-1:0:0:0:0:1.5", []output.Config{{Name: "DP-1", Scale: 1.5}}},
		{"DP-1:0:0:0:0:2:flipped-90", []output.Config{{Name: "DP-1", Scale: 2, Transform: output.Flipped90}}},
		{"DP-1:0:0:0:0:0:normal", []output.Config{{Name: "DP-1"}}},
		{"DP-1:0:0:0:0:0:sideways", []output.Config{{Name: "DP-1"}}},
		{"DP-1:a:10:b:c:d", []output.Config{{Name: "DP-1", Y: 10}}},
		{
			"DP-1:0:0,HDMI-A-1:1920:0:1280:1024:1:270",
			[]output.Config{
				{Name: "DP-1"},
				{Name: "HDMI-A-1", X: 1920, Width: 1280, Height: 1024, Scale: 1, Transform: output.Rotate270},
			},
		},
	}
	for _, test := range tests {
		got := output.Parse(test.in)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("Parse(%q) = %+v, want %+v", test.in, got, test.want)
		}
	}
}

func TestParseTransforms(t *testing.T) {
	transforms := map[string]output.Transform{
		"normal":      output.Normal,
		"0":           output.Normal,
		"90":          output.Rotate90,
		"180":         output.Rotate180,
		"270":         output.Rotate270,
		"flipped":     output.Flipped,
		"flipped-90":  output.Flipped90,
		"flipped-180": output.Flipped180,
		"flipped-270": output.Flipped270,
	}
	for name, want := range transforms {
		got := output.Parse("DP-1:0:0:0:0:0:" + name)[0].Transform
		if got != want {
			t.Errorf("transform %q = %v, want %v", name, got, want)
		}
	}
}
