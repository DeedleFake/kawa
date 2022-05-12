package main

import (
	"fmt"
	"image"

	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

type ViewTargeter interface {
	TargetView() *View
}

var edgeCursors = [...]string{
	wlr.EdgeNone:                   "",
	wlr.EdgeTop:                    "top_side",
	wlr.EdgeLeft:                   "left_side",
	wlr.EdgeRight:                  "right_side",
	wlr.EdgeBottom:                 "bottom_side",
	wlr.EdgeTop | wlr.EdgeLeft:     "top_left_corner",
	wlr.EdgeTop | wlr.EdgeRight:    "top_right_corner",
	wlr.EdgeBottom | wlr.EdgeLeft:  "bottom_left_corner",
	wlr.EdgeBottom | wlr.EdgeRight: "bottom_right_corner",
}

type View struct {
	ViewSurface
	X, Y int

	Map           wlr.Listener
	Destroy       wlr.Listener
	RequestMove   wlr.Listener
	RequestResize wlr.Listener
}

func (view *View) Release() {
	view.Destroy.Destroy()
	view.Map.Destroy()
	view.RequestMove.Destroy()
	view.RequestResize.Destroy()
}

type Popup struct {
	Surface wlr.XDGSurface

	Destroy wlr.Listener
}

func (p *Popup) Release() {
	p.Destroy.Destroy()
}

func (server *Server) viewBounds(out *Output, view *View) image.Rectangle {
	var r image.Rectangle
	view.ForEachSurface(func(s wlr.Surface, sx, sy int) {
		if server.isPopupSurface(s) {
			return
		}
		r = r.Union(server.surfaceBounds(out, s, view.X+sx, view.Y+sy))
	})
	return r
}

func (server *Server) surfaceBounds(out *Output, surface wlr.Surface, x, y int) image.Rectangle {
	var ox, oy float64
	scale := float32(1)
	if out != nil {
		ox, oy = server.outputLayout.OutputCoords(out.Output)
		scale = out.Output.Scale()
	}

	current := surface.Current()
	return box(
		int((ox+float64(x))*float64(scale)),
		int((oy+float64(y))*float64(scale)),
		int(float64(current.Width())*float64(scale)),
		int(float64(current.Height())*float64(scale)),
	)
}

func (server *Server) targetView() *View {
	m, ok := server.inputMode.(ViewTargeter)
	if !ok {
		return nil
	}

	return m.TargetView()
}

func (server *Server) viewAt(out *Output, x, y float64) (*View, wlr.Edges, wlr.Surface, float64, float64) {
	if out == nil {
		out = server.outputAt(x, y)
	}

	p := image.Pt(int(x), int(y))
	for i := len(server.views) - 1; i >= 0; i-- {
		view := server.views[i]
		if !view.Mapped() {
			continue
		}

		surface, sx, sy, ok := view.SurfaceAt(x-float64(view.X), y-float64(view.Y))
		if ok {
			return view, wlr.EdgeNone, surface, sx, sy
		}

		r := server.viewBounds(nil, view)
		if !p.In(r.Inset(-WindowBorder)) {
			continue
		}

		left := image.Rect(r.Min.X-WindowBorder, r.Min.Y, r.Max.X, r.Max.Y)
		if p.In(left) {
			return view, wlr.EdgeLeft, wlr.Surface{}, 0, 0
		}

		top := image.Rect(r.Min.X, r.Min.Y-WindowBorder, r.Max.X, r.Max.Y)
		if p.In(top) {
			return view, wlr.EdgeTop, wlr.Surface{}, 0, 0
		}

		right := image.Rect(r.Min.X, r.Min.Y, r.Max.X+WindowBorder, r.Max.Y)
		if p.In(right) {
			return view, wlr.EdgeRight, wlr.Surface{}, 0, 0
		}

		bottom := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y+WindowBorder)
		if p.In(bottom) {
			return view, wlr.EdgeBottom, wlr.Surface{}, 0, 0
		}

		if (p.X < r.Min.X) && (p.Y < r.Min.Y) {
			return view, wlr.EdgeTop | wlr.EdgeLeft, wlr.Surface{}, 0, 0
		}
		if (p.X >= r.Max.X) && (p.Y < r.Min.Y) {
			return view, wlr.EdgeTop | wlr.EdgeRight, wlr.Surface{}, 0, 0
		}
		if (p.X < r.Min.X) && (p.Y >= r.Max.Y) {
			return view, wlr.EdgeBottom | wlr.EdgeLeft, wlr.Surface{}, 0, 0
		}
		if (p.X >= r.Max.X) && (p.Y >= r.Max.Y) {
			return view, wlr.EdgeBottom | wlr.EdgeRight, wlr.Surface{}, 0, 0
		}

		// Where else could it possibly be if it gets to here?
		panic(fmt.Errorf("If you see this, there's a bug.\np = %+v\nr = %+v", p, r))
	}

	return nil, wlr.EdgeNone, wlr.Surface{}, 0, 0
}

func (server *Server) onNewXWaylandSurface(surface wlr.XWaylandSurface) {
	view := View{
		ViewSurface: &viewSurfaceXWayland{s: surface},
		X:           -1,
		Y:           -1,
	}
	view.Destroy = surface.OnDestroy(func(s wlr.XWaylandSurface) {
		server.onDestroyView(&view)
	})
	view.Map = surface.OnMap(func(s wlr.XWaylandSurface) {
		server.onMapView(&view)
	})
	view.RequestMove = surface.OnRequestMove(func(s wlr.XWaylandSurface) {
		server.startMove(&view)
	})
	view.RequestResize = surface.OnRequestResize(func(s wlr.XWaylandSurface, edges wlr.Edges) {
		server.startBorderResize(&view, edges)
	})

	server.addView(&view)
}

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	switch surface.Role() {
	case wlr.XDGSurfaceRoleTopLevel:
		server.addXDGTopLevel(surface)
	case wlr.XDGSurfaceRolePopup:
		server.addXDGPopup(surface)
	case wlr.XDGSurfaceRoleNone:
		// TODO
	}
}

func (server *Server) addXDGPopup(surface wlr.XDGSurface) {
	p := Popup{
		Surface: surface,
	}
	p.Destroy = surface.OnDestroy(func(s wlr.XDGSurface) {
		server.onDestroyPopup(&p)
	})

	server.popups = append(server.popups, &p)
}

func (server *Server) isPopupSurface(surface wlr.Surface) (ok bool) {
	for _, p := range server.popups {
		p.Surface.ForEachSurface(func(s wlr.Surface, sx, sy int) {
			if s == surface {
				ok = true
			}
		})
	}
	return ok
}

func (server *Server) onDestroyPopup(p *Popup) {
	i := slices.Index(server.popups, p)
	server.popups = slices.Delete(server.popups, i, i+1)

	p.Release()
}

func (server *Server) addXDGTopLevel(surface wlr.XDGSurface) {
	view := View{
		ViewSurface: &viewSurfaceXDG{s: surface},
		X:           -1,
		Y:           -1,
	}
	view.Destroy = surface.OnDestroy(func(s wlr.XDGSurface) {
		server.onDestroyView(&view)
	})
	view.Map = surface.OnMap(func(s wlr.XDGSurface) {
		server.onMapView(&view)
	})
	view.RequestMove = surface.TopLevel().OnRequestMove(func(client wlr.SeatClient, serial uint32) {
		server.startMove(&view)
	})
	view.RequestResize = surface.TopLevel().OnRequestResize(func(client wlr.SeatClient, serial uint32, edges wlr.Edges) {
		server.startBorderResize(&view, edges)
	})

	surface.TopLevelSetTiled(wlr.EdgeLeft | wlr.EdgeRight | wlr.EdgeTop | wlr.EdgeBottom)

	server.addView(&view)
}

func (server *Server) onDestroyView(view *View) {
	view.Release()

	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)

	// TODO: Figure out why this causes a wlroots assertion failure.
	//if len(server.views) != 0 {
	//	n := server.views[len(server.views)-1]
	//	server.focusView(n, n.Surface())
	//}
}

func (server *Server) onMapView(view *View) {
	pid := view.PID()

	nv, ok := server.newViews[pid]
	if ok {
		delete(server.newViews, pid)

		server.resizeViewTo(nil, view, *nv.To)

		if nv.OnStarted != nil {
			nv.OnStarted(view)
		}

		return
	}

	out := server.outputAt(server.cursor.X(), server.cursor.Y())
	if out == nil {
		if len(server.outputs) == 0 {
			return
		}
		out = server.outputs[0]
	}

	if (view.X == -1) || (view.Y == -1) {
		server.centerViewOnOutput(out, view)
		return
	}

	server.moveViewTo(out, view, view.X, view.Y)
}

func (server *Server) addView(view *View) {
	server.views = append(server.views, view)

	nv, ok := server.newViews[view.PID()]
	if ok {
		view.Resize(nv.To.Dx(), nv.To.Dy())
	}
}

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	layout := server.outputLayout.Get(out.Output)
	current := view.Surface().Current()
	ow, oh := out.Output.EffectiveResolution()

	server.moveViewTo(
		out,
		view,
		layout.X()+(ow/2-current.Width()/2),
		layout.Y()+(oh/2-current.Height()/2),
	)
}

func (server *Server) moveViewTo(out *Output, view *View, x, y int) {
	view.X = x
	view.Y = y

	if out == nil {
		out = server.outputAt(float64(x), float64(y))
		if out == nil {
			return
		}
	}
	view.Surface().SendEnter(out.Output)
}

func (server *Server) resizeViewTo(out *Output, view *View, r image.Rectangle) {
	if out == nil {
		out = server.outputAt(float64(r.Min.X), float64(r.Min.Y))
	}

	vb := server.viewBounds(out, view)
	sb := server.surfaceBounds(out, view.Surface(), view.X, view.Y)
	off := sb.Min.Sub(vb.Min)
	r = r.Add(off)

	view.X = r.Min.X
	view.Y = r.Min.Y
	view.Resize(r.Dx(), r.Dy())

	view.Surface().SendEnter(out.Output)
}

func (server *Server) focusView(view *View, s wlr.Surface) {
	if !s.Valid() {
		s = view.Surface()
	}

	prev := server.seat.KeyboardState().FocusedSurface()
	if prev == s {
		return
	}
	pv := server.viewForSurface(prev)
	if pv != nil {
		pv.Activate(false)
	}

	k := server.seat.GetKeyboard()
	server.seat.KeyboardNotifyEnter(s, k.Keycodes(), k.Modifiers())

	view.Activate(true)
	server.bringViewToFront(view)
}

func (server *Server) viewForSurface(s wlr.Surface) *View {
	for _, view := range server.views {
		if view.HasSurface(s) {
			return view
		}
	}
	for _, view := range server.hidden {
		if view.HasSurface(s) {
			return view
		}
	}

	return nil
}

func (server *Server) bringViewToFront(view *View) {
	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)
	server.views = append(server.views, view)
}

func (server *Server) hideView(view *View) {
	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)

	server.hidden = append(server.hidden, view)
	server.mainMenu.Add(server, view.Title())
}

func (server *Server) unhideView(view *View) {
	i := slices.Index(server.hidden, view)
	server.hidden = slices.Delete(server.hidden, i, i+1)
	server.mainMenu.Remove(5 + i)

	server.views = append(server.views, view)
	server.focusView(view, view.Surface())
}

func (server *Server) closeView(view *View) {
	view.Close()
}
