//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package cws is a set of tools for data exchange in the Prusa CWS format
package cws

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"io/ioutil"
	"strings"
	"time"

	"testing"

	"github.com/ezrec/uv3dp"
)

type bufferReader struct {
	data []byte
}

func (br *bufferReader) ReadAt(p []byte, off int64) (n int, err error) {
	copy(p, br.data[off:])
	n = len(p)

	return
}

func (br *bufferReader) Len() int64 {
	return int64(len(br.data))
}

var (
	testProperties = uv3dp.Properties{
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
			LightOnTime:   time.Millisecond * 16500,
			LightOffTime:  time.Millisecond * 2250,
			LiftHeight:    5.5,
			LiftSpeed:     120.0,
			RetractHeight: 3.3,
			RetractSpeed:  200.0,
		},
		Bottom: uv3dp.Bottom{
			Count: 2,
			Style: uv3dp.BottomStyleFade,
			Exposure: uv3dp.Exposure{
				LightOnTime:   time.Millisecond * 16500,
				LightOffTime:  time.Millisecond * 2250,
				LiftHeight:    5.5,
				LiftSpeed:     120.0,
				RetractHeight: 3.3,
				RetractSpeed:  200.0,
			},
		},
		Preview: map[uv3dp.PreviewType]image.Image{
			uv3dp.PreviewTypeTiny: image.NewRGBA(image.Rect(0, 0, 10, 10)),
			uv3dp.PreviewTypeHuge: image.NewCMYK(image.Rect(0, 0, 20, 12)),
		},
	}
)

const (
	testConfigIni = `; github.com/ezrec/uv3dp uv3dp v0.0.0 64-bits 1-01-01 00:00:00
;(****Build and Slicing Parameters****)
;(Pix per mm X            = 2.000 )
;(Pix per mm Y            = 2.000 )
;(X Resolution            = 10 )
;(Y Resolution            = 20 )
;(Layer Thickness         = 0.050 mm )
;(Layer Time              = 16500 ms )
;(Render Outlines         = False )
;(Outline Width Inset     = 2 )
;(Outline Width Outset    = 0 )
;(Bottom Layers Time      = 16500 ms )
;(Number of Bottom Layers = 2 )
;(Blanking Layer Time     = 2250 ms )
;(Build Direction         = Bottom_Up )
;(Lift Distance           = 5 mm )
;(Slide/Tilt Value        = 0 )
;(Use Mainlift GCode Tab  = False )
;(Anti Aliasing           = True )
;(Anti Aliasing Value     = 2.000 )
;(Z Lift Feed Rate        = 120.000 mm/s )
;(Z Bottom Lift Feed Rate = 120.000 mm/s )
;(Z Lift Retract Rate     = 200.000 mm/s )
;(Flip X                  = True )
;(Flip Y                  = True )
;(Number of Slices        = 4 )
(****Machine Configuration ******)
;(Platform X Size         = 5.00mm )
;(Platform Y Size         = 10.00mm )
;(Platform Z Size         = 0.20mm )
;(Max X Feedrate          = 200mm/s )
;(Max Y Feedrate          = 200mm/s )
;(Max Z Feedrate          = 200mm/s )
;(Machine Type            = UV_LCD )

G28
G21 ;Set units to be mm
G91 ;Relative Positioning
M17 ;Enable motors
<Slice> Blank
M106 S0

;<Slice> 0
M106 S255
;<Delay> 16500
M106 S0
;<Slice> Blank
G1 Z5.500 F120
G1 Z-5.450 F120
;<Delay> 6000

;<Slice> 1
M106 S255
;<Delay> 16500
M106 S0
;<Slice> Blank
G1 Z5.500 F120
G1 Z-5.450 F120
;<Delay> 6000

;<Slice> 2
M106 S255
;<Delay> 16500
M106 S0
;<Slice> Blank
G1 Z5.500 F120
G1 Z-5.450 F120
;<Delay> 6000

;<Slice> 3
M106 S255
;<Delay> 16500
M106 S0
;<Slice> Blank
G1 Z5.500 F120

M18 ;Disable Motors
M106 S0
G1 Z80
;<Completed>
`
)

func TestEncodeEmptyCWS(t *testing.T) {
	// Collect an empty printable
	time_Now = func() (now time.Time) { return }

	buffPng := &bytes.Buffer{}
	png.Encode(buffPng, image.NewGray(testProperties.Bounds()))
	png_empty := buffPng.Bytes()

	expected_zip := map[string]([]byte){
		"uv3dp.gcode":   []byte(testConfigIni),
		"uv3dp0000.png": png_empty,
		"uv3dp0001.png": png_empty,
		"uv3dp0002.png": png_empty,
		"uv3dp0003.png": png_empty,
	}

	empty := uv3dp.NewEmptyPrintable(testProperties)

	formatter := NewCWSFormatter(".cws")

	buffWriter := &bytes.Buffer{}
	formatter.Encode(buffWriter, empty)

	buffReader := &bufferReader{buffWriter.Bytes()}

	archive, _ := zip.NewReader(buffReader, buffReader.Len())

	fileMap := map[string](*zip.File){}
	for _, file := range archive.File {
		fileMap[file.Name] = file
	}

	for name, expected := range expected_zip {
		file, found := fileMap[name]
		if !found {
			t.Errorf("%v: Not found in encoded archive", name)
			continue
		}

		rc, _ := file.Open()
		defer rc.Close()
		got, _ := ioutil.ReadAll(rc)

		if !bytes.Equal(expected, got) {
			if strings.HasSuffix(name, ".gcode") {
				t.Errorf("%s: expected:\n%v\n  got:\n%v", name, string(expected), string(got))
			} else {
				t.Errorf("%s: expected %d bytes, got %d bytes", name, expected, got)
			}
		}
	}
}
