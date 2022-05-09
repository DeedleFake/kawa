package main

import (
	"os"
	"time"

	"deedles.dev/wlr"
	"deedles.dev/wlr/xkb"
)

func (server *Server) onCursorMotion(dev wlr.InputDevice, t time.Time, dx, dy float64) {
	server.cursor.Move(dev, dx, dy)
	server.processCursorMotion(t)
}

func (server *Server) onCursorMotionAbsolute(dev wlr.InputDevice, t time.Time, x, y float64) {
	server.cursor.WarpAbsolute(dev, x, y)
	server.processCursorMotion(t)
}

func (server *Server) processCursorMotion(t time.Time) {
	// TODO
}

func (server *Server) onCursorButton(dev wlr.InputDevice, t time.Time, b uint32, state wlr.ButtonState) {
	// TODO
}

func (server *Server) onCursorAxis(dev wlr.InputDevice, t time.Time, source wlr.AxisSource, orient wlr.AxisOrientation, delta float64, deltaDiscrete int32) {
	server.seat.PointerNotifyAxis(t, orient, delta, deltaDiscrete, source)
}

func (server *Server) onCursorFrame() {
	server.seat.PointerNotifyFrame()
}

func (server *Server) onNewInput(device wlr.InputDevice) {
	switch device.Type() {
	case wlr.InputDeviceTypeKeyboard:
		server.newKeyboard(device)
	case wlr.InputDeviceTypePointer:
		server.newPointer(device)
	}

	caps := wlr.SeatCapabilityPointer
	if len(server.keyboards) != 0 {
		caps |= wlr.SeatCapabilityKeyboard
	}
	server.seat.SetCapabilities(caps)
}

func (server *Server) newKeyboard(device wlr.InputDevice) {
	kb := Keyboard{
		Server: server,
		Device: device,
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

	wkb := device.Keyboard()
	wkb.SetKeymap(keymap)
	wkb.SetRepeatInfo(25, 600)

	kb.Modifiers = wkb.OnModifiers(kb.onModifiers)
	kb.Key = wkb.OnKey(kb.onKey)

	server.seat.SetKeyboard(device)
	server.keyboards = append(server.keyboards, kb)
}

func (server *Server) newPointer(device wlr.InputDevice) {
	server.cursor.AttachInputDevice(device)
}

func (server *Server) onRequestCursor(client wlr.SeatClient, surface wlr.Surface, serial uint32, hotspotX, hotspotY int32) {
	panic("Not implemented.")
}

func (kb *Keyboard) onModifiers(keyboard wlr.Keyboard) {
	server := kb.Server

	server.seat.SetKeyboard(kb.Device)
	server.seat.KeyboardNotifyModifiers(keyboard.Modifiers())
}

func (kb *Keyboard) onKey(keyboard wlr.Keyboard, t time.Time, code uint32, update bool, state wlr.KeyState) {
	server := kb.Server

	if server.handleShortcut() {
		return
	}

	server.seat.SetKeyboard(kb.Device)
	server.seat.KeyboardNotifyKey(t, code, state)
}

func (server *Server) handleShortcut() bool {
	// TODO
	return false
}
