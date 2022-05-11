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
	X, Y          int
	XDGSurface    wlr.XDGSurface
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

func (server *Server) viewBounds(out *Output, view *View) image.Rectangle {
	var r image.Rectangle
	view.XDGSurface.ForEachSurface(func(s wlr.Surface, sx, sy int) {
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

		surface, sx, sy, ok := view.XDGSurface.SurfaceAt(x-float64(view.X), y-float64(view.Y))
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

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	if surface.Role() != wlr.XDGSurfaceRoleTopLevel {
		server.addXDGPopup(surface)
		return
	}

	server.addXDGTopLevel(surface)
}

func (server *Server) addXDGPopup(surface wlr.XDGSurface) {
	// TODO
}

func (server *Server) addXDGTopLevel(surface wlr.XDGSurface) {
	view := View{
		X:          -1,
		Y:          -1,
		XDGSurface: surface,
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
	//	server.focusView(n, n.XDGSurface.Surface())
	//}
}

func (server *Server) onMapView(view *View) {
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

	client := view.XDGSurface.Resource().GetClient()
	pid, _, _ := client.GetCredentials()

	nv, ok := server.newViews[pid]
	if ok {
		delete(server.newViews, pid)

		view.X = nv.To.Min.X
		view.Y = nv.To.Min.Y
		view.XDGSurface.TopLevelSetSize(
			uint32(nv.To.Dx()),
			uint32(nv.To.Dy()),
		)
		nv.OnStarted(view)
	}
}

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	layout := server.outputLayout.Get(out.Output)
	current := view.XDGSurface.Surface().Current()
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
	view.XDGSurface.Surface().SendEnter(out.Output)
}

func (server *Server) resizeViewTo(out *Output, view *View, r image.Rectangle) {
	vb := server.viewBounds(out, view)
	sb := server.surfaceBounds(out, view.XDGSurface.Surface(), view.X, view.Y)
	off := sb.Min.Sub(vb.Min)
	r = r.Add(off)

	view.X = r.Min.X
	view.Y = r.Min.Y
	view.XDGSurface.TopLevelSetSize(uint32(r.Dx()), uint32(r.Dy()))

	if out == nil {
		out = server.outputAt(float64(view.X), float64(view.Y))
	}
	view.XDGSurface.Surface().SendEnter(out.Output)
}

func (server *Server) focusView(view *View, s wlr.Surface) {
	if !s.Valid() {
		s = view.XDGSurface.Surface()
	}

	prev := server.seat.KeyboardState().FocusedSurface()
	if prev == s {
		return
	}
	if prev.Valid() && (prev.Type() == wlr.SurfaceTypeXDG) {
		xdg := wlr.XDGSurfaceFromWLRSurface(prev)
		xdg.TopLevelSetActivated(false)
	}

	k := server.seat.GetKeyboard()
	server.seat.KeyboardNotifyEnter(s, k.Keycodes(), k.Modifiers())

	view.XDGSurface.TopLevelSetActivated(true)
	server.bringViewToFront(view)
}

func (server *Server) bringViewToFront(view *View) {
	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)
	server.views = append(server.views, view)
}

func (server *Server) closeView(view *View) {
	view.XDGSurface.SendClose()
}
