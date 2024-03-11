package core

import (
	"context"
	"fmt"

	"github.com/joshuarubin/go-sway"
	"github.com/neurlang/wayland/wl"
	"github.com/neurlang/wayland/wlclient"
)

// Output represents physical and pixel dimensions of a monitor.
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

// OutputCache is a cache of wayland output info.
type OutputCache struct {
	swayCl sway.Client
	lookup map[string]*Output

	isValid bool
}

func NewOutputCache(cl sway.Client) *OutputCache {
	return &OutputCache{
		swayCl: cl,
		lookup: make(map[string]*Output),
	}
}

// look returns value from lookup (translates ok to err)
func (c *OutputCache) look(name string) (*Output, error) {
	out, ok := c.lookup[name]
	if !ok {
		return nil, fmt.Errorf("output %q not found", name)
	}
	return out, nil
}

// Get returns the specified wayland output
func (c *OutputCache) Get(ctx context.Context, name string) (*Output, error) {
	// if cache is valid, return the value
	if c.isValid {
		return c.look(name)
	}

	// if cache is invalid, update it
	outs, err := c.fetchOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed updating cache: %w", err)
	}

	c.lookup = outs
	c.isValid = true

	return c.look(name)
}

func (c *OutputCache) Invalidate() {
	c.isValid = false
}

// fetchOutputs returns an up to date info about outputs.
func (c *OutputCache) fetchOutputs(ctx context.Context) (map[string]*Output, error) {
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
	outs, err := c.swayCl.GetOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("c.swayCl.GetOutputs: %w", err)
	}

	// merge
	ret := make(map[string]*Output)
	for _, o := range outs {
		p, ok := physical[o.Name]
		if !ok {
			continue
		}

		ret[o.Name] = &Output{
			Name:           o.Name,
			Width:          int(o.Rect.Width),
			Height:         int(o.Rect.Height),
			PhysicalWidth:  p.PhysicalWidth,
			PhysicalHeight: p.PhysicalHeight,
		}
	}

	return ret, nil
}
