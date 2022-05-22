package main

import (
	"io"

	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
)

type ViewSurface interface {
	io.Closer

	Title() string
	PID() int
	Surface() wlr.Surface
	Resize(w, h int)
	SetResizing(bool)
	SetMinimized(bool)
	SetMaximized(bool)

	MinWidth() float64
	MinHeight() float64

	Mapped() bool
	Activated() bool
	SetActivated(bool)

	ForEachSurface(func(wlr.Surface, int, int))
	SurfaceAt(geom.Point[float64]) (s wlr.Surface, sp geom.Point[float64], ok bool)
	HasSurface(wlr.Surface) bool
}

type viewSurfaceXDG struct {
	s wlr.XDGSurface
}

func (s *viewSurfaceXDG) PID() int {
	client := s.s.Resource().GetClient()
	pid, _, _ := client.GetCredentials()
	return pid
}

func (s *viewSurfaceXDG) HasSurface(surface wlr.Surface) (has bool) {
	s.ForEachSurface((s, x, y) => {
		if s == surface {
			has = true
		}
	})
	return has
}

func (s *viewSurfaceXDG) Close() error {
	s.s.SendClose()
	return nil
}

func (s *viewSurfaceXDG) Title() string {
	return s.s.TopLevel().Title()
}

func (s *viewSurfaceXDG) Resize(w, h int) {
	s.s.TopLevelSetSize(uint32(w), uint32(h))
}

func (s *viewSurfaceXDG) SetResizing(resizing bool) {
	s.s.TopLevelSetResizing(resizing)
}

func (s *viewSurfaceXDG) SetMinimized(m bool) {
	// Apparently XDG clients can't be minimized. Huh.
}

func (s *viewSurfaceXDG) SetMaximized(m bool) {
	s.s.TopLevelSetMaximized(m)
}

func (s *viewSurfaceXDG) MinWidth() float64 {
	return float64(s.s.TopLevel().Current().MinWidth())
}

func (s *viewSurfaceXDG) MinHeight() float64 {
	return float64(s.s.TopLevel().Current().MinHeight())
}

func (s *viewSurfaceXDG) Surface() wlr.Surface {
	return s.s.Surface()
}

func (s *viewSurfaceXDG) Mapped() bool {
	return s.s.Mapped()
}

func (s *viewSurfaceXDG) SetActivated(a bool) {
	s.s.TopLevelSetActivated(a)
}

func (s *viewSurfaceXDG) Activated() bool {
	return s.s.TopLevel().Current().Activated()
}

func (s *viewSurfaceXDG) ForEachSurface(cb func(wlr.Surface, int, int)) {
	s.s.ForEachSurface(cb)
}

func (s *viewSurfaceXDG) SurfaceAt(p geom.Point[float64]) (surface wlr.Surface, sp geom.Point[float64], ok bool) {
	surface, sx, sy, ok := s.s.SurfaceAt(p.X, p.Y)
	return surface, geom.Pt(sx, sy), ok
}

type viewSurfaceXWayland struct {
	s         wlr.XWaylandSurface
	activated bool
}

func (s *viewSurfaceXWayland) PID() int {
	return -1 // There doesn't seem to be a way to get this...
}

func (s *viewSurfaceXWayland) HasSurface(surface wlr.Surface) (has bool) {
	s.ForEachSurface((s, x, y) => {
		if s == surface {
			has = true
		}
	})
	return has
}

func (s *viewSurfaceXWayland) Close() error {
	s.s.Close()
	return nil
}

func (s *viewSurfaceXWayland) Title() string {
	return s.s.Title()
}

func (s *viewSurfaceXWayland) Resize(w, h int) {
	s.s.Configure(0, 0, uint16(w), uint16(h))
}

func (s *viewSurfaceXWayland) SetResizing(resizing bool) {
	// Doesn't make sense for XWayland clients, it seems.
}

func (s *viewSurfaceXWayland) SetMinimized(m bool) {
	s.s.SetMinimized(m)
}

func (s *viewSurfaceXWayland) SetMaximized(m bool) {
	s.s.SetMaximized(m)
}

func (s *viewSurfaceXWayland) MinWidth() float64 {
	return MinWidth
}

func (s *viewSurfaceXWayland) MinHeight() float64 {
	return MinHeight
}

func (s *viewSurfaceXWayland) Surface() wlr.Surface {
	return s.s.Surface()
}

func (s *viewSurfaceXWayland) Mapped() bool {
	return s.s.Mapped()
}

func (s *viewSurfaceXWayland) SetActivated(a bool) {
	s.s.Activate(a)
	s.activated = a
}

func (s *viewSurfaceXWayland) Activated() bool {
	return s.activated
}

func (s *viewSurfaceXWayland) ForEachSurface(cb func(wlr.Surface, int, int)) {
	s.s.Surface().ForEachSurface(cb)
}

func (s *viewSurfaceXWayland) SurfaceAt(p geom.Point[float64]) (surface wlr.Surface, sp geom.Point[float64], ok bool) {
	surface, sx, sy, ok := s.s.Surface().SurfaceAt(p.X, p.Y)
	return surface, geom.Pt(sx, sy), ok
}
