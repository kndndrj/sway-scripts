package output

import (
	"fmt"

	"github.com/neurlang/wayland/wl"
	"github.com/neurlang/wayland/wlclient"
)

type Output struct {
	Name           string
	Width          int
	Height         int
	PhysicalWidth  int
	PhysicalHeight int
}

var (
	_ wl.OutputGeometryHandler = (*outputListener)(nil)
	_ wl.OutputModeHandler     = (*outputListener)(nil)
	_ wl.OutputNameHandler     = (*outputListener)(nil)
)

type outputListener struct {
	index      int       // current index in list
	outputs    []*Output // list of outputs
	doneFlangs int       // when this is equal to number of event handlers (e.g. 3), index is incremented
}

// prepareNextOutput prepares next output and returns a reference to it.
func (ol *outputListener) prepareNextOutput() *Output {
	ol.doneFlangs += 1

	if ol.doneFlangs > 3 {
		ol.index += 1
		ol.doneFlangs = 0
	}

	if len(ol.outputs) < ol.index+1 {
		ol.outputs = append(ol.outputs, &Output{})
	}

	return ol.outputs[len(ol.outputs)-1]
}

func (ol *outputListener) HandleOutputGeometry(e wl.OutputGeometryEvent) {
	output := ol.prepareNextOutput()
	output.PhysicalWidth = int(e.PhysicalWidth)
	output.PhysicalHeight = int(e.PhysicalHeight)
}

func (ol *outputListener) HandleOutputMode(e wl.OutputModeEvent) {
	output := ol.prepareNextOutput()
	output.Width = int(e.Width)
	output.Height = int(e.Height)
}

func (ol *outputListener) HandleOutputName(e wl.OutputNameEvent) {
	output := ol.prepareNextOutput()
	output.Name = e.Name
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

	output := wlclient.RegistryBindOutputInterface(rl.registry, e.Name, e.Version)

	output.AddGeometryHandler(rl.outputListener)
	output.AddModeHandler(rl.outputListener)
	output.AddNameHandler(rl.outputListener)
}

func (rl *registryListener) HandleRegistryGlobalRemove(wl.RegistryGlobalRemoveEvent) {}

// Get returns all wayland outputs
func Get() ([]*Output, error) {
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

	return listener.outputListener.outputs, nil
}
