package main

import (
	"os"
	"time"

	"deedles.dev/wlr"
	"deedles.dev/wlr/xkb"
)

type CursorMover interface {
	CursorMoved(*Server, time.Time)
}

type CursorButtonPresser interface {
	CursorButtonPressed(*Server, wlr.InputDevice, wlr.CursorButton, time.Time)
}

type CursorButtonReleaser interface {
	CursorButtonReleased(*Server, wlr.InputDevice, wlr.CursorButton, time.Time)
}

type CursorRequester interface {
	RequestCursor(*Server, wlr.Surface, int, int)
}

type Keyboard struct {
	Device wlr.InputDevice

	onModifiersListener wlr.Listener
	onKeyListener       wlr.Listener
}

func (server *Server) onNewInput(device wlr.InputDevice) {
	switch device.Type() {
	case wlr.InputDeviceTypeKeyboard:
		server.addKeyboard(device)
	case wlr.InputDeviceTypePointer:
		server.addPointer(device)
	}
}

func (server *Server) onKeyboardModifiers(kb *Keyboard) {
	server.seat.SetKeyboard(kb.Device)
	server.seat.KeyboardNotifyModifiers(kb.Device.Keyboard().Modifiers())
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
	if server.handleKeyboardShortcut(kb, code, t) {
		return
	}

	server.seat.SetKeyboard(kb.Device)
	server.seat.KeyboardNotifyKey(t, code, wlr.KeyStatePressed)
}

func (server *Server) onKeyboardKeyReleased(kb *Keyboard, code uint32, update bool, t time.Time) {
	server.seat.SetKeyboard(kb.Device)
	server.seat.KeyboardNotifyKey(t, code, wlr.KeyStateReleased)
}

func (server *Server) onCursorMotion(dev wlr.InputDevice, t time.Time, dx, dy float64) {
	server.cursor.Move(dev, dx, dy)

	m, ok := server.inputMode.(CursorMover)
	if ok {
		m.CursorMoved(server, t)
	}
}

func (server *Server) onCursorMotionAbsolute(dev wlr.InputDevice, t time.Time, x, y float64) {
	server.cursor.WarpAbsolute(dev, x, y)

	m, ok := server.inputMode.(CursorMover)
	if ok {
		m.CursorMoved(server, t)
	}
}

func (server *Server) onCursorButton(dev wlr.InputDevice, t time.Time, b wlr.CursorButton, state wlr.ButtonState) {
	switch state {
	case wlr.ButtonPressed:
		m, ok := server.inputMode.(CursorButtonPresser)
		if ok {
			m.CursorButtonPressed(server, dev, b, t)
		}
	case wlr.ButtonReleased:
		m, ok := server.inputMode.(CursorButtonReleaser)
		if ok {
			m.CursorButtonReleased(server, dev, b, t)
		}
	}
}

func (server *Server) onCursorAxis(dev wlr.InputDevice, t time.Time, source wlr.AxisSource, orient wlr.AxisOrientation, delta float64, deltaDiscrete int32) {
	server.seat.PointerNotifyAxis(t, orient, delta, deltaDiscrete, source)
}

func (server *Server) onCursorFrame() {
	server.seat.PointerNotifyFrame()
}

func (server *Server) onRequestCursor(client wlr.SeatClient, surface wlr.Surface, serial uint32, hotspotX, hotspotY int32) {
	m, ok := server.inputMode.(CursorRequester)
	if !ok {
		return
	}

	focused := server.seat.PointerState().FocusedClient()
	if focused == client {
		m.RequestCursor(server, surface, int(hotspotX), int(hotspotY))
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

	kb.onModifiersListener = wkb.OnModifiers(func(k wlr.Keyboard) {
		server.onKeyboardModifiers(&kb)
	})
	kb.onKeyListener = wkb.OnKey(func(k wlr.Keyboard, t time.Time, code uint32, update bool, state wlr.KeyState) {
		server.onKeyboardKey(&kb, code, update, state, t)
	})

	server.seat.SetKeyboard(dev)
	server.keyboards = append(server.keyboards, &kb)

	server.seat.SetCapabilities(server.seat.Capabilities() | wlr.SeatCapabilityKeyboard)
}

func (server *Server) addPointer(dev wlr.InputDevice) {
	server.cursor.AttachInputDevice(dev)
	server.seat.SetCapabilities(server.seat.Capabilities() | wlr.SeatCapabilityPointer)
	server.setCursor("left_ptr")

	server.pointers = append(server.pointers, dev)
}

func (server *Server) setCursor(name string) {
	if name == "" {
		return
	}

	server.xwayland.SetCursor(server.cursorMgr.GetXCursor(name, 1).Image(0))
	server.cursorMgr.SetCursorImage(name, server.cursor)
}

func (server *Server) handleKeyboardShortcut(kb *Keyboard, code uint32, t time.Time) bool {
	// TODO
	return false
}
