//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package pws

import (
	"fmt"
	"image"
	"image/color"

	"hash/crc64"
)

const (
	rle1EncodingLimit = 0x7d // Yah, I know. Feels weird. But required.
)

var tab64 *crc64.Table

func hash64(data []byte) (hash uint64) {
	if tab64 == nil {
		tab64 = crc64.MakeTable(crc64.ECMA)
	}

	hash = crc64.Checksum(data, tab64)
	return
}

func rle1EncodeBitmap(bm image.Image, level, levels int) (rle []byte, hash uint64, bitsOn uint) {
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
				if rep == rle1EncodingLimit {
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

func rle1DecodeInto(pix []uint8, rle []byte) (data []byte, err error) {
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

		if n == len(pix) {
			break
		}

		if n > len(pix) {
			err = fmt.Errorf("image ran off the end: %v(%v) of %v", n-reps, reps, len(pix))
			return
		}
	}

	if n != len(pix) {
		err = fmt.Errorf("image ended short: %v of %v", n, len(pix))
		return
	}

	data = rle[index+1:]

	return
}

func rle1DecodeBitmaps(bounds image.Rectangle, rle []byte, levels int) (gm *image.Gray, err error) {
	switch levels {
	case 1:
	case 2:
	case 4:
	case 8:
	default:
		err = fmt.Errorf("invalid Anti-Alias image set: %d levels", levels)
		return
	}

	pixSize := bounds.Size().X * bounds.Size().Y

	gm = &image.Gray{
		Pix:    make([]uint8, pixSize),
		Stride: bounds.Size().X,
		Rect:   bounds,
	}

	for level := 0; level < levels; level++ {
		rle, err = rle1DecodeInto(gm.Pix, rle)
		if err != nil {
			err = fmt.Errorf("antialias %v/%v: %w", level, levels, err)
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
