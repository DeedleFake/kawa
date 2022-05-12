package main

import (
	"io"

	"deedles.dev/wlr"
)

type ViewSurface interface {
	io.Closer

	Title() string
	PID() int
	Surface() wlr.Surface
	Resize(w, h int)
	SetResizing(bool)

	Mapped() bool
	Activate(bool)
	Activated() bool

	ForEachSurface(func(wlr.Surface, int, int))
	SurfaceAt(x, y float64) (s wlr.Surface, sx, sy float64, ok bool)
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
	s.ForEachSurface(func(s wlr.Surface, x, y int) {
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

func (s *viewSurfaceXDG) Surface() wlr.Surface {
	return s.s.Surface()
}

func (s *viewSurfaceXDG) Mapped() bool {
	return s.s.Mapped()
}

func (s *viewSurfaceXDG) Activate(a bool) {
	s.s.TopLevelSetActivated(a)
}

func (s *viewSurfaceXDG) Activated() bool {
	return s.s.TopLevel().Current().Activated()
}

func (s *viewSurfaceXDG) ForEachSurface(cb func(wlr.Surface, int, int)) {
	s.s.ForEachSurface(cb)
}

func (s *viewSurfaceXDG) SurfaceAt(x, y float64) (surface wlr.Surface, sx, sy float64, ok bool) {
	return s.s.SurfaceAt(x, y)
}

type viewSurfaceXWayland struct {
	s         wlr.XWaylandSurface
	activated bool
}

func (s *viewSurfaceXWayland) PID() int {
	return -1 // There doesn't seem to be a way to get this...
}

func (s *viewSurfaceXWayland) HasSurface(surface wlr.Surface) (has bool) {
	s.ForEachSurface(func(s wlr.Surface, x, y int) {
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

func (s *viewSurfaceXWayland) Surface() wlr.Surface {
	return s.s.Surface()
}

func (s *viewSurfaceXWayland) Mapped() bool {
	return s.s.Mapped()
}

func (s *viewSurfaceXWayland) Activate(a bool) {
	s.s.Activate(a)
	s.activated = a
}

func (s *viewSurfaceXWayland) Activated() bool {
	return s.activated
}

func (s *viewSurfaceXWayland) ForEachSurface(cb func(wlr.Surface, int, int)) {
	s.s.Surface().ForEachSurface(cb)
}

func (s *viewSurfaceXWayland) SurfaceAt(x, y float64) (surface wlr.Surface, sx, sy float64, ok bool) {
	return s.s.Surface().SurfaceAt(x, y)
}
