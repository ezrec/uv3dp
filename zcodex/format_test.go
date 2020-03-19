//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package zcodex is a set of tools for data exchange in the Prusa SL1 format
package zcodex

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io/ioutil"
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
				LightOnTime:   time.Millisecond * 80000,
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
		Metadata: map[string](interface{}){},
	}
)

const (
	testResinMetadata = `{"Guid":"62FBB25B-1E22-4B4D-A7CA-A2013F22785D","Material":"BASIC GREY","MaterialId":1,"LayerThickness":0.05,"PrintTime":16886,"LayerTime":16500,"BottomLayersTime":80000,"AdditionalSupportLayerTime":0,"BottomLayersNumber":2,"BlankingLayerTime":2250,"TotalMaterialVolumeUsed":16.21,"TotalMaterialWeightUsed":0,"TotalLayersCount":4,"DisableSettingsChanges":false,"Pauses":[],"Layers":[{"Layer":0,"UsedMaterialVolume":0},{"Layer":1,"UsedMaterialVolume":0},{"Layer":2,"UsedMaterialVolume":0},{"Layer":3,"UsedMaterialVolume":0}]}
`
)

func TestEncodeEmptyZcodex(t *testing.T) {
	// Collect an empty printable
	time_Now = func() (now time.Time) { return }

	buffPng := &bytes.Buffer{}
	png.Encode(buffPng, image.NewGray(testProperties.Bounds()))
	png_empty := buffPng.Bytes()

	expected_zip := map[string]([]byte){
		"ResinMetadata":                  []byte(testResinMetadata),
		"ResinSlicesData/Slice00000.png": png_empty,
		"ResinSlicesData/Slice00001.png": png_empty,
		"ResinSlicesData/Slice00002.png": png_empty,
		"ResinSlicesData/Slice00003.png": png_empty,
	}

	var rm ResinMetadata
	json.Unmarshal([]byte(testResinMetadata), &rm)
	testProperties.Metadata["zcodex/ResinMetadata"] = &rm

	empty := uv3dp.NewEmptyPrintable(testProperties)

	formatter := NewZcodexFormatter(".zcodex")

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
			if name == "ResinMetadata!" {
				t.Errorf("%s: expected:\n%v\n  got:\n%v", name, string(expected), string(got))
			} else {
				t.Errorf("%s: expected:\n%#v\n, got:\n%#v", name, expected, got)
			}
		}
	}
}
