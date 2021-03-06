//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package uv3dp is a set of tools for data exchange between UV Resin based 3D printers
package cbddlp

import (
	"image"
	"testing"
)

func TestDecodeBinary(t *testing.T) {
	in_rle := []byte{
		0x00, // No 0 bits
		0x80, // No 1 bits
		0x08, // 8 0 bits
		0x88, // 8 1 bits
	}
	out_gray := []uint8{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
	rect := image.Rect(0, 0, 8, 2)

	var gm *image.Gray
	gm, err := rleDecodeBitmaps(rect, []([]byte){in_rle})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if gm.Stride != rect.Size().X {
		t.Fatalf("expected 8, got %v", gm.Stride)
	}

	if len(gm.Pix) != len(out_gray) {
		t.Fatalf("expected %v, got %v", len(out_gray), len(gm.Pix))
	}

	for n, v := range out_gray {
		if gm.Pix[n] != v {
			t.Errorf("%v: expected %v, got %v", n, v, gm.Pix[n])
		}
	}
}

func TestEncodeBinary(t *testing.T) {
	rect := image.Rect(0, 0, 8, 21)
	gray := &image.Gray{
		Pix: []uint8{
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 00
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 08
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 10
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 18
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 20
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 28
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 30
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 38
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 40
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 48
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 50
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 58
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 60
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 68
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 70
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 78
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 80
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // 88
			0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, // 90
			0xff, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff, 0x00, // 98
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // a0
		},
		Stride: rect.Size().X,
		Rect:   rect,
	}

	out_rle := []byte{
		0x88, // 8 1 bits
		0x08, // 8 0 bits
		0xfd, // 125 1 bits
		0x88, // 6 1 bits
		0x02, // 2 0 bits
		0x82, // 2 1 bits
		0x01, // 1 0 bits
		0x81, // 1 1 bits
		0x01, // 1 0 bits
		0x81, // 1 1 bits
		0x01, // 1 0 bits
		0x81, // 1 1 bits
		0x09, // 9 0 bits
	}
	out_hash := uint64(0x2b7a251cf0df82f0)
	out_bits := uint(146)

	rle, hash, bits := rleEncodeBitmap(gray, 0, 1)

	if bits != out_bits {
		t.Errorf("expected %v, got %v", out_bits, bits)
	}

	if out_hash != hash {
		t.Errorf("expected %#v, got %#v", out_hash, hash)
	}

	if len(rle) != len(out_rle) {
		t.Fatalf("expected %v, got %v", len(out_rle), len(rle))
	}

	for n, b := range out_rle {
		if rle[n] != b {
			t.Errorf("%v: expected %#v, got %#v", n, b, rle[n])
		}
	}

	// All empty
	rect = image.Rect(0, 0, 127, 4)
	gray.Rect = rect
	gray.Stride = rect.Size().X
	gray.Pix = make([]byte, rect.Size().X*rect.Size().Y)

	rle, hash, bits = rleEncodeBitmap(gray, 0, 1)

	out_rle = []byte{0x7d, 0x7d, 0x7d, 0x7d, 0x08}
	out_hash = uint64(0x174c6ac17d4207cf)
	out_bits = 0

	if out_bits != bits {
		t.Errorf("expected %v, got %v", out_bits, bits)
	}

	if out_hash != hash {
		t.Errorf("expected %#v, got %#v", out_hash, hash)
	}

	if len(rle) != len(out_rle) {
		t.Fatalf("expected %v, got %v [% 02x]", len(out_rle), len(rle), rle)
	}

	for n, b := range out_rle {
		if rle[n] != b {
			t.Errorf("%v: expected %#v, got %#v", n, b, rle[n])
		}
	}

}
