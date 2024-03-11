package output

import (
	"context"
	"fmt"
	"log"

	"github.com/joshuarubin/go-sway"
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
	_ wl.OutputNameHandler     = (*outputListener)(nil)
)

type physicalDimensions struct {
	Name           string
	PhysicalWidth  int
	PhysicalHeight int
}

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

func (ol *outputListener) collect() map[string]*physicalDimensions {
	ret := make(map[string]*physicalDimensions, len(ol.dimensions))
	for _, dim := range ol.dimensions {
		ret[dim.Name] = dim
	}

	return ret
}

func (ol *outputListener) HandleOutputGeometry(e wl.OutputGeometryEvent) {
	output := ol.prepareNextOutput()
	output.PhysicalWidth = int(e.PhysicalWidth)
	output.PhysicalHeight = int(e.PhysicalHeight)
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
	output.AddNameHandler(rl.outputListener)
}

func (rl *registryListener) HandleRegistryGlobalRemove(wl.RegistryGlobalRemoveEvent) {}

// Get returns all wayland outputs
func Get(ctx context.Context) ([]*Output, error) {
	//
	// get physical sizes from wayland directly
	//
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

	physical := listener.outputListener.collect()

	//
	// get resolutions from sway
	//
	client, err := sway.New(ctx)
	if err != nil {
		log.Fatalf("sway.New: %s", err)
	}

	outs, err := client.GetOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("client.GetOutputs: %w", err)
	}

	// merge
	var outputs []*Output
	for _, o := range outs {
		p, ok := physical[o.Name]
		if !ok {
			continue
		}

		outputs = append(outputs, &Output{
			Name:           o.Name,
			Width:          int(o.Rect.Width),
			Height:         int(o.Rect.Height),
			PhysicalWidth:  p.PhysicalWidth,
			PhysicalHeight: p.PhysicalHeight,
		})
	}

	return outputs, nil
}
