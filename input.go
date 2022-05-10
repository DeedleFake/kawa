package main

import (
	"os"
	"time"

	"deedles.dev/wlr"
	"deedles.dev/wlr/xkb"
)

func (server *Server) onNewInput(device wlr.InputDevice) {
	switch device.Type() {
	case wlr.InputDeviceTypeKeyboard:
		server.addKeyboard(device)
	case wlr.InputDeviceTypePointer:
		server.addPointer(device)
	}
}

func (server *Server) addKeyboard(dev wlr.InputDevice) {
	kb := Keyboard{
		Device: dev,
	}

	rules := xkb.RuleNames{
		Rules:   os.Getenv("XKB_DEFAULT_RULES"),
		Model:   os.Getenv("XKB_DEFAULT_MODEL"),
		Layout:  os.Getenv("XKB_DEFAULT_LAYOUT"),
		Variant: os.Getenv("XKB_DEFAULT_VARIANT"),
		Options: os.Getenv("XKB_DEFAULT_OPTIONS"),
	}

	ctx := xkb.NewContext(xkb.ContextNoFlags)
	defer ctx.Unref()

	keymap := xkb.NewKeymapFromNames(ctx, &rules, xkb.KeymapCompileNoFlags)
	defer keymap.Unref()

	wkb := dev.Keyboard()
	wkb.SetKeymap(keymap)
	wkb.SetRepeatInfo(25, 600)

	kb.Modifiers = wkb.OnModifiers(func(k wlr.Keyboard) {
		server.onKeyboardModifiers(&kb)
	})
	kb.Key = wkb.OnKey(func(k wlr.Keyboard, t time.Time, code uint32, update bool, state wlr.KeyState) {
		server.onKeyboardKey(&kb, code, update, state, t)
	})

	server.seat.SetKeyboard(dev)
	server.keyboards = append(server.keyboards, &kb)

	server.seat.SetCapabilities(server.seat.Capabilities() | wlr.SeatCapabilityKeyboard)
}

func (server *Server) addPointer(dev wlr.InputDevice) {
	server.cursor.AttachInputDevice(dev)
	server.seat.SetCapabilities(server.seat.Capabilities() | wlr.SeatCapabilityPointer)

	server.pointers = append(server.pointers, dev)
}

func (server *Server) onKeyboardModifiers(kb *Keyboard) {
	// TODO
}

func (server *Server) onKeyboardKey(kb *Keyboard, code uint32, update bool, state wlr.KeyState, t time.Time) {
	switch state {
	case wlr.KeyStatePressed:
		server.onKeyboardKeyPressed(kb, code, update, t)
	case wlr.KeyStateReleased:
		server.onKeyboardKeyReleased(kb, code, update, t)
	}
}

func (server *Server) onKeyboardKeyPressed(kb *Keyboard, code uint32, update bool, t time.Time) {
	// TODO
}

func (server *Server) onKeyboardKeyReleased(kb *Keyboard, code uint32, update bool, t time.Time) {
	// TODO
}

func (server *Server) onCursorMotion(dev wlr.InputDevice, t time.Time, dx, dy float64) {
	server.cursor.Move(dev, dx, dy)
	server.inputMode.CursorMoved(server, t)
}

func (server *Server) onCursorMotionAbsolute(dev wlr.InputDevice, t time.Time, x, y float64) {
	server.cursor.WarpAbsolute(dev, x, y)
	server.inputMode.CursorMoved(server, t)
}

func (server *Server) onCursorButton(dev wlr.InputDevice, t time.Time, b wlr.CursorButton, state wlr.ButtonState) {
	switch state {
	case wlr.ButtonPressed:
		server.inputMode.CursorButtonPressed(server, dev, b, t)
	case wlr.ButtonReleased:
		server.inputMode.CursorButtonReleased(server, dev, b, t)
	}
}

func (server *Server) onCursorAxis(dev wlr.InputDevice, t time.Time, source wlr.AxisSource, orient wlr.AxisOrientation, delta float64, deltaDiscrete int32) {
	server.seat.PointerNotifyAxis(t, orient, delta, deltaDiscrete, source)
}

func (server *Server) onCursorFrame() {
	server.seat.PointerNotifyFrame()
}

func (server *Server) onRequestCursor(client wlr.SeatClient, surface wlr.Surface, serial uint32, hotspotX, hotspotY int32) {
	m, ok := server.inputMode.(interface {
		RequestCursor(*Server, wlr.Surface, int, int)
	})
	if !ok {
		return
	}

	focused := server.seat.PointerState().FocusedClient()
	if focused == client {
		m.RequestCursor(server, surface, int(hotspotX), int(hotspotY))
	}
}

func (server *Server) setCursor(name string) {
	if name == "" {
		return
	}

	server.cursorMgr.SetCursorImage(name, server.cursor)
}

//func (server *Server) processCursorMotion(t time.Time) {
//	var sx, sy float64
//	var surface wlr.Surface
//	var view *View
//
//	var ok bool
//	if server.inputState == InputStateNone {
//		view, surface, sx, sy, ok = server.viewAt(server.cursor.X(), server.cursor.Y())
//	}
//	if !ok {
//		switch server.inputState {
//		case InputStateMoveSelect, InputStateResizeSelect, InputStateDeleteSelect, InputStateHideSelect:
//			server.cursorMgr.SetCursorImage("hand1", server.cursor)
//
//		case InputStateMove, InputStateResizeEnd, InputStateNewEnd:
//			server.cursorMgr.SetCursorImage("grabbing", server.cursor)
//
//		case InputStateBorderDrag:
//			server.cursorMgr.SetCursorImage(server.corner, server.cursor)
//
//		case InputStateResizeStart, InputStateNewStart:
//			server.cursorMgr.SetCursorImage("top_left_corner", server.cursor)
//
//		default:
//			server.cursorMgr.SetCursorImage("left_ptr", server.cursor)
//		}
//	}
//
//	if surface.Valid() {
//		focusChanged := server.seat.PointerState().FocusedSurface() != surface
//		server.seat.PointerNotifyEnter(surface, sx, sy)
//		if !focusChanged {
//			server.seat.PointerNotifyMotion(t, sx, sy)
//		}
//		return
//	}
//
//	if view != nil {
//		server.cursorMgr.SetCursorImage(corners[view.Area], server.cursor)
//	}
//
//	server.seat.PointerClearFocus()
//}
//
//func (server *Server) cursorButtonInternal(dev wlr.InputDevice, t time.Time, b wlr.CursorButton, state wlr.ButtonState) {
//	menu := box(server.menu.X, server.menu.Y, server.menu.Width, server.menu.Height)
//
//	switch server.inputState {
//	case InputStateNone:
//		if state == wlr.ButtonPressed {
//			server.inputState = InputStateMenu
//			server.menu.X = int(server.cursor.X())
//			server.menu.Y = int(server.cursor.Y())
//		}
//
//	case InputStateMenu:
//		if image.Pt(int(server.cursor.X()), int(server.cursor.Y())).In(menu) {
//			server.menuSelect()
//			break
//		}
//		if state == wlr.ButtonPressed {
//			server.inputState = InputStateNone
//			server.menu.X = -1
//			server.menu.Y = -1
//		}
//
//	case InputStateNewStart:
//		if state != wlr.ButtonPressed {
//			break
//		}
//
//		server.interactive.SX = int(server.cursor.X())
//		server.interactive.SY = int(server.cursor.Y())
//		server.inputState = InputStateNewEnd
//
//	case InputStateNewEnd:
//		server.newView()
//		server.viewEndInteractive()
//
//	case InputStateResizeSelect:
//		if state != wlr.ButtonPressed {
//			break
//		}
//
//		view, surface, sx, sy, ok := server.viewAt(server.cursor.X(), server.cursor.Y())
//		if !ok {
//			server.viewEndInteractive()
//			break
//		}
//		view.beginInteractive(surface, sx, sy, "grabbing", InputStateResizeStart)
//
//	case InputStateResizeStart:
//		if state != wlr.ButtonPressed {
//			break
//		}
//
//		server.interactive.SX = int(server.cursor.X())
//		server.interactive.SY = int(server.cursor.Y())
//		server.interactive.View.Area = ViewAreaBorderBottomRight
//		server.inputState = InputStateResizeEnd
//
//	default:
//		panic("Not implemented.")
//	}
//}
//
//func (server *Server) menuSelect() {
//	server.menu.X = -1
//	server.menu.Y = -1
//	switch server.menu.Selected {
//	case 0:
//		server.inputState = InputStateNewStart
//		server.cursorMgr.SetCursorImage("top_left_corner", server.cursor)
//
//	case 1:
//		server.inputState = InputStateResizeSelect
//		server.cursorMgr.SetCursorImage("hand1", server.cursor)
//
//	case 2:
//		server.inputState = InputStateMoveSelect
//		server.cursorMgr.SetCursorImage("hand1", server.cursor)
//
//	case 3:
//		server.inputState = InputStateDeleteSelect
//		server.cursorMgr.SetCursorImage("hand1", server.cursor)
//
//	default:
//		server.inputState = InputStateNone
//	}
//}
//
//func (server *Server) newKeyboard(device wlr.InputDevice) {
//}
//
//func (kb *Keyboard) onModifiers(keyboard wlr.Keyboard) {
//	server := kb.Server
//
//	server.seat.SetKeyboard(kb.Device)
//	server.seat.KeyboardNotifyModifiers(keyboard.Modifiers())
//}
//
//func (kb *Keyboard) onKey(keyboard wlr.Keyboard, t time.Time, code uint32, update bool, state wlr.KeyState) {
//	server := kb.Server
//
//	if server.handleShortcut() {
//		return
//	}
//
//	server.seat.SetKeyboard(kb.Device)
//	server.seat.KeyboardNotifyKey(t, code, state)
//}
//
//func (server *Server) handleShortcut() bool {
//	// TODO
//	return false
//}
//
//func (view *View) beginInteractive(surface wlr.Surface, sx, sy float64, cname string, state InputState) {
//	server := view.Server
//
//	view.focus(surface)
//	server.interactive.View = view
//	server.interactive.SX = int(sx)
//	server.interactive.SY = int(sy)
//	server.inputState = state
//	server.cursorMgr.SetCursorImage(cname, server.cursor)
//}
//
//func (server *Server) viewEndInteractive() {
//	server.inputState = InputStateNone
//	server.interactive.View = nil
//	server.cursorMgr.SetCursorImage("left_ptr", server.cursor)
//}
//
//func (server *Server) newView() {
//	panic("Not implemented.")
//}
