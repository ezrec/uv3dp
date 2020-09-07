//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package uv3dp is a set of tools for data exchange between UV Resin based 3D printers
package cbddlp

import (
	"bytes"
	"image"

	"testing"

	"github.com/ezrec/uv3dp"
	"github.com/google/go-cmp/cmp"
)

var (
	greyMap = []byte{0x00, 0x1f, 0x3f, 0x5f, 0x7f, 0x9f, 0xbf, 0xdf, 0xff, 0x7f}
	aa1Map  = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00}
	aa2Map  = []byte{0x00, 0x00, 0x00, 0x00, 0x7f, 0x7f, 0x7f, 0x7f, 0xff, 0x7f}
	aa4Map  = []byte{0x00, 0x00, 0x3f, 0x3f, 0x7f, 0x7f, 0xbf, 0xbf, 0xff, 0x7f}
	aa8Map  = []byte{0x00, 0x1f, 0x3f, 0x5f, 0x7f, 0x9f, 0xbf, 0xdf, 0xff, 0x7f}
)

type AliasPrintable struct {
	uv3dp.Printable
}

func (ap *AliasPrintable) LayerImage(index int) (ig *image.Gray) {
	ig = ap.Printable.LayerImage(index)

	ig.Pix = greyMap

	return
}

var (
	// Collect an alias printable
	aliasPrintable = &AliasPrintable{&uv3dp.Print{uv3dp.Properties{
		Size: uv3dp.Size{
			X: 10,
			Y: 1,
			Millimeter: uv3dp.SizeMillimeter{
				X: 10.0,
				Y: 1.0,
			},
			Layers:      1, // 1 normal
			LayerHeight: 0.05,
		},
		Exposure: uv3dp.Exposure{
			LightOnTime:   0.000000001,
			LightOffTime:  0.000000001,
			LightPWM:      255,
			LiftHeight:    1.0,
			LiftSpeed:     1.0,
			RetractHeight: 1.0,
			RetractSpeed:  1.0,
		},
		Bottom: uv3dp.Bottom{
			Count: 1,
			Exposure: uv3dp.Exposure{
				LightOnTime:   0.000000001,
				LightOffTime:  0.000000001,
				LightPWM:      255,
				LiftHeight:    1.0,
				LiftSpeed:     1.0,
				RetractHeight: 1.0,
				RetractSpeed:  1.0,
			},
		},
		Preview: map[uv3dp.PreviewType]image.Image{
			uv3dp.PreviewTypeTiny: image.NewGray(image.Rect(0, 0, 1, 1)),
			uv3dp.PreviewTypeHuge: image.NewGray(image.Rect(0, 0, 1, 1)),
		},
	}}}
)

// reuse 'bufferMap' from format_empty_test.go

func TestAlias(t *testing.T) {
	table := []struct {
		AntiAlias int
		Raw       []byte
		GrayMap   []byte
	}{
		{AntiAlias: 1, GrayMap: aa1Map},
		{AntiAlias: 2, GrayMap: aa2Map},
		{AntiAlias: 4, GrayMap: aa4Map},
		{AntiAlias: 8, GrayMap: aa8Map},
	}

	for n, item := range table {
		formatter := NewFormatter(".cbddlp")
		formatter.AntiAlias = item.AntiAlias

		buffWriter := &bytes.Buffer{}
		formatter.Encode(buffWriter, aliasPrintable)

		encoded_buff := buffWriter.Bytes()

		table[n].Raw = encoded_buff
	}

	for _, item := range table {
		formatter := NewFormatter(".cbddlp")

		aliasRaw := item.Raw
		buffReader := &bufferMap{Buffer: aliasRaw}

		result, err := formatter.Decode(buffReader, int64(len(aliasRaw)))
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}

		eProp := uv3dp.Properties{
			Size:     aliasPrintable.Size(),
			Exposure: aliasPrintable.Exposure(),
			Bottom:   aliasPrintable.Bottom(),
		}
		rProp := uv3dp.Properties{
			Size:     result.Size(),
			Exposure: result.Exposure(),
			Bottom:   result.Bottom(),
		}

		fixupV2(&eProp)
		fixupV2(&rProp)

		if !cmp.Equal(eProp, rProp) {
			t.Errorf("aa%v: expected input printable to match expected printable!", item.AntiAlias)
			t.Logf("expected: %+v", eProp)
			t.Logf("actual  : %+v", rProp)
		}

		rLayer := result.LayerImage(0)
		if !cmp.Equal(rLayer.Pix, item.GrayMap) {
			t.Errorf("aa%v: expected image to exactly match", item.AntiAlias)
			t.Logf("expected: %+#v", item.GrayMap)
			t.Logf("actual  : %+#v", rLayer.Pix)
		}
	}
}
