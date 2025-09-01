#ifndef OUTPUT_C_H
#define OUTPUT_C_H

#include <stddef.h>

enum error {
  ERROR_NONE,
  ERROR_WL_DISPLAY_CONNECT_FAILED,
  ERROR_FAILED_ALLOCATING_RESULT_BUFFER
};

struct output_props {
  const char *name;
  int physical_width;
  int physical_height;
};

enum error list_wl_outputs(struct output_props **result, size_t *count);

#endif
