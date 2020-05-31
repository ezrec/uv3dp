//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package pws

import (
	"fmt"
	"image"

	"encoding/binary"
)

const (
	rle4EncodingLimit = 0xfff
)

// Encodings:
//  0N NN -> Next 0xNNN bits are black
//  fN NN -> Next 0xNNN bits are white
//  1N -> next N bits are ??
//  4N -> next N bits are ??
//  8N -> next N bits are 50% grey
//  9N -> next N bits are ??
//  BN -> next N bits are ??
//  DN -> next N bits are ??
//  End is 16 bit checksum (?)

func rle4EncodeBitmap(bm image.Image, bit, bits int) (rle []byte, hash uint64, bitsOn uint) {
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

	obit := false
	rep := 0
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			c := bm.At(base.X+x, base.Y+y)
			r, g, b, _ := c.RGBA()
			ngrey := uint16(r | g | b)
			nbit := (ngrey & (1 << ((16 - bits) + bit))) != 0
			if nbit == obit {
				rep++
				if rep == rle4EncodingLimit {
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

func rle4DecodeInto(pix []uint8, rle []byte, mask uint8) (data []byte, err error) {
	var index int

	n := 0
	for index = 0; index < len(rle); index++ {
		b := rle[index]
		code := (b >> 4)
		reps := int(b & 0xf)
		var color byte
		switch code {
		case 0x0:
			color = byte(0x00)
			index++
			reps = (reps * 256) + int(rle[index])
		case 0xf:
			color = byte(0xff)
			index++
			reps = (reps * 256) + int(rle[index])
		default:
			color = (code << 4) | code
		}

		color &= mask

		// We only need to set the non-zero pixels
		if color != 0 {
			for i := 0; i < reps; i++ {
				pix[n+i] |= color
			}
		}

		n += reps

		if n == len(pix) {
			index++
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

	data = rle[index:]

	expect := rle4CRC16(rle[:index])

	if len(data) < 2 {
		err = fmt.Errorf("missing expected checksum len(2), got: %+#v", rle)
		return
	}

	check := binary.BigEndian.Uint16(data)

	if check != expect {
		err = fmt.Errorf("checksum expected %04x, got %04x", expect, check)
		return
	}

	data = data[2:]

	return
}

// CRC-16-ANSI (aka CRC-16-IMB) Polynomial: x^16 + x^15 + x^2 + 1
var crc16Table = [256]uint16{
	0x0000, 0xc0c1, 0xc181, 0x0140, 0xc301, 0x03c0, 0x0280, 0xc241,
	0xc601, 0x06c0, 0x0780, 0xc741, 0x0500, 0xc5c1, 0xc481, 0x0440,
	0xcc01, 0x0cc0, 0x0d80, 0xcd41, 0x0f00, 0xcfc1, 0xce81, 0x0e40,
	0x0a00, 0xcac1, 0xcb81, 0x0b40, 0xc901, 0x09c0, 0x0880, 0xc841,
	0xd801, 0x18c0, 0x1980, 0xd941, 0x1b00, 0xdbc1, 0xda81, 0x1a40,
	0x1e00, 0xdec1, 0xdf81, 0x1f40, 0xdd01, 0x1dc0, 0x1c80, 0xdc41,
	0x1400, 0xd4c1, 0xd581, 0x1540, 0xd701, 0x17c0, 0x1680, 0xd641,
	0xd201, 0x12c0, 0x1380, 0xd341, 0x1100, 0xd1c1, 0xd081, 0x1040,
	0xf001, 0x30c0, 0x3180, 0xf141, 0x3300, 0xf3c1, 0xf281, 0x3240,
	0x3600, 0xf6c1, 0xf781, 0x3740, 0xf501, 0x35c0, 0x3480, 0xf441,
	0x3c00, 0xfcc1, 0xfd81, 0x3d40, 0xff01, 0x3fc0, 0x3e80, 0xfe41,
	0xfa01, 0x3ac0, 0x3b80, 0xfb41, 0x3900, 0xf9c1, 0xf881, 0x3840,
	0x2800, 0xe8c1, 0xe981, 0x2940, 0xeb01, 0x2bc0, 0x2a80, 0xea41,
	0xee01, 0x2ec0, 0x2f80, 0xef41, 0x2d00, 0xedc1, 0xec81, 0x2c40,
	0xe401, 0x24c0, 0x2580, 0xe541, 0x2700, 0xe7c1, 0xe681, 0x2640,
	0x2200, 0xe2c1, 0xe381, 0x2340, 0xe101, 0x21c0, 0x2080, 0xe041,
	0xa001, 0x60c0, 0x6180, 0xa141, 0x6300, 0xa3c1, 0xa281, 0x6240,
	0x6600, 0xa6c1, 0xa781, 0x6740, 0xa501, 0x65c0, 0x6480, 0xa441,
	0x6c00, 0xacc1, 0xad81, 0x6d40, 0xaf01, 0x6fc0, 0x6e80, 0xae41,
	0xaa01, 0x6ac0, 0x6b80, 0xab41, 0x6900, 0xa9c1, 0xa881, 0x6840,
	0x7800, 0xb8c1, 0xb981, 0x7940, 0xbb01, 0x7bc0, 0x7a80, 0xba41,
	0xbe01, 0x7ec0, 0x7f80, 0xbf41, 0x7d00, 0xbdc1, 0xbc81, 0x7c40,
	0xb401, 0x74c0, 0x7580, 0xb541, 0x7700, 0xb7c1, 0xb681, 0x7640,
	0x7200, 0xb2c1, 0xb381, 0x7340, 0xb101, 0x71c0, 0x7080, 0xb041,
	0x5000, 0x90c1, 0x9181, 0x5140, 0x9301, 0x53c0, 0x5280, 0x9241,
	0x9601, 0x56c0, 0x5780, 0x9741, 0x5500, 0x95c1, 0x9481, 0x5440,
	0x9c01, 0x5cc0, 0x5d80, 0x9d41, 0x5f00, 0x9fc1, 0x9e81, 0x5e40,
	0x5a00, 0x9ac1, 0x9b81, 0x5b40, 0x9901, 0x59c0, 0x5880, 0x9841,
	0x8801, 0x48c0, 0x4980, 0x8941, 0x4b00, 0x8bc1, 0x8a81, 0x4a40,
	0x4e00, 0x8ec1, 0x8f81, 0x4f40, 0x8d01, 0x4dc0, 0x4c80, 0x8c41,
	0x4400, 0x84c1, 0x8581, 0x4540, 0x8701, 0x47c0, 0x4680, 0x8641,
	0x8201, 0x42c0, 0x4380, 0x8341, 0x4100, 0x81c1, 0x8081, 0x4040,
}

// Joy. After much experimenantation, it looks like AnyCubic uses a CRC16-ANSI
// table - but tried to make it 'better' by double-stirring the input data.
// .. and, of course, this just led to decreased entropy.
//
// Whatever, can't fix stupid.
func rle4CRC16(rle []byte) (crc16 uint16) {
	for n := 0; n < len(rle); n++ {
		b := rle[n]
		crc16 = (crc16 << 8) ^ crc16Table[((crc16>>8)^crc16Table[b])&0xff]
	}

	crc16 = (crc16Table[crc16&0xff] * 0x100) + crc16Table[(crc16>>8)&0xff]

	return
}

func rle4DecodeBitmaps(bounds image.Rectangle, rle []byte, bits int) (gm *image.Gray, err error) {
	pixSize := bounds.Size().X * bounds.Size().Y

	gm = &image.Gray{
		Pix:    make([]uint8, pixSize),
		Stride: bounds.Size().X,
		Rect:   bounds,
	}

	mask := byte(0xff)
	rle, err = rle4DecodeInto(gm.Pix, rle, mask)
	if err != nil {
		return
	}

	if len(rle) != 0 {
		err = fmt.Errorf("extra data after compressed image: %+#v", rle)
		return
	}

	return
}

func rle4EncodeBitmaps(gm *image.Gray, bits int) (rle []byte, err error) {

	lastColor := -1
	reps := 0

	putReps := func(color int, reps int) {
		for reps > 0 {
			done := reps
			if (color == 0) || (color == 0xf) {
				if done > 0xfff {
					done = 0xfff
				}
				more := []byte{0, 0}
				binary.BigEndian.PutUint16(more, uint16(done|(color<<12)))
				rle = append(rle, more...)
			} else {
				if done > 0xf {
					done = 0xf
				}
				rle = append(rle, uint8(done|(color<<4)))
			}

			reps -= done
		}
	}

	for _, b := range gm.Pix {
		color := int(b >> 4)
		if color == lastColor {
			reps++
		} else {
			putReps(lastColor, reps)
			lastColor = color
			reps = 1
		}
	}

	putReps(lastColor, reps)

	crc := []byte{0, 0}
	binary.BigEndian.PutUint16(crc, rle4CRC16(rle))

	rle = append(rle, crc...)

	return
}
