package main

import (
	"deedles.dev/wlr"
)

// activationToken is what kawa knows about an xdg-activation token
// beyond what wlroots keeps.
type activationToken struct {
	// trusted is true for tokens that kawa made itself for a program
	// that the user started from it.
	trusted bool
	// hadFocus is true if the surface that the client said the token
	// came from belonged to whatever had the keyboard when the token
	// was made.
	hadFocus bool

	onDestroyListener wlr.Listener
}

func (server *Server) initActivation() {
	server.activation = wlr.CreateXDGActivationV1(server.display)
	server.activationTokens = make(map[wlr.XDGActivationTokenV1]*activationToken)
	server.onNewActivationTokenListener = server.activation.OnNewToken(func(t wlr.XDGActivationTokenV1) {
		server.trackActivationToken(t, &activationToken{
			hadFocus: server.surfaceHasFocus(t.Surface()),
		})
	})
	server.onRequestActivateListener = server.activation.OnRequestActivate(server.onRequestActivate)
}

func (server *Server) trackActivationToken(t wlr.XDGActivationTokenV1, info *activationToken) {
	info.onDestroyListener = t.OnDestroy(func(t wlr.XDGActivationTokenV1) {
		info.onDestroyListener.Destroy()
		delete(server.activationTokens, t)
	})
	server.activationTokens[t] = info
}

// newActivationToken makes a token for a program that kawa is about to
// start. It returns an empty string if wlroots couldn't make one.
func (server *Server) newActivationToken() string {
	t := server.activation.CreateToken()
	if !t.Valid() {
		return ""
	}
	server.trackActivationToken(t, &activationToken{trusted: true})
	return t.Name()
}

// surfaceHasFocus returns true if s is, or belongs to the same window
// or layer surface as, the surface that has the keyboard.
func (server *Server) surfaceHasFocus(s wlr.Surface) bool {
	focused := server.seat.KeyboardState().FocusedSurface()
	if !s.Valid() || !focused.Valid() {
		return false
	}
	if s == focused {
		return true
	}
	if view := server.viewForSurface(focused); view != nil {
		return view.HasSurface(s) || view.isPopupSurface(s)
	}
	if ls := server.layerForSurface(focused); ls != nil {
		return server.layerForSurface(s) == ls
	}
	return false
}

// activationAllowed decides whether a client may take the keyboard with
// t. Only a token that comes from user input on the surface that the
// user was using counts: wlroots already refuses tokens whose serial
// was never sent to the client, so a seat means a real input event, and
// the token's surface must have had the keyboard when the token was
// made or have it now. A token without a surface can't be tied to what
// the user was doing, so it doesn't count.
func (server *Server) activationAllowed(t wlr.XDGActivationTokenV1) bool {
	info := server.activationTokens[t]
	if (info != nil) && info.trusted {
		return true
	}
	if !t.Seat().Valid() {
		return false
	}
	return ((info != nil) && info.hadFocus) || server.surfaceHasFocus(t.Surface())
}

func (server *Server) onRequestActivate(event wlr.XDGActivationV1RequestActivateEvent) {
	view := server.viewForMainSurface(event.Surface())
	if view == nil {
		return
	}
	if (view == server.focusedView()) && !server.isViewHidden(view) {
		return
	}

	allowed := server.activationAllowed(event.Token())
	wlr.Log(wlr.Debug, "activation request for %q: allowed=%v mapped=%v", view.Title(), allowed, view.Mapped())

	// Clients often ask before their window is mapped, such as one that
	// was started with a token in its environment.
	if !view.Mapped() {
		view.activateOnMap = view.activateOnMap || allowed
		view.attention = !view.activateOnMap
		return
	}

	if !allowed {
		server.setViewAttention(view, true)
		return
	}
	server.activateView(view)
}

// activateView shows view if it's hidden, and focuses and raises it.
func (server *Server) activateView(view *View) {
	if server.isViewHidden(view) {
		server.unhideView(view)
		return
	}
	server.focusView(view, view.Surface())
}

func (server *Server) setViewAttention(view *View, attention bool) {
	if view.attention == attention {
		return
	}
	view.attention = attention
	server.updateTitles()
}

// viewForMainSurface finds the view whose main surface is s, mapped or
// not.
func (server *Server) viewForMainSurface(s wlr.Surface) *View {
	if !s.Valid() {
		return nil
	}
	for _, list := range [][]*View{server.views, server.tiled, server.hidden} {
		for _, view := range list {
			if view.Surface() == s {
				return view
			}
		}
	}
	return nil
}
