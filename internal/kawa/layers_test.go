package kawa

import (
	"testing"

	"deedles.dev/kawa/internal/anchor"
	"deedles.dev/wlr"
)

func TestAnchorEdges(t *testing.T) {
	tests := []struct {
		edge   anchor.Edges
		anchor wlr.LayerSurfaceV1Anchor
		wlr    wlr.Edges
	}{
		{anchor.Top, wlr.LayerSurfaceV1AnchorTop, wlr.EdgeTop},
		{anchor.Bottom, wlr.LayerSurfaceV1AnchorBottom, wlr.EdgeBottom},
		{anchor.Left, wlr.LayerSurfaceV1AnchorLeft, wlr.EdgeLeft},
		{anchor.Right, wlr.LayerSurfaceV1AnchorRight, wlr.EdgeRight},
	}
	for _, test := range tests {
		if anchor.Edges(test.anchor) != test.edge {
			t.Errorf("layer surface anchor %d is not anchor.Edges %d", test.anchor, test.edge)
		}
		if anchor.Edges(test.wlr) != test.edge {
			t.Errorf("wlr.Edges %d is not anchor.Edges %d", test.wlr, test.edge)
		}
	}
	if anchor.Edges(wlr.EdgeNone) != 0 {
		t.Errorf("wlr.EdgeNone is %d", wlr.EdgeNone)
	}
}
