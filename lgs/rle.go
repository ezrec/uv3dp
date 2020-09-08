//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package lgs handles input and output of Longer Orange 10 print files
package lgs

import (
	"fmt"
	"image"
	"image/color"
)

func Rle4Encode(pic *image.Gray) (data []byte, err error) {
	bounds := pic.Bounds()

	addSpan := func(color uint8, span uint) (out []byte) {
		for ; span > 0; span >>= 4 {
			datum := uint8(span&0xf) | (color & 0xf0)
			out = append([]byte{datum}, out...)
		}
		return
	}

	span := uint(0)
	lc := uint8(0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := pic.GrayAt(x, y).Y & 0xf0
			if c == lc {
				span++
			} else {
				data = append(data, addSpan(lc, span)...)
				span = 1
			}
			lc = c
		}
	}

	data = append(data, addSpan(lc, span)...)

	return
}

func Rle4Decode(data []byte, bounds image.Rectangle) (gi *image.Gray) {

	gi = image.NewGray(bounds)

	last := uint8(0)
	span := 0
	index := 0

	addSpan := func(color uint8, span int) {
		for ; span > 0; span-- {
			if index >= len(gi.Pix) {
				panic(fmt.Sprintf("%v bytes too many", span))
				return
			}
			gi.Pix[index] = color
			index++
		}
	}

	for _, b := range data {
		color := (b & 0xf0) | (b >> 4)
		if color == last {
			span = (span << 4) | int(b&0xf)
		} else {
			addSpan(last, span)
			span = int(b & 0xf)
		}
		last = color
	}

	addSpan(last, span)

	if index != len(gi.Pix) {
		panic(fmt.Sprintf("%v bytes missing of %v\n", len(gi.Pix)-index, len(gi.Pix)))
	}

	return
}

func RGB15Encode(pic image.Image) (data []byte) {
	bounds := pic.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(pic.At(x, y)).(color.NRGBA)
			rgb15 := (uint16(c.R>>3) << 11) |
				(uint16(c.G>>3) << 6) |
				(uint16(c.B>>3) << 0)
			data = append(data, uint8(rgb15>>8), uint8(rgb15&0xff))
		}
	}

	return
}

func RGB15Decode(bounds image.Rectangle, data []byte) (pic image.Image) {
	size := bounds.Size()

	preview := image.NewNRGBA(bounds)

	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			n := (y*size.X + x) * 2
			rgb15 := (uint16(data[n+0]) << 8) | uint16(data[n+1])
			r := ((uint8(rgb15>>11) & 0x1f) << 3) | 0x7
			g := ((uint8(rgb15>>6) & 0x1f) << 3) | 0x7
			b := ((uint8(rgb15>>0) & 0x1f) << 3) | 0x7
			preview.Set(x, y, color.NRGBA{r, g, b, 0xff})
		}
	}

	pic = preview

	return
}
