package core

import (
	"context"
	"fmt"

	"github.com/neurlang/wayland/wl"
	"github.com/neurlang/wayland/wlclient"
)

// fetchOutputs returns an up to date info about outputs.
func fetchOutputs(_ context.Context) ([]*physicalDimensions, error) {
	// get physical sizes from wayland directly
	display, err := wlclient.DisplayConnect(nil)
	if err != nil {
		return nil, fmt.Errorf("wlclient.DisplayConnect: %w", err)
	}
	defer wlclient.DisplayDisconnect(display)

	registry, err := wlclient.DisplayGetRegistry(display)
	if err != nil {
		return nil, fmt.Errorf("wlclient.DisplayGetRegistry: %w", err)
	}
	defer wlclient.RegistryDestroy(registry)

	listener := &registryListener{
		registry:       registry,
		outputListener: &outputListener{},
	}

	wlclient.RegistryAddListener(registry, listener)

	err = wlclient.DisplayDispatch(display)
	if err != nil {
		return nil, fmt.Errorf("wlclient.DisplayDispatch: %w", err)
	}
	// first roundtrip triggers registry listener
	err = wlclient.DisplayRoundtrip(display)
	if err != nil {
		return nil, fmt.Errorf("wlclient.DisplayRoundtrip: %w", err)
	}
	// second roundtrip triggers output listener
	err = wlclient.DisplayRoundtrip(display)
	if err != nil {
		return nil, fmt.Errorf("wlclient.DisplayRoundtrip: %w", err)
	}

	return listener.outputListener.collect(), nil
}

var (
	_ wl.OutputGeometryHandler = (*outputListener)(nil)
	_ wl.OutputNameHandler     = (*outputListener)(nil)
)

type outputListener struct {
	index      int                   // current index in list
	dimensions []*physicalDimensions // list of outputs
	doneFlangs int                   // when this is equal to number of event handlers (e.g. 2), index is incremented
}

// prepareNextOutput prepares next output and returns a reference to it.
func (ol *outputListener) prepareNextOutput() *physicalDimensions {
	ol.doneFlangs += 1

	if ol.doneFlangs > 2 {
		ol.index += 1
		ol.doneFlangs = 0
	}

	if len(ol.dimensions) < ol.index+1 {
		ol.dimensions = append(ol.dimensions, &physicalDimensions{})
	}

	return ol.dimensions[len(ol.dimensions)-1]
}

func (ol *outputListener) collect() []*physicalDimensions {
	return ol.dimensions
}

func (ol *outputListener) HandleOutputGeometry(e wl.OutputGeometryEvent) {
	dim := ol.prepareNextOutput()
	dim.PhysicalWidth = int(e.PhysicalWidth)
	dim.PhysicalHeight = int(e.PhysicalHeight)
}

func (ol *outputListener) HandleOutputName(e wl.OutputNameEvent) {
	dim := ol.prepareNextOutput()
	dim.Name = e.Name
}

var _ wlclient.RegistryListener = (*registryListener)(nil)

type registryListener struct {
	registry       *wl.Registry
	outputListener *outputListener
}

func (rl *registryListener) HandleRegistryGlobal(e wl.RegistryGlobalEvent) {
	if e.Interface != "wl_output" {
		return
	}

	out := wlclient.RegistryBindOutputInterface(rl.registry, e.Name, e.Version)

	out.AddGeometryHandler(rl.outputListener)
	out.AddNameHandler(rl.outputListener)
}

func (rl *registryListener) HandleRegistryGlobalRemove(wl.RegistryGlobalRemoveEvent) {}
