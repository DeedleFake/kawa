package kawa

import (
	"testing"

	"deedles.dev/ximage/geom"
)

func TestNewBoxShow(t *testing.T) {
	server := &Server{newViews: map[int]geom.Rect[float64]{42: {}}}
	m := &newBox{box: geom.Rt[float64](10, 20, 300, 200)}

	m.show(server)
	if want := geom.Rt[float64](10, 20, 300, 200); server.overlay.box != want {
		t.Fatalf("before start: overlay box = %v, want %v", server.overlay.box, want)
	}

	m.started, m.pid = true, 42
	m.box.Max = geom.Pt[float64](400, 250)
	m.show(server)
	if !server.overlay.box.IsZero() {
		t.Errorf("after start: overlay box = %v, want none", server.overlay.box)
	}
	if want := geom.Rt[float64](10, 20, 400, 250); server.newViews[42] != want {
		t.Errorf("after start: New box = %v, want %v", server.newViews[42], want)
	}

	delete(server.newViews, 42)
	m.show(server)
	if len(server.newViews) != 0 {
		t.Errorf("after exit: New boxes = %v, want none", server.newViews)
	}
}
