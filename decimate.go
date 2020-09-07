//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
)

type DecimatedPrintable struct {
	Printable
	Passes     int // Number of passes of decimation
	FirstLayer int // First layer to start decimating
	Layers     int // Count of layers to decimate
}

func NewDecimatedPrintable(printable Printable) (dp *DecimatedPrintable) {
	dp = &DecimatedPrintable{
		Printable:  printable,
		Passes:     1,
		FirstLayer: 0,
		Layers:     printable.Size().Layers,
	}

	return
}

func (dec *DecimatedPrintable) LayerImage(index int) (ig *image.Gray) {
	ig = dec.Printable.LayerImage(index)

	if index >= dec.FirstLayer && ((index - dec.FirstLayer) < dec.Layers) {
		for pass := 0; pass < dec.Passes; pass++ {
			ig = decimateGray(ig)
		}
	}

	return
}

// Sum an image
func sumImage(sum *image.Gray, gm *image.Gray, dx int, dy int) {
	size := sum.Bounds().Size()

	min := func(a int, b int) int {
		if a < b {
			return a
		}
		return b
	}

	for y := 0; y < size.Y; y++ {
		n := y * sum.Stride
		dst := sum.Pix[n : n+size.X]

		ny := y + dy
		// Handle top and bottom rows
		if ny < 0 || ny >= size.Y {
			for x := 0; x < size.X; x++ {
				dst[x] += 1
			}
		} else {
			src := gm.Pix[ny*gm.Stride : (ny+1)*gm.Stride]
			switch {
			case dx < 0:
				src = src[1:]
				dst[len(dst)-1] += 1
			case dx > 0:
				dst[0] += 1
				dst = dst[1:]
			}

			size := min(len(src), len(dst))
			for x := 0; x < size; x++ {
				if src[x] > 127 {
					dst[x]++
				}
			}
		}
	}
}

// Decimate the layer
// Assumptions:
//   - border outside of image is 'all on'
//   - to remain on, a pixel must be surrounded by 8 pixels
func decimateGray(in *image.Gray) (gm *image.Gray) {
	size := in.Bounds().Size()
	gm = &image.Gray{
		Stride: size.X,
		Pix:    make([]uint8, size.X*size.Y),
		Rect:   in.Bounds(),
	}

	for y := -1; y <= 1; y++ {
		for x := -1; x <= 1; x++ {
			sumImage(gm, in, x, y)
		}
	}

	for n := 0; n < size.X*size.Y; n++ {
		if gm.Pix[n] > 8 {
			gm.Pix[n] = 0xff
		} else {
			gm.Pix[n] = 0x00
		}
	}

	return
}
