package kawa

import (
	"testing"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

func TestBorderEdges(t *testing.T) {
	r := geom.Rt[float64](100, 100, 200, 150)
	tests := []struct {
		name string
		p    geom.Point[float64]
		want wlr.Edges
	}{
		{"left", geom.Pt[float64](97, 120), wlr.EdgeLeft},
		{"top", geom.Pt[float64](150, 97), wlr.EdgeTop},
		{"right", geom.Pt[float64](202, 120), wlr.EdgeRight},
		{"bottom", geom.Pt[float64](150, 152), wlr.EdgeBottom},
		{"top left", geom.Pt[float64](97, 97), wlr.EdgeTop | wlr.EdgeLeft},
		{"top right", geom.Pt[float64](202, 97), wlr.EdgeTop | wlr.EdgeRight},
		{"bottom left", geom.Pt[float64](97, 152), wlr.EdgeBottom | wlr.EdgeLeft},
		{"bottom right", geom.Pt[float64](202, 152), wlr.EdgeBottom | wlr.EdgeRight},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := borderEdges(r, test.p); got != test.want {
				t.Errorf("borderEdges(%v, %v) = %v, want %v", r, test.p, got, test.want)
			}
		})
	}
}
