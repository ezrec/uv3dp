//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package sl1 is a set of tools for data exchange in the Prusa SL1 format
package sl1

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
			LightExposure: time.Millisecond * 16500,
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
				LightExposure: time.Millisecond * 16500,
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
	testConfigIni = `action = print
expTime = 16.5
expTimeFirst = 16.5
fileCreationTimestamp = 1-01-01 at 00:00:00 UTC
jobDir = uv3dp
layerHeight = 0.05
materialName = 3DM-ABS @0.05
numFade = 2
numFast = 4
numSlow = 0
printProfile = 0.05 Normal
printTime = 75.425
printerModel = SL1
printerProfile = Original Prusa SL1
prusaSlicerVersion = uv3dp
usedMaterial = 0.0
`
)

func TestEncodeEmptySl1(t *testing.T) {
	// Collect an empty printable
	time_Now = func() (now time.Time) { return }

	buffPng := &bytes.Buffer{}
	png.Encode(buffPng, image.NewGray(testProperties.Bounds()))
	png_empty := buffPng.Bytes()

	expected_zip := map[string]([]byte){
		"config.ini":     []byte(testConfigIni),
		"uv3dp00000.png": png_empty,
		"uv3dp00001.png": png_empty,
		"uv3dp00002.png": png_empty,
		"uv3dp00003.png": png_empty,
	}

	empty := uv3dp.NewEmptyPrintable(testProperties)

	formatter := NewSl1Formatter(".sl1")

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
			if strings.HasSuffix(name, ".ini") {
				t.Errorf("%s: expected:\n%v\n  got:\n%v", name, string(expected), string(got))
			} else {
				t.Errorf("%s: expected %d bytes, got %d bytes", name, expected, got)
			}
		}
	}
}
