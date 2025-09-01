#include "output_c.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wayland-client.h>
#include <wayland-util.h>

struct ctx {
  struct wl_list outputs;
};

struct output_t {
  int id;

  struct output_props props;

  struct ctx *ctx;
  struct wl_output *output;

  struct wl_list link; // link to the next element in linked list (see wl_list)
};

static void output_handle_geometry(void *data, struct wl_output *wl_output,
                                   int32_t x, int32_t y, int32_t physical_width,
                                   int32_t physical_height, int32_t subpixel,
                                   const char *make, const char *model,
                                   int32_t output_transform) {
  struct output_t *out = (struct output_t *)data;
  out->props.physical_width = physical_width;
  out->props.physical_height = physical_height;
}

static void output_handle_mode(void *data, struct wl_output *wl_output,
                               uint32_t flags, int32_t width, int32_t height,
                               int32_t refresh) {}

static void output_handle_done(void *data, struct wl_output *wl_output) {}

static void output_handle_scale(void *data, struct wl_output *wl_output,
                                int32_t scale) {}

static void output_handle_name(void *data, struct wl_output *wl_output,
                               const char *name) {
  struct output_t *out = (struct output_t *)data;
  out->props.name = name;
}

void output_handle_description(void *data, struct wl_output *wl_output,
                               const char *description) {};

static const struct wl_output_listener output_listener = {
    output_handle_geometry, output_handle_mode, output_handle_done,
    output_handle_scale,    output_handle_name, output_handle_description,
};

static void global_registry_handler(void *data, struct wl_registry *registry,
                                    uint32_t id, const char *interface,
                                    uint32_t version) {
  if (!strcmp(interface, "wl_output")) {
    struct ctx *ctx = (struct ctx *)data;
    struct output_t *output = malloc(sizeof(struct output_t));
    output->ctx = ctx;
    output->id = id;
    output->output =
        wl_registry_bind(registry, id, &wl_output_interface, version);
    wl_list_insert(&ctx->outputs, &output->link);
    wl_output_add_listener(output->output, &output_listener, output);
  }
}

static void global_registry_remover(void *data, struct wl_registry *registry,
                                    uint32_t id) {}

static const struct wl_registry_listener registry_listener = {
    global_registry_handler, global_registry_remover};

enum error list_wl_outputs(struct output_props **result, size_t *count) {
  struct wl_display *display = wl_display_connect(NULL);
  if (display == NULL) {
    return ERROR_WL_DISPLAY_CONNECT_FAILED;
  }

  struct ctx ctx;
  wl_list_init(&ctx.outputs);

  struct wl_registry *registry = wl_display_get_registry(display);
  wl_registry_add_listener(registry, &registry_listener, &ctx);

  wl_display_dispatch(display);
  wl_display_roundtrip(display);

  *count = (size_t)wl_list_length(&ctx.outputs);
  *result = malloc(*count * sizeof(struct output_props));
  if (*result == NULL) {
    return ERROR_FAILED_ALLOCATING_RESULT_BUFFER;
  }

  struct output_t *out, *tmp;
  int i = 0;
  wl_list_for_each_safe(out, tmp, &ctx.outputs, link) {
    wl_output_destroy(out->output);
    wl_list_remove(&out->link);

    (*result)[i] = out->props;

    free(out);
    i += 1;
  }
  wl_registry_destroy(registry);
  wl_display_disconnect(display);

  return ERROR_NONE;
}
