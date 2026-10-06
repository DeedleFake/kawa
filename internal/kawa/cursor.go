package kawa

import (
	"deedles.dev/wlr"
)

// setCursor and onRequestCursor are the only code that changes the
// cursor image.

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

const interactCursor = "hand"

// hoverState is what updateHoverCursor remembers between motions. It
// starts over each time an interaction ends.
type hoverState struct {
	inView    bool
	prevEdges wlr.Edges
}

// updateHoverCursor picks the image for what's under the pointer while
// no interaction is active.
func (server *Server) updateHoverCursor(h hit) {
	hs := &server.hover
	if h.edges != hs.prevEdges {
		cursor := interactCursor
		if !server.isViewTiled(h.view) {
			cursor = edgeCursors[h.edges]
			hs.prevEdges = h.edges
		}
		server.setCursor(cursor)
	}
	if (h.view == nil) && hs.inView {
		server.setCursor("left_ptr")
	}
	hs.inView = h.view != nil
}

func (server *Server) setCursor(name string) {
	if name == "" {
		return
	}

	xcursor := server.cursorMgr.GetXCursor(name, 1)
	if server.xwayland.Valid() && xcursor.Valid() {
		server.xwayland.SetCursor(xcursor.Image(0))
	}
	server.cursor.SetXCursor(server.cursorMgr, name)
}

// onRequestCursor shows a client's own image, but only while no
// interaction is active and only for the client under the pointer.
func (server *Server) onRequestCursor(client wlr.SeatClient, surface wlr.Surface, serial uint32, hotspotX, hotspotY int32) {
	if server.interaction != nil {
		return
	}

	if server.seat.PointerState().FocusedClient() == client {
		server.cursor.SetSurface(surface, hotspotX, hotspotY)
	}
}
