package main

import (
	"flag"
	"strings"

	"deedles.dev/wlr"
)

func main() {
	cage := flag.String("cage", "cage -d", "wrapper to use for caging windows")
	term := flag.String("term", "alacritty", "terminal to use when creating a new window")
	flag.Parse()

	server := Server{
		Cage: strings.Fields(*cage),
		Term: strings.Fields(*term),
	}

	wlr.InitLog(wlr.Debug, nil)

	server.display = wlr.CreateDisplay()
	server.backend = wlr.AutocreateBackend(server.display)
	server.renderer = wlr.AutocreateRenderer(server.backend)
	server.allocator = wlr.AutocreateAllocator(server.backend, server.renderer)
	server.renderer.InitWLDisplay(server.display)

	wlr.CreateCompositor(server.display, server.renderer)
	wlr.CreateDataDeviceManager(server.display)

	wlr.CreateExportDMABufV1(server.display)
	wlr.CreateScreencopyManagerV1(server.display)
	wlr.CreateDataControlManagerV1(server.display)
	wlr.CreatePrimarySelectionV1DeviceManager(server.display)

	wlr.CreateGammaControlManagerV1(server.display)

	server.newOutput = server.backend.OnNewOutput(server.NewOutput)

	server.outputLayout = wlr.CreateOutputLayout()
	wlr.CreateXDGOutputManagerV1(server.display, server.outputLayout)

	server.cursor = wlr.CreateCursor()
	server.cursor.AttachOutputLayout(server.outputLayout)
	server.cursorMgr = wlr.CreateXCursorManager("", 24)
	server.cursorMgr.Load(1)

	for _, c := range server.outputConfigs {
		server.cursorMgr.Load(float64(c.Scale))
	}

	server.cursorMotion = server.cursor.OnMotion(server.CursorMotion)
	server.cursorMotionAbsolute = server.cursor.OnMotionAbsolute(server.CursorMotionAbsolute)
	server.cursorButton = server.cursor.OnButton(server.CursorButton)
	server.cursorAxis = server.cursor.OnAxis(server.CursorAxis)
	server.cursorFrame = server.cursor.OnFrame(server.CursorFrame)

	//wl_list_init(&server.inputs);
	//server.new_input.notify = server_new_input;
	//wl_signal_add(&server.backend->events.new_input, &server.new_input);

	//server.seat = wlr_seat_create(server.wl_display, "seat0");
	//server.request_cursor.notify = seat_request_cursor;
	//wl_signal_add(&server.seat->events.request_set_cursor, &server.request_cursor);
	//wl_list_init(&server.keyboards);
	//wl_list_init(&server.pointers);

	//wl_list_init(&server.views);
	//server.xdg_shell = wlr_xdg_shell_create(server.wl_display);
	//server.new_xdg_surface.notify = server_new_xdg_surface;
	//wl_signal_add(&server.xdg_shell->events.new_surface, &server.new_xdg_surface);

	//wl_list_init(&server.new_views);

	//server.layer_shell = wlr_layer_shell_v1_create(server.wl_display);
	//server.new_layer_surface.notify = server_new_layer_surface;
	//wl_signal_add(&server.layer_shell->events.new_surface, &server.new_layer_surface);

	//server.menu.x = server.menu.y = -1;
	//gen_menu_textures(&server);
	//server.input_state = INPUT_STATE_NONE;

	//const char *socket = wl_display_add_socket_auto(server.wl_display);
	//if (!socket) {
	//	wlr_backend_destroy(server.backend);
	//	return 1;
	//}

	//if (!wlr_backend_start(server.backend)) {
	//	wlr_backend_destroy(server.backend);
	//	wl_display_destroy(server.wl_display);
	//	return 1;
	//}

	//setenv("WAYLAND_DISPLAY", socket, true);
	//wlr_log(WLR_INFO, "Running Wayland compositor on WAYLAND_DISPLAY=%s", socket);
	//wl_display_run(server.wl_display);

	//wl_display_destroy_clients(server.wl_display);
	//wl_display_destroy(server.wl_display);
	//return 0;
}
