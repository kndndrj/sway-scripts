//go:build cgo

package core

// #cgo pkg-config: wayland-client
// #include <stdlib.h>
// #include "output_c.h"
//
// void free_props(struct output_props *arr) { free(arr); }
import "C"

import (
	"context"
	"errors"
	"unsafe"
)

func init() {
	// assigns based on build flags
	fetcher = fetcherC
}

// fetchOutputs returns an up to date info about outputs.
func fetcherC(ctx context.Context) ([]*physicalDimensions, error) {
	var cprops *C.struct_output_props
	var count C.size_t
	switch C.list_wl_outputs(&cprops, &count) {
	case C.ERROR_WL_DISPLAY_CONNECT_FAILED:
		return nil, errors.New("failed connecting to wayland display")
	case C.ERROR_FAILED_ALLOCATING_RESULT_BUFFER:
		return nil, errors.New("failed allocating result buffer")
	}
	if cprops == nil || count == 0 {
		return nil, errors.New("invalid returned values from C api")
	}
	defer C.free_props(cprops)

	length := int(count)
	slice := make([]*physicalDimensions, length)

	// Convert C array to Go slice
	// WARNING: need unsafe pointer arithmetic
	cSlice := (*[1 << 30]C.struct_output_props)(unsafe.Pointer(cprops))[:length:length]
	for i := range length {
		slice[i] = &physicalDimensions{
			Name:           C.GoString(cSlice[i].name),
			PhysicalWidth:  int(cSlice[i].physical_width),
			PhysicalHeight: int(cSlice[i].physical_height),
		}
	}

	return slice, nil
}
