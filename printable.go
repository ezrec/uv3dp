//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
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
