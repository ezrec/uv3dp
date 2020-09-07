//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
)

type Print struct {
	Properties
}

func (p *Print) Size() Size {
	return p.Properties.Size
}

func (p *Print) Exposure() Exposure {
	return p.Properties.Exposure
}

func (p *Print) Bottom() Bottom {
	return p.Properties.Bottom
}

func (p *Print) Preview(index PreviewType) (ig image.Image, ok bool) {
	ig, ok = p.Properties.Preview[index]
	return
}

func (p *Print) Metadata(index string) (data interface{}, ok bool) {
	data, ok = p.Properties.Metadata[index]
	return
}

func (p *Print) LayerImage(index int) (ig *image.Gray) {
	return image.NewGray(p.Properties.Bounds())
}

func NewEmptyPrintable(prop Properties) (p *Print) {
	p = &Print{
		Properties: prop,
	}

	return
}
