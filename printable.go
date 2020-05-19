//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
	"runtime"
)

// Everything needed to print a single layer
type Layer struct {
	Z        float32     // Z height in mm
	Exposure *Exposure   // Layer exposure settings
	Image    *image.Gray // Image mask
}

type Printable interface {
	Properties() (prop Properties)
	Layer(index int) (layer Layer)
}

// WithAllLayers executes a function in parallel over all of the layers
func WithAllLayers(p Printable, do func(n int, layer Layer)) {
	layers := p.Properties().Size.Layers

	guard := make(chan struct{}, runtime.GOMAXPROCS(0))
	for n := 0; n < layers; n++ {
		guard <- struct{}{}
		go func(p Printable, do func(n int, layer Layer), n int) {
			layer := p.Layer(n)
			do(n, layer)
			layer.Image = nil
			runtime.GC()
			<-guard
		}(p, do, n)
	}
}
