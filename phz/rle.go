//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package phz

import (
	"fmt"
	"image"
	"image/color"

	"encoding/binary"
	"hash/crc64"
)

const (
	rle8EncodingLimit  = 125 // Yah, I know. Feels weird. But required.
	rle16EncodingLimit = 0xfff
)

var tab64 *crc64.Table

func hash64(data []byte) (hash uint64) {
	if tab64 == nil {
		tab64 = crc64.MakeTable(crc64.ECMA)
	}

	hash = crc64.Checksum(data, tab64)
	return
}

func rleEncodeGraymap(bm image.Image) (rle []byte, hash uint64, bitsOn uint) {
	base := bm.Bounds().Min
	size := bm.Bounds().Size()

	addRep := func(gray7 uint8, stride int) {
		if gray7 > 0 {
			bitsOn += uint(stride)
		}

		rle = append(rle, gray7|0x80)
		stride--
		for done := 0; done < stride; {
			todo := 0x7d
			if (stride - done) < todo {
				todo = (stride - done)
			}

			rle = append(rle, byte(todo))

			done += todo
		}
	}

	color := byte(0xff)
	var stride int

	for y := 0; y < size.Y; y++ {
		var grey7 uint8
		for x := 0; x < size.X; x++ {
			c := bm.At(base.X+x, base.Y+y)
			r, g, b, _ := c.RGBA()
			// 7 bits per pixel (clamped to 0..0x7c)
			grey7 = uint8(uint16(r|g|b)>>9) & 0x7f
			if grey7 > 0x7c {
				grey7 = 0x7c
			}

			switch {
			case color == 0xff:
				color = grey7
				stride = 1
			case grey7 != color || x == size.X/2:
				addRep(color, stride)
				color = grey7
				stride = 1
			default:
				stride++
			}
		}
		addRep(color, stride)
		color = 0xff
	}

	hash = hash64(rle)

	return
}

func rleDecodeGraymap(bounds image.Rectangle, rle []byte) (gm *image.Gray, err error) {
	limit := bounds.Size().X * bounds.Size().Y
	pix := make([]byte, limit)

	var index int
	var lastColor byte

	for n := 0; n < len(rle); n++ {
		code := rle[n]
		if (code & 0x80) == 0x80 {
			// Convert from 0..124 to 8bpp
			lastColor = ((code & 0x7f) << 1) | (code & 1)
			if lastColor >= 0xfc {
				// Make 'white' actually white
				lastColor = 0xff
			}
			if index < limit {
				pix[index] = lastColor
			}
			index++
		} else {
			for i := 0; i < int(code); i++ {
				if index < limit {
					pix[index] = lastColor
				}
				index++
			}
		}
	}

	if index != limit {
		err = fmt.Errorf("expected %v pixels, saw %v", limit, index)
		return
	}

	gm = &image.Gray{
		Pix:    pix,
		Stride: bounds.Size().X,
		Rect:   bounds,
	}

	return
}

func color5to8(c5 uint16) (c8 uint8) {
	return uint8((c5 << 3) | (c5 >> 2))
}

func color16to5(c16 uint32) (c5 uint16) {
	return uint16((c16 >> (16 - 5)) & 0x1f)
}

const repeatRGB15Mask = uint16(1 << 5)

func rleRGB15(color15 uint16, rep int) (rle []byte) {
	switch rep {
	case 0:
		// pass...
	case 1:
		data := [2]byte{}
		binary.LittleEndian.PutUint16(data[0:2], color15)
		rle = data[:]
	case 2:
		data := [4]byte{}
		binary.LittleEndian.PutUint16(data[0:2], color15)
		binary.LittleEndian.PutUint16(data[2:4], color15)
		rle = data[:]
	default:
		data := [4]byte{}
		binary.LittleEndian.PutUint16(data[0:2], color15|repeatRGB15Mask)
		binary.LittleEndian.PutUint16(data[2:4], uint16(rep-1)|(0x3000))
		rle = data[:]
	}

	return
}

func rleEncodeRGB15(bm image.Image) (rle []byte, hash uint64) {
	base := bm.Bounds().Min
	size := bm.Bounds().Size()

	color15 := uint16(0)
	rep := 0
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			ncR, ncG, ncB, _ := bm.At(base.X+x, base.Y+y).RGBA()
			ncolor15 := color16to5(ncB)
			ncolor15 |= color16to5(ncG) << 6
			ncolor15 |= color16to5(ncR) << 11
			if ncolor15 == color15 {
				rep++
				if rep == rle16EncodingLimit {
					rle = append(rle, rleRGB15(color15, rep)...)
					rep = 0
				}
			} else {
				rle = append(rle, rleRGB15(color15, rep)...)
				color15 = ncolor15
				rep = 1
			}
		}
	}

	rle = append(rle, rleRGB15(color15, rep)...)

	hash = hash64(rle)

	return
}

func rleDecodeRGB15(bounds image.Rectangle, rle []byte) (view *image.RGBA, err error) {
	view = image.NewRGBA(bounds)

	y := bounds.Min.Y
	x := bounds.Min.X
	for n := 0; n < len(rle); n += 2 {
		color16 := binary.LittleEndian.Uint16(rle[n : n+2])
		repeat := int(1)
		if (color16 & repeatRGB15Mask) != 0 {
			n += 2
			repeat += int(binary.LittleEndian.Uint16(rle[n:n+2]) & 0xfff)
		}

		colorRgba := color.RGBA{
			R: color5to8((color16 >> 11) & 0x1f),
			G: color5to8((color16 >> 6) & 0x1f),
			B: color5to8((color16 >> 0) & 0x1f),
			A: 255,
		}

		for r := 0; r < repeat; r++ {
			view.Set(x, y, colorRgba)
			x++
			if x == bounds.Max.X {
				x = bounds.Min.X
				y++
			}
		}
	}

	return
}
