/* Minimal Wayland client: connect to WAYLAND_DISPLAY and dump registry globals. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wayland-client.h>

struct state {
	int count;
};

static void registry_global(void *data, struct wl_registry *registry,
			    uint32_t name, const char *interface, uint32_t version)
{
	struct state *st = data;
	(void)registry;
	st->count++;
	printf("%u\t%s\tv%u\n", name, interface, version);
}

static void registry_global_remove(void *data, struct wl_registry *registry, uint32_t name)
{
	(void)data;
	(void)registry;
	(void)name;
}

static const struct wl_registry_listener registry_listener = {
	.global = registry_global,
	.global_remove = registry_global_remove,
};

int main(int argc, char **argv)
{
	(void)argc;
	(void)argv;
	struct wl_display *display = wl_display_connect(NULL);
	if (!display) {
		fprintf(stderr, "wayland-registry: wl_display_connect failed (WAYLAND_DISPLAY=%s)\n",
			getenv("WAYLAND_DISPLAY") ? getenv("WAYLAND_DISPLAY") : "(unset)");
		return 1;
	}

	struct state st = {0};
	struct wl_registry *registry = wl_display_get_registry(display);
	wl_registry_add_listener(registry, &registry_listener, &st);
	if (wl_display_roundtrip(display) < 0) {
		fprintf(stderr, "wayland-registry: roundtrip failed\n");
		wl_registry_destroy(registry);
		wl_display_disconnect(display);
		return 1;
	}

	printf("globals=%d\n", st.count);
	wl_registry_destroy(registry);
	wl_display_disconnect(display);
	return st.count > 0 ? 0 : 2;
}
