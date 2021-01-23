//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package ctb

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"

	"testing"

	"github.com/ezrec/uv3dp"
	"github.com/google/go-cmp/cmp"
)

var (
	// Collect an empty printable
	emptyPrintable = &uv3dp.Print{uv3dp.Properties{
		Size: uv3dp.Size{
			X: 10,
			Y: 20,
			Millimeter: uv3dp.SizeMillimeter{
				X: 20.0,
				Y: 40.0,
			},
			Layers:      4, // 2 bottom, 2 normal
			LayerHeight: 0.05,
		},
		Exposure: uv3dp.Exposure{
			LightOnTime:   16.500,
			LightOffTime:  2.250,
			LightPWM:      255,
			LiftHeight:    5.5,
			LiftSpeed:     120.0,
			RetractHeight: defaultRetractHeight, // field cannot be saved by CTB format
			RetractSpeed:  200.0,
		},
		Bottom: uv3dp.Bottom{
			Count: 2,
			Exposure: uv3dp.Exposure{
				LightOnTime:   16.500,
				LightOffTime:  2.250,
				LightPWM:      255,
				LiftHeight:    5.5,
				LiftSpeed:     120.0,
				RetractHeight: defaultRetractHeight, // field cannot be saved by CTB format
				RetractSpeed:  200.0,
			},
		},
		Preview: map[uv3dp.PreviewType]image.Image{
			uv3dp.PreviewTypeTiny: image.NewRGBA(image.Rect(0, 0, 10, 10)),
			uv3dp.PreviewTypeHuge: image.NewCMYK(image.Rect(0, 0, 20, 12)),
		},
	}}

	emptyRaw = []byte{0x86, 0x0, 0xfd, 0x12, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0xa0, 0x41, 0x0, 0x0, 0x20, 0x42, 0x0, 0x0, 0x1b, 0x43, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcd, 0xcc, 0x4c, 0x3e, 0xcd, 0xcc, 0x4c, 0x3d, 0x0, 0x0, 0x84, 0x41, 0x0, 0x0, 0x84, 0x41, 0x0, 0x0, 0x10, 0x40, 0x2, 0x0, 0x0, 0x0, 0xa, 0x0, 0x0, 0x0, 0x14, 0x0, 0x0, 0x0, 0x70, 0x0, 0x0, 0x0, 0x47, 0x1, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x94, 0x0, 0x0, 0x0, 0x6a, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0xb8, 0x0, 0x0, 0x0, 0x3c, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0xff, 0x0, 0xff, 0x0, 0x42, 0x4, 0xcb, 0x9a, 0xf4, 0x0, 0x0, 0x0, 0x4c, 0x0, 0x0, 0x0, 0x14, 0x0, 0x0, 0x0, 0xc, 0x0, 0x0, 0x0, 0x90, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xef, 0x30, 0xa, 0x0, 0x0, 0x0, 0xa, 0x0, 0x0, 0x0, 0xb4, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x0, 0x63, 0x30, 0x0, 0x0, 0xb0, 0x40, 0x0, 0x0, 0xf0, 0x42, 0x0, 0x0, 0xb0, 0x40, 0x0, 0x0, 0xf0, 0x42, 0x0, 0x0, 0x48, 0x43, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x10, 0x40, 0x0, 0x0, 0x10, 0x40, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x00, 0x00, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x40, 0x1, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0, 0x78, 0x56, 0x34, 0x12, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x7, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0xcd, 0xcc, 0x4c, 0x3d, 0x0, 0x0, 0x84, 0x41, 0x0, 0x0, 0x10, 0x40, 0xd7, 0x1, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcd, 0xcc, 0xcc, 0x3d, 0x0, 0x0, 0x84, 0x41, 0x0, 0x0, 0x10, 0x40, 0xda, 0x1, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x9a, 0x99, 0x19, 0x3e, 0x0, 0x0, 0x84, 0x41, 0x0, 0x0, 0x10, 0x40, 0xdd, 0x1, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xcd, 0xcc, 0x4c, 0x3e, 0x0, 0x0, 0x84, 0x41, 0x0, 0x0, 0x10, 0x40, 0xe0, 0x1, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x61, 0x23, 0x7e, 0x35, 0x46, 0xfd, 0xa, 0xf9, 0x7c, 0xde, 0x1c}
)

type bufferMap struct {
	Buffer []byte
	Offset int64
}

func (bm *bufferMap) ReadAt(buff []byte, off int64) (size int, err error) {
	size = copy(buff, bm.Buffer[off:])
	if len(buff) > 0 && size == 0 {
		err = io.EOF
	}
	return
}

func (bm *bufferMap) Read(buff []byte) (size int, err error) {
	size, err = bm.ReadAt(buff, bm.Offset)
	if err != nil {
		return
	}
	bm.Offset += int64(size)
	return
}

func TestEmptyToRaw(t *testing.T) {

	table := []struct {
		Format string
		Raw    []byte
	}{
		{".ctb", emptyRaw},
	}

	for _, item := range table {
		formatter := NewFormatter(item.Format)
		formatter.Version = 2

		buffWriter := &bytes.Buffer{}
		formatter.Encode(buffWriter, emptyPrintable)

		encoded_buff := buffWriter.Bytes()

		emptyRaw := item.Raw
		if !bytes.Equal(encoded_buff, emptyRaw) {
			t.Logf("%+#v\n", encoded_buff)
			if len(encoded_buff) == len(emptyRaw) {
				for i := 0; i < len(emptyRaw)-3; i += 4 {
					a := binary.LittleEndian.Uint32(emptyRaw[i : i+4])
					b := binary.LittleEndian.Uint32(encoded_buff[i : i+4])
					if a != b {
						t.Logf("[%05x] %08x != %08x", i, a, b)
					}
				}
			}
			t.Errorf("%v: expected [%d byte encoding], got [%d byte encoding]", item.Format, len(emptyRaw), len(encoded_buff))
		}
	}
}

func fixup(prop *uv3dp.Properties) {
	// Fill in all the properties that the V2 (.ctb) format can't save
	prop.Exposure.RetractHeight = defaultRetractHeight
	prop.Bottom.Exposure.RetractHeight = defaultRetractHeight
	prop.Preview = nil
}

func imageEqual(a, b image.Image) bool {
	aRect := a.Bounds()
	bRect := b.Bounds()

	if !cmp.Equal(aRect, bRect) {
		return false
	}

	for y := aRect.Min.Y; y < aRect.Max.Y; y++ {
		for x := aRect.Min.X; x < aRect.Max.X; x++ {
			aColor := a.At(x, y)
			bColor := b.At(x, y)
			if !cmp.Equal(aColor, bColor) {
				return false
			}
		}
	}

	return true
}

func TestRawToEmpty(t *testing.T) {
	table := []struct {
		Format string
		Raw    []byte
		Fixup  func(*uv3dp.Properties)
	}{
		{Format: ".ctb", Raw: emptyRaw, Fixup: fixup},
	}

	for _, item := range table {
		formatter := NewFormatter(item.Format)

		emptyRaw := item.Raw
		buffReader := &bufferMap{Buffer: emptyRaw}

		result, err := formatter.Decode(buffReader, int64(len(emptyRaw)))
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}

		eProp := uv3dp.Properties{
			Size:     emptyPrintable.Size(),
			Exposure: emptyPrintable.Exposure(),
			Bottom:   emptyPrintable.Bottom(),
		}
		rProp := uv3dp.Properties{
			Size:     result.Size(),
			Exposure: result.Exposure(),
			Bottom:   result.Bottom(),
		}

		// Remote non-savable values
		item.Fixup(&eProp)
		item.Fixup(&rProp)

		if !cmp.Equal(eProp, rProp) {
			t.Errorf("expected input printable to match expected printable!")
			t.Logf("%+v", eProp)
			t.Logf("%+v", rProp)
		}

		for n := 0; n < eProp.Size.Layers; n++ {
			rLayerZ := result.LayerZ(n)
			eLayerZ := emptyPrintable.LayerZ(n)

			if rLayerZ != eLayerZ {
				t.Errorf("layer %d: expected Z %f did not match result Z %f", n, eLayerZ, rLayerZ)
			}

			rLayerExposure := result.LayerExposure(n)
			eLayerExposure := emptyPrintable.LayerExposure(n)

			if !cmp.Equal(eLayerExposure, rLayerExposure) {
				t.Errorf("layer %d: expected exposure did not match result exposure", n)
				t.Logf("expect: %+v", eLayerExposure)
				t.Logf("result: %+v", rLayerExposure)
			}

			rLayerImage := result.LayerImage(n)
			eLayerImage := emptyPrintable.LayerImage(n)

			if !imageEqual(eLayerImage, rLayerImage) {
				t.Errorf("layer %d: images did not match", n)
			}
		}
	}
}
