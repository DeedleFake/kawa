package anchor_test

import (
	"testing"

	"deedles.dev/kawa/internal/anchor"
	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

func TestPlace(t *testing.T) {
	full := geom.Rt(0, 0, 1024, 768)
	usable := geom.Rt(0, 25, 1024, 768)
	tests := []struct {
		name  string
		state anchor.State
		want  geom.Rect[int]
	}{
		{
			"stretched wallpaper",
			anchor.State{Anchor: wlr.EdgeTop | wlr.EdgeBottom | wlr.EdgeLeft | wlr.EdgeRight, Zone: -1},
			full,
		},
		{
			"top panel",
			anchor.State{Anchor: wlr.EdgeTop | wlr.EdgeLeft | wlr.EdgeRight, Height: 30, Zone: 30},
			geom.Rt(0, 25, 1024, 55),
		},
		{
			"bottom panel with margins",
			anchor.State{Anchor: wlr.EdgeBottom | wlr.EdgeLeft | wlr.EdgeRight, Height: 30, Margin: anchor.Margins{Bottom: 5, Left: 10, Right: 20}},
			geom.Rt(10, 733, 1004, 763),
		},
		{
			"left dock",
			anchor.State{Anchor: wlr.EdgeLeft, Width: 64, Height: 400, Margin: anchor.Margins{Left: 4}},
			geom.Rt(4, 196, 68, 596),
		},
		{
			"right dock",
			anchor.State{Anchor: wlr.EdgeRight, Width: 64, Height: 400, Margin: anchor.Margins{Right: 4}},
			geom.Rt(956, 196, 1020, 596),
		},
		{
			"centered",
			anchor.State{Width: 300, Height: 200},
			geom.Rt(362, 296, 662, 496),
		},
		{
			"anchored to both sides with a size",
			anchor.State{Anchor: wlr.EdgeLeft | wlr.EdgeRight, Width: 300, Height: 200},
			geom.Rt(362, 296, 662, 496),
		},
		{
			"margins wider than the area",
			anchor.State{Anchor: wlr.EdgeLeft | wlr.EdgeRight, Height: 10, Margin: anchor.Margins{Left: 800, Right: 800}},
			geom.Rt(800, 391, 800, 401),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := anchor.Place(test.state, full, usable)
			if got != test.want {
				t.Errorf("Place(%+v) = %v, want %v", test.state, got, test.want)
			}
		})
	}
}

func TestExclude(t *testing.T) {
	usable := geom.Rt(0, 25, 1024, 768)
	state := anchor.State{Zone: 30, Margin: anchor.Margins{Top: 1, Right: 2, Bottom: 3, Left: 4}}
	tests := []struct {
		edge wlr.Edges
		want geom.Rect[int]
	}{
		{0, usable},
		{wlr.EdgeTop, geom.Rt(0, 56, 1024, 768)},
		{wlr.EdgeBottom, geom.Rt(0, 25, 1024, 735)},
		{wlr.EdgeLeft, geom.Rt(34, 25, 1024, 768)},
		{wlr.EdgeRight, geom.Rt(0, 25, 992, 768)},
		{wlr.EdgeTop | wlr.EdgeLeft, usable},
	}
	for _, test := range tests {
		got := anchor.Exclude(usable, state, test.edge)
		if got != test.want {
			t.Errorf("Exclude(%v, %+v, %v) = %v, want %v", usable, state, test.edge, got, test.want)
		}
	}
}
