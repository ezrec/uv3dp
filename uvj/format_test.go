//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package uvj is a set of tools for data exchange in the Prusa SL1 format
package uvj

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
			LightOnTime:   16.500,
			LightOffTime:  2.250,
			LightPWM:      255,
			LiftHeight:    5.5,
			LiftSpeed:     120.0,
			RetractHeight: 3.3,
			RetractSpeed:  200.0,
		},
		Bottom: uv3dp.Bottom{
			Count: 2,
			Style: uv3dp.BottomStyleFade,
			Exposure: uv3dp.Exposure{
				LightOnTime:   16.500,
				LightOffTime:  2.250,
				LightPWM:      255,
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
	testConfigJson = `{
  "Properties": {
    "Size": {
      "X": 10,
      "Y": 20,
      "Millimeter": {
        "X": 20,
        "Y": 40
      },
      "Layers": 4,
      "LayerHeight": 0.05
    },
    "Exposure": {
      "LightOnTime": 16.5,
      "LightOffTime": 2.25,
      "LightPWM": 255,
      "LiftHeight": 5.5,
      "LiftSpeed": 120,
      "RetractHeight": 3.3,
      "RetractSpeed": 200
    },
    "Bottom": {
      "LightOnTime": 16.5,
      "LightOffTime": 2.25,
      "LightPWM": 255,
      "LiftHeight": 5.5,
      "LiftSpeed": 120,
      "RetractHeight": 3.3,
      "RetractSpeed": 200,
      "Count": 2,
      "Style": 1
    }
  },
  "Layers": [
    {
      "Z": 0,
      "Exposure": {
        "LightOnTime": 16.5,
        "LightOffTime": 2.25,
        "LightPWM": 255,
        "LiftHeight": 5.5,
        "LiftSpeed": 120,
        "RetractHeight": 3.3,
        "RetractSpeed": 200
      }
    },
    {
      "Z": 0.05,
      "Exposure": {
        "LightOnTime": 16.5,
        "LightOffTime": 2.25,
        "LightPWM": 255,
        "LiftHeight": 5.5,
        "LiftSpeed": 120,
        "RetractHeight": 3.3,
        "RetractSpeed": 200
      }
    },
    {
      "Z": 0.1,
      "Exposure": {
        "LightOnTime": 16.5,
        "LightOffTime": 2.25,
        "LightPWM": 255,
        "LiftHeight": 5.5,
        "LiftSpeed": 120,
        "RetractHeight": 3.3,
        "RetractSpeed": 200
      }
    },
    {
      "Z": 0.15,
      "Exposure": {
        "LightOnTime": 16.5,
        "LightOffTime": 2.25,
        "LightPWM": 255,
        "LiftHeight": 5.5,
        "LiftSpeed": 120,
        "RetractHeight": 3.3,
        "RetractSpeed": 200
      }
    }
  ]
}
`
)

func TestEncodeEmptyUVJ(t *testing.T) {
	// Collect an empty printable
	time_Now = func() (now time.Time) { return }

	buffPng := &bytes.Buffer{}
	png.Encode(buffPng, image.NewGray(testProperties.Bounds()))
	png_empty := buffPng.Bytes()

	expected_zip := map[string]([]byte){
		"config.json":        []byte(testConfigJson),
		"slice/00000000.png": png_empty,
		"slice/00000001.png": png_empty,
		"slice/00000002.png": png_empty,
		"slice/00000003.png": png_empty,
	}

	empty := uv3dp.NewEmptyPrintable(testProperties)

	formatter := NewUVJFormatter(".uvj")

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
			if strings.HasSuffix(name, ".json_") {
				t.Errorf("%s: expected:\n%v\n  got:\n%v", name, string(expected), string(got))
			} else {
				t.Errorf("%s: expected %d bytes, got %d bytes", name, expected, got)
			}
		}
	}
}
