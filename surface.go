package main

import (
	"io"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

type ViewSurface interface {
	io.Closer

	Title() string
	PID() int
	Surface() wlr.Surface
	SetResizing(bool)
	SetMinimized(bool)
	SetMaximized(bool)

	Resize(w, h int)
	Geometry() geom.Rect[int]
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
	return s.s.HasSurface(surface)
}

func (s *viewSurfaceXDG) Close() error {
	s.s.Toplevel().SendClose()
	return nil
}

func (s *viewSurfaceXDG) Title() string {
	return s.s.Toplevel().Title()
}

func (s *viewSurfaceXDG) Resize(w, h int) {
	s.s.Toplevel().SetSize(int32(w), int32(h))
}

func (s *viewSurfaceXDG) SetResizing(resizing bool) {
	s.s.Toplevel().SetResizing(resizing)
}

func (s *viewSurfaceXDG) SetMinimized(m bool) {
	// Apparently XDG clients can't be minimized. Huh.
}

func (s *viewSurfaceXDG) SetMaximized(m bool) {
	s.s.Toplevel().SetMaximized(m)
}

func (s *viewSurfaceXDG) Geometry() geom.Rect[int] {
	return geom.FromImageRect(s.s.GetGeometry())
}

func (s *viewSurfaceXDG) MinWidth() float64 {
	return float64(s.s.Toplevel().Current().MinWidth())
}

func (s *viewSurfaceXDG) MinHeight() float64 {
	return float64(s.s.Toplevel().Current().MinHeight())
}

func (s *viewSurfaceXDG) Surface() wlr.Surface {
	return s.s.Surface()
}

func (s *viewSurfaceXDG) Mapped() bool {
	return s.s.Mapped()
}

func (s *viewSurfaceXDG) SetActivated(a bool) {
	s.s.Toplevel().SetActivated(a)
}

func (s *viewSurfaceXDG) Activated() bool {
	return s.s.Toplevel().Current().Activated()
}

func (s *viewSurfaceXDG) ForEachSurface(cb func(wlr.Surface, int, int)) {
	s.s.ForEachSurface(cb)
}

func (s *viewSurfaceXDG) SurfaceAt(p geom.Point[float64]) (surface wlr.Surface, sp geom.Point[float64], ok bool) {
	surface, sx, sy, ok := s.s.SurfaceAt(p.X, p.Y)
	return surface, geom.Pt(sx, sy), ok
}

type viewSurfaceXwayland struct {
	s         wlr.XwaylandSurface
	activated bool
}

func (s *viewSurfaceXwayland) PID() int {
	return -1 // There doesn't seem to be a way to get this...
}

func (s *viewSurfaceXwayland) HasSurface(surface wlr.Surface) (has bool) {
	return s.s.Surface().HasSurface(surface)
}

func (s *viewSurfaceXwayland) Close() error {
	s.s.Close()
	return nil
}

func (s *viewSurfaceXwayland) Title() string {
	return s.s.Title()
}

func (s *viewSurfaceXwayland) Resize(w, h int) {
	s.s.Configure(0, 0, uint16(w), uint16(h))
}

func (s *viewSurfaceXwayland) SetResizing(resizing bool) {
	// Doesn't make sense for Xwayland clients, it seems.
}

func (s *viewSurfaceXwayland) SetMinimized(m bool) {
	s.s.SetMinimized(m)
}

func (s *viewSurfaceXwayland) SetMaximized(m bool) {
	s.s.SetMaximized(m)
}

func (s *viewSurfaceXwayland) Geometry() geom.Rect[int] {
	return geom.Rt(0, 0, s.s.Width(), s.s.Height())
}

func (s *viewSurfaceXwayland) MinWidth() float64 {
	return MinWidth
}

func (s *viewSurfaceXwayland) MinHeight() float64 {
	return MinHeight
}

func (s *viewSurfaceXwayland) Surface() wlr.Surface {
	return s.s.Surface()
}

func (s *viewSurfaceXwayland) Mapped() bool {
	return s.s.Mapped()
}

func (s *viewSurfaceXwayland) SetActivated(a bool) {
	s.s.Activate(a)
	s.activated = a
}

func (s *viewSurfaceXwayland) Activated() bool {
	return s.activated
}

func (s *viewSurfaceXwayland) ForEachSurface(cb func(wlr.Surface, int, int)) {
	s.s.Surface().ForEachSurface(cb)
}

func (s *viewSurfaceXwayland) SurfaceAt(p geom.Point[float64]) (surface wlr.Surface, sp geom.Point[float64], ok bool) {
	surface, sx, sy, ok := s.s.Surface().SurfaceAt(p.X, p.Y)
	return surface, geom.Pt(sx, sy), ok
}
