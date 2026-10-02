/* activation-client: a tiny xdg-activation-v1 test client for kawa.
 * It maps a 320x200 window, titled NAME, in a solid color picked from NAME,
 * and prints what it does and keyboard enter/leave on stdout.
 *   activation-client NAME launch CMD...    on each click, get a token (click serial + own surface) and start CMD with it
 *   activation-client NAME start            activate with $XDG_ACTIVATION_TOKEN before mapping
 *   activation-client NAME start-after      activate with $XDG_ACTIVATION_TOKEN after mapping
 *   activation-client NAME self SECS        after SECS, get a token with no serial or surface and activate self
 *   activation-client NAME self-stale SECS  after SECS, get a token from the last click's serial and own surface, activate self
 *   activation-client NAME fifo PATH        activate with each token written to the FIFO at PATH, like a single-instance app
 * ACTIVATION_CLIENT_MINIMIZE=1 asks to be minimized right after mapping.
 * launch.sh generates the protocol code with wayland-scanner and builds this. */
#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <poll.h>
#include <signal.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <time.h>
#include <unistd.h>
#include <wayland-client.h>
#include "xdg-shell-client.h"
#include "xdg-activation-v1-client.h"

static struct wl_compositor *compositor;
static struct wl_shm *shm;
static struct wl_seat *seat;
static struct xdg_wm_base *wm_base;
static struct xdg_activation_v1 *activation;
static struct wl_surface *surface;
static struct xdg_toplevel *toplevel;
static const char *name, *mode;
static char **cmd;
static uint32_t last_serial;
static int configured, running = 1;
static double deadline;

static double now(void) {
	struct timespec ts;
	clock_gettime(CLOCK_MONOTONIC, &ts);
	return ts.tv_sec + ts.tv_nsec / 1e9;
}

static void draw(int w, int h, uint32_t color) {
	int stride = w * 4, size = stride * h;
	int fd = memfd_create("activation-client", 0);
	ftruncate(fd, size);
	uint32_t *px = mmap(NULL, size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
	for (int i = 0; i < w * h; i++) px[i] = color;
	munmap(px, size);
	struct wl_shm_pool *pool = wl_shm_create_pool(shm, fd, size);
	struct wl_buffer *buf = wl_shm_pool_create_buffer(pool, 0, w, h, stride, WL_SHM_FORMAT_XRGB8888);
	wl_shm_pool_destroy(pool);
	close(fd);
	wl_surface_attach(surface, buf, 0, 0);
	wl_surface_damage_buffer(surface, 0, 0, w, h);
	wl_surface_commit(surface);
}

static void activate(const char *token, const char *why) {
	printf("%s: activate %s (%s)\n", name, token, why);
	fflush(stdout);
	xdg_activation_v1_activate(activation, token, surface);
}

static void token_done(void *data, struct xdg_activation_token_v1 *t, const char *token) {
	printf("%s: token done %s\n", name, token);
	fflush(stdout);
	if (strcmp(mode, "launch") == 0) {
		if (fork() == 0) {
			setenv("XDG_ACTIVATION_TOKEN", token, 1);
			setenv("DESKTOP_STARTUP_ID", token, 1);
			execvp(cmd[0], cmd);
			_exit(127);
		}
	} else {
		activate(token, mode);
	}
	xdg_activation_token_v1_destroy(t);
}
static const struct xdg_activation_token_v1_listener token_listener = { .done = token_done };

static void request_token(int with_serial) {
	struct xdg_activation_token_v1 *t = xdg_activation_v1_get_activation_token(activation);
	xdg_activation_token_v1_add_listener(t, &token_listener, NULL);
	if (with_serial) {
		xdg_activation_token_v1_set_serial(t, last_serial, seat);
		xdg_activation_token_v1_set_surface(t, surface);
	}
	xdg_activation_token_v1_commit(t);
}

static void p_enter(void *d, struct wl_pointer *p, uint32_t s, struct wl_surface *sf, wl_fixed_t x, wl_fixed_t y) {}
static void p_leave(void *d, struct wl_pointer *p, uint32_t s, struct wl_surface *sf) {}
static void p_motion(void *d, struct wl_pointer *p, uint32_t t, wl_fixed_t x, wl_fixed_t y) {}
static void p_button(void *d, struct wl_pointer *p, uint32_t serial, uint32_t t, uint32_t button, uint32_t state) {
	if (state != WL_POINTER_BUTTON_STATE_PRESSED) return;
	last_serial = serial;
	if (strcmp(mode, "launch") == 0) request_token(1);
}
static void p_axis(void *d, struct wl_pointer *p, uint32_t t, uint32_t a, wl_fixed_t v) {}
static const struct wl_pointer_listener pointer_listener = { p_enter, p_leave, p_motion, p_button, p_axis };

static void k_keymap(void *d, struct wl_keyboard *k, uint32_t f, int32_t fd, uint32_t sz) { close(fd); }
static void k_enter(void *d, struct wl_keyboard *k, uint32_t s, struct wl_surface *sf, struct wl_array *keys) {
	printf("%s: keyboard enter\n", name);
	fflush(stdout);
}
static void k_leave(void *d, struct wl_keyboard *k, uint32_t s, struct wl_surface *sf) {
	printf("%s: keyboard leave\n", name);
	fflush(stdout);
}
static void k_key(void *d, struct wl_keyboard *k, uint32_t s, uint32_t t, uint32_t key, uint32_t st) {}
static void k_mods(void *d, struct wl_keyboard *k, uint32_t s, uint32_t a, uint32_t b, uint32_t c, uint32_t g) {}
static void k_repeat(void *d, struct wl_keyboard *k, int32_t r, int32_t dl) {}
static const struct wl_keyboard_listener keyboard_listener = { k_keymap, k_enter, k_leave, k_key, k_mods, k_repeat };

static void seat_caps(void *d, struct wl_seat *s, uint32_t caps) {
	static int done;
	if (done) return;
	done = 1;
	if (caps & WL_SEAT_CAPABILITY_POINTER) wl_pointer_add_listener(wl_seat_get_pointer(s), &pointer_listener, NULL);
	if (caps & WL_SEAT_CAPABILITY_KEYBOARD) wl_keyboard_add_listener(wl_seat_get_keyboard(s), &keyboard_listener, NULL);
}
static void seat_name(void *d, struct wl_seat *s, const char *n) {}
static const struct wl_seat_listener seat_listener = { seat_caps, seat_name };

static void wm_ping(void *d, struct xdg_wm_base *b, uint32_t serial) { xdg_wm_base_pong(b, serial); }
static const struct xdg_wm_base_listener wm_listener = { wm_ping };

static void xs_configure(void *d, struct xdg_surface *xs, uint32_t serial) {
	xdg_surface_ack_configure(xs, serial);
	if (!configured) {
		configured = 1;
		uint32_t h = 0;
		for (const char *c = name; *c; c++) h = h * 31 + (unsigned char)*c;
		draw(320, 200, 0xFF000000 | (h & 0x7F7F7F) | 0x404040);
		if (strcmp(mode, "start-after") == 0 && getenv("XDG_ACTIVATION_TOKEN"))
			activate(getenv("XDG_ACTIVATION_TOKEN"), "after map");
		if (getenv("ACTIVATION_CLIENT_MINIMIZE")) {
			printf("%s: set_minimized\n", name);
			fflush(stdout);
			xdg_toplevel_set_minimized(toplevel);
		}
	} else {
		wl_surface_commit(surface);
	}
}
static const struct xdg_surface_listener xs_listener = { xs_configure };

static void tl_configure(void *d, struct xdg_toplevel *t, int32_t w, int32_t h, struct wl_array *s) {}
static void tl_close(void *d, struct xdg_toplevel *t) { running = 0; }
static const struct xdg_toplevel_listener tl_listener = { tl_configure, tl_close };

static void reg_global(void *d, struct wl_registry *r, uint32_t id, const char *iface, uint32_t v) {
	if (!strcmp(iface, wl_compositor_interface.name)) compositor = wl_registry_bind(r, id, &wl_compositor_interface, 4);
	else if (!strcmp(iface, wl_shm_interface.name)) shm = wl_registry_bind(r, id, &wl_shm_interface, 1);
	else if (!strcmp(iface, wl_seat_interface.name)) { seat = wl_registry_bind(r, id, &wl_seat_interface, 1); wl_seat_add_listener(seat, &seat_listener, NULL); }
	else if (!strcmp(iface, xdg_wm_base_interface.name)) { wm_base = wl_registry_bind(r, id, &xdg_wm_base_interface, 1); xdg_wm_base_add_listener(wm_base, &wm_listener, NULL); }
	else if (!strcmp(iface, xdg_activation_v1_interface.name)) activation = wl_registry_bind(r, id, &xdg_activation_v1_interface, 1);
}
static void reg_remove(void *d, struct wl_registry *r, uint32_t id) {}
static const struct wl_registry_listener reg_listener = { reg_global, reg_remove };

int main(int argc, char **argv) {
	if (argc < 3) { fprintf(stderr, "usage: activation-client NAME MODE [ARGS]\n"); return 2; }
	name = argv[1];
	signal(SIGCHLD, SIG_IGN); /* reap what launch starts */
	mode = argv[2];
	cmd = argv + 3;
	struct wl_display *dpy = wl_display_connect(NULL);
	if (!dpy) { perror("connect"); return 1; }
	wl_registry_add_listener(wl_display_get_registry(dpy), &reg_listener, NULL);
	wl_display_roundtrip(dpy);
	wl_display_roundtrip(dpy);
	if (!activation) { fprintf(stderr, "%s: no xdg_activation_v1\n", name); }

	surface = wl_compositor_create_surface(compositor);
	struct xdg_surface *xs = xdg_wm_base_get_xdg_surface(wm_base, surface);
	xdg_surface_add_listener(xs, &xs_listener, NULL);
	toplevel = xdg_surface_get_toplevel(xs);
	xdg_toplevel_add_listener(toplevel, &tl_listener, NULL);
	xdg_toplevel_set_title(toplevel, name);
	xdg_toplevel_set_app_id(toplevel, name);
	if (strcmp(mode, "start") == 0 && activation && getenv("XDG_ACTIVATION_TOKEN"))
		activate(getenv("XDG_ACTIVATION_TOKEN"), "before map");
	wl_surface_commit(surface);

	if (!strcmp(mode, "self") || !strcmp(mode, "self-stale"))
		deadline = now() + (argc > 3 ? atof(argv[3]) : 5);

	struct pollfd pfd[2] = { { .fd = wl_display_get_fd(dpy), .events = POLLIN }, { .fd = -1, .events = POLLIN } };
	if (!strcmp(mode, "fifo")) pfd[1].fd = open(argv[3], O_RDWR | O_NONBLOCK);
	while (running) {
		wl_display_dispatch_pending(dpy);
		wl_display_flush(dpy);
		int timeout = -1;
		if (deadline > 0) {
			double left = deadline - now();
			if (left <= 0) {
				deadline = 0;
				request_token(!strcmp(mode, "self-stale"));
				continue;
			}
			timeout = (int)(left * 1000) + 1;
		}
		if (poll(pfd, 2, timeout) <= 0) continue;
		if (pfd[1].revents & POLLIN) {
			char buf[256];
			ssize_t n = read(pfd[1].fd, buf, sizeof(buf) - 1);
			if (n > 0) {
				buf[n] = 0;
				buf[strcspn(buf, "\n")] = 0;
				activate(buf, "fifo");
				wl_display_flush(dpy);
			}
		}
		if ((pfd[0].revents & POLLIN) && wl_display_dispatch(dpy) < 0) break;
	}
	return 0;
}
