//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"testing"

	"bufio"
	"image"
	"strings"
)

var (
	gm_eye = `Image
XXXXX
XX XX
X X X
XX XX
XXXXX
`
	gm_eye_dec = `Box
X   X



X   X
`

	gm_bottom = `Bottom

 XXX
XXXXX
XXXXX
XXXXX
`

	gm_bottom_dec = `Bottom Decimated


  X
XXXXX
XXXXX
`
)

func grayFrom(desc string) (gm *image.Gray) {
	reader := strings.NewReader(desc)
	scanner := bufio.NewScanner(reader)
	scanner.Scan()

	var lines []string
	stride := 0
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if len(line) > stride {
			stride = len(line)
		}
	}

	pix := make([]uint8, stride*len(lines))
	for y, line := range lines {
		n := y * stride
		for x, c := range line {
			if c != ' ' {
				pix[n+x] = 0xff
			}
		}
	}

	gm = &image.Gray{
		Rect:   image.Rect(0, 0, stride, len(lines)),
		Stride: stride,
		Pix:    pix,
	}

	return
}

func TestDecimate(t *testing.T) {
	table := []struct {
		in  string
		out string
	}{
		{in: gm_bottom, out: gm_bottom_dec},
		{in: gm_eye, out: gm_eye_dec},
	}

	for _, item := range table {
		gm_in := grayFrom(item.in)
		gm_out := grayFrom(item.out)

		val := decimateGray(gm_in)

		for n := 0; n < len(val.Pix); n++ {
			if val.Pix[n] != gm_out.Pix[n] {
				t.Fatalf("%s %d expected %#v, got %#v", item.out, n, gm_out.Pix[n], val.Pix[n])
			}
		}
	}
}
