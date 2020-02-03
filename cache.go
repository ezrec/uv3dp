//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

type CachedPrintable struct {
	Printable

	cacheDepth int
	layerCache map[int]Layer
}

func NewCachedPrintable(printable Printable, cacheDepth int) (cp *CachedPrintable) {
	cp = &CachedPrintable{
		Printable:  printable,
		layerCache: make(map[int]Layer, cacheDepth),
		cacheDepth: cacheDepth,
	}
	return
}

func (cp *CachedPrintable) Layer(index int) (layer Layer) {
	layer, found := cp.layerCache[index]

	if !found {
		if len(cp.layerCache) >= cp.cacheDepth {
			for key := range cp.layerCache {
				delete(cp.layerCache, key)
				break
			}
		}

		layer = cp.Printable.Layer(index)

		cp.layerCache[index] = layer
	}

	return
}
