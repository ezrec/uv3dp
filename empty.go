//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
)

type EmptyPrintable struct {
	properties Properties
}

func NewEmptyPrintable(prop Properties) (empty *EmptyPrintable) {
	empty = &EmptyPrintable{
		properties: prop,
	}

	return
}

func (empty *EmptyPrintable) Properties() (prop Properties) {
	prop = empty.properties

	return
}

func (empty *EmptyPrintable) Layer(index int) (layer Layer) {
	prop := &empty.properties

	layer.Z = prop.Size.LayerHeight * float32(index)
	layer.Image = image.NewGray(empty.properties.Bounds())

	if index < prop.Bottom.Count {
		layer.Exposure = &prop.Bottom.Exposure
	} else {
		layer.Exposure = &prop.Exposure
	}

	return
}
