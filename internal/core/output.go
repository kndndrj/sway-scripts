package core

import (
	"context"
	"fmt"
	"maps"

	"github.com/joshuarubin/go-sway"
)

// Output represents physical and pixel dimensions of a monitor.
type Output struct {
	Name           string
	Width          int
	Height         int
	PhysicalWidth  int
	PhysicalHeight int
	X              int
	Y              int
}

type physicalDimensions struct {
	Name           string
	PhysicalWidth  int
	PhysicalHeight int
}

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

// Get returns the specified wayland output
func (c *OutputCache) Get(ctx context.Context, name string) (*Output, error) {
	// if cache is valid, return the value
	if c.isValid {
		if out, ok := c.lookup[name]; ok {
			return out, nil
		}
	}

	// if cache is invalid, or the value was not found, update the cache.
	lookup, err := c.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// update the map with new keys
	maps.Copy(c.lookup, lookup)
	c.isValid = true

	if out, ok := c.lookup[name]; ok {
		return out, nil
	}
	return nil, fmt.Errorf("output %q not found", name)
}

func (c *OutputCache) fetch(ctx context.Context) (map[string]*Output, error) {
	// fetch physical dimensions from wayland client
	physical, err := fetchOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed fetching outputs: %w", err)
	}

	// get resolutions from sway
	outs, err := c.swayCl.GetOutputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("c.swayCl.GetOutputs: %w", err)
	}

	phys := make(map[string]*physicalDimensions, len(outs))
	for _, p := range physical {
		phys[p.Name] = p
	}

	// merge
	lookup := make(map[string]*Output)
	for _, o := range outs {
		p, ok := phys[o.Name]
		if !ok {
			continue
		}

		lookup[o.Name] = &Output{
			Name:           o.Name,
			Width:          int(o.Rect.Width),
			Height:         int(o.Rect.Height),
			PhysicalWidth:  p.PhysicalWidth,
			PhysicalHeight: p.PhysicalHeight,
			X:              int(o.Rect.X),
			Y:              int(o.Rect.Y),
		}
	}

	return lookup, nil
}

func (c *OutputCache) Invalidate() {
	c.isValid = false
}
