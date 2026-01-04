//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package cbddlp

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

func rleEncodeBitmap(bm image.Image, level, levels int) (rle []byte, hash uint64, bitsOn uint) {
	base := bm.Bounds().Min
	size := bm.Bounds().Size()

	addRep := func(bit bool, rep int) {
		if rep > 0 {
			by := uint8(rep)
			if bit {
				by |= 0x80
				bitsOn += uint(rep)
			}
			rle = append(rle, by)
		}
	}

	// thresholds:
	// aa 1:  127
	// aa 2:  255 127
	// aa 4:  255 191 127 63
	// aa 8:  255 223 191 159 127 95 63 31
	threshold := byte((int(256/levels) * level) - 1)

	obit := false
	rep := 0
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			c := bm.At(base.X+x, base.Y+y)
			ngrey := color.GrayModel.Convert(c).(color.Gray).Y
			nbit := ngrey >= threshold
			if nbit == obit {
				rep++
				if rep == rle8EncodingLimit {
					addRep(obit, rep)
					rep = 0
				}
			} else {
				addRep(obit, rep)
				obit = nbit
				rep = 1
			}
		}
	}

	// Collect stragglers
	addRep(obit, rep)

	hash = hash64(rle)

	return
}

func rleDecodeInto(pix []uint8, rle []byte) (err error) {
	var index int
	var b byte

	n := 0
	for index, b = range rle {
		// Lower 7 bits is the repeat count for the bit (0..127)
		reps := int(b & 0x7f)

		// We only need to set the non-zero pixels
		// High bit is on for white, off for black
		if (b & 0x80) != 0 {
			for i := 0; i < reps; i++ {
				pix[n+i]++
			}
		}
		n += reps
	}

	if index != len(rle)-1 {
		err = fmt.Errorf("what? bytes left: %d", len(rle)-index-1)
		return
	}

	return
}

func rleDecodeBitmaps(bounds image.Rectangle, rleSet []([]byte)) (gm *image.Gray, err error) {
	levels := len(rleSet)

	pixSize := bounds.Size().X * bounds.Size().Y

	gm = &image.Gray{
		Pix:    make([]uint8, pixSize),
		Stride: bounds.Size().X,
		Rect:   bounds,
	}

	for _, rle := range rleSet {
		err = rleDecodeInto(gm.Pix, rle)
		if err != nil {
			return
		}
	}

	// Convert counts into colors
	for n, c := range gm.Pix {
		newC := int(c) * (256 / levels)
		if newC > 0 {
			newC--
		}
		gm.Pix[n] = uint8(newC)
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
