//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
	"runtime"
	"sync"
	"time"
)

type Printable interface {
	Size() Size
	Exposure() Exposure
	Bottom() Bottom
	Preview(index PreviewType) (image.Image, bool)
	MetadataKeys() []string
	Metadata(key string) (data interface{}, ok bool)
	LayerZ(index int) float32
	LayerExposure(index int) Exposure
	LayerImage(index int) *image.Gray
}

// WithAllLayers executes a function in parallel over all of the layers
func WithAllLayers(p Printable, do func(p Printable, n int)) {
	layers := p.Size().Layers

	prog := NewProgress(layers)
	defer prog.Close()

	guard := make(chan struct{}, runtime.GOMAXPROCS(0))
	for n := 0; n < layers; n++ {
		guard <- struct{}{}
		go func(p Printable, do func(p Printable, n int), n int) {
			do(p, n)
			prog.Indicate()
			runtime.GC()
			<-guard
		}(p, do, n)
	}
}

// WithEachLayer executes a function in over all of the layers, serially (but possibly out of order)
func WithEachLayer(p Printable, do func(p Printable, n int)) {
	var mutex sync.Mutex

	WithAllLayers(p, func(p Printable, n int) {
		mutex.Lock()
		do(p, n)
		mutex.Unlock()
	})
}

// Get the total print time for a printable
func PrintDuration(p Printable) (duration time.Duration) {
	layers := p.Size().Layers

	for n := 0; n < layers; n++ {
		exposure := p.LayerExposure(n)
		duration += exposure.Duration()
	}

	return
}
