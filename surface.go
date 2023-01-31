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
	s.s.TopLevel().SendClose()
	return nil
}

func (s *viewSurfaceXDG) Title() string {
	return s.s.TopLevel().Title()
}

func (s *viewSurfaceXDG) Resize(w, h int) {
	s.s.TopLevel().SetSize(int32(w), int32(h))
}

func (s *viewSurfaceXDG) SetResizing(resizing bool) {
	s.s.TopLevel().SetResizing(resizing)
}

func (s *viewSurfaceXDG) SetMinimized(m bool) {
	// Apparently XDG clients can't be minimized. Huh.
}

func (s *viewSurfaceXDG) SetMaximized(m bool) {
	s.s.TopLevel().SetMaximized(m)
}

func (s *viewSurfaceXDG) Geometry() geom.Rect[int] {
	return geom.FromImageRect(s.s.GetGeometry())
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
	s.s.TopLevel().SetActivated(a)
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
	return s.s.Surface().HasSurface(surface)
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

func (s *viewSurfaceXWayland) Geometry() geom.Rect[int] {
	return geom.Rt(0, 0, s.s.Width(), s.s.Height())
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
