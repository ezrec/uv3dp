//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package zcodex

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

const (
	mmPerPixel       = 0.05
	defaultPixelsX   = 1440
	defaultPixelsY   = 2560
	defaultCacheSize = 16
)

type UserSettingsData struct {
	MaxLayer                         int
	PrintTime                        string
	MaterialVolume                   float32
	IsAdvanced                       int
	Printer                          string
	MaterialType                     string
	MaterialId                       string
	LayerThickness                   string
	RaftEnabled                      int
	RaftHeight                       float32
	RaftOffset                       float32
	ModelLiftEnabled                 int
	ModelLiftHeight                  float32
	CrossSupportEnabled              int
	LayerExposureTime                int
	LayerThicknessesDisplayTime      []string
	ExposureOffTime                  int
	BottomLayerExposureTime          int
	BottomLayersCount                int
	SupportAdditionalExposureEnabled int
	SupportAdditionalExposureTime    int
	ZLiftDistance                    float32
	ZLiftRetractRate                 float32
	ZLiftFeedRate                    float32
	AntiAliasing                     int
	XCorrection                      int
	YCorrection                      int
	HollowEnabled                    int
	HollowThickness                  float32
	InfillDensity                    float32
}

type ResinMetadataLayer struct {
	Layer              int
	UsedMaterialVolume float32
}

type ResinMetadata struct {
	Guid                       string
	Material                   string
	MaterialId                 int
	LayerThickness             float32
	PrintTime                  int
	LayerTime                  int
	BottomLayersTime           int
	AdditionalSupportLayerTime int
	BottomLayersNumber         int
	BlankingLayerTime          int
	TotalMaterialVolumeUsed    float32
	TotalMaterialWeightUsed    float32
	TotalLayersCount           int
	DisableSettingsChanges     bool
	Pauses                     []int
	Layers                     []ResinMetadataLayer
}

type Zcodex struct {
	uv3dp.Print
	layerPng []([]byte)
}

type ZcodexFormat struct {
	*pflag.FlagSet
}

func NewZcodexFormatter(suffix string) (sf *ZcodexFormat) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &ZcodexFormat{
		FlagSet: flagSet,
	}

	sf.SetInterspersed(false)

	return
}

func (sf *ZcodexFormat) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	archive := zip.NewWriter(writer)
	defer archive.Close()

	var rm ResinMetadata
	anon, ok := printable.Metadata("zcodex/ResinMetadata")
	if ok {
		rmptr, ok := anon.(*ResinMetadata)
		if ok {
			rm = *rmptr
		}
	}

	size := printable.Size()
	exposure := printable.Exposure()
	bottom := printable.Bottom()

	rm.LayerThickness = size.LayerHeight
	rm.LayerTime = int(exposure.LightOnTime * 1000.0)
	rm.BottomLayersTime = int(bottom.Exposure.LightOnTime * 1000.0)
	rm.TotalLayersCount = size.Layers
	rm.BottomLayersNumber = bottom.Count
	rm.BlankingLayerTime = int(exposure.LightOffTime * 1000.0)

	var us UserSettingsData
	anon, ok = printable.Metadata("zcodex/UserSettingsData")
	if ok {
		usptr, ok := anon.(*UserSettingsData)
		if ok {
			us = *usptr
		}
	}

	us.MaxLayer = size.Layers
	us.LayerThickness = fmt.Sprintf("%.2g mm", size.LayerHeight)
	us.LayerExposureTime = int(exposure.LightOnTime * 1000.0)
	us.ExposureOffTime = int(exposure.LightOffTime * 1000.0)
	us.BottomLayerExposureTime = int(bottom.Exposure.LightOnTime * 1000.0)
	us.BottomLayersCount = bottom.Count
	us.ZLiftDistance = exposure.LiftHeight
	us.ZLiftRetractRate = exposure.RetractSpeed
	us.ZLiftFeedRate = exposure.LiftSpeed

	rm.Layers = make([]ResinMetadataLayer, size.Layers)

	// Create all the layers
	for n := 0; n < size.Layers; n++ {
		filename := fmt.Sprintf("ResinSlicesData/Slice%05d.png", n)

		writer, err = archive.Create(filename)
		if err != nil {
			return
		}

		err = png.Encode(writer, printable.LayerImage(n))
		if err != nil {
			return
		}

		rm.Layers[n] = ResinMetadataLayer{Layer: n, UsedMaterialVolume: 0.0}
	}

	// Save the UserSettingsData
	writer, err = archive.Create("UserSettingsData")
	if err != nil {
		return
	}

	err = json.NewEncoder(writer).Encode(&us)
	if err != nil {
		return
	}

	// Save the ResinMetadata
	writer, err = archive.Create("ResinMetadata")
	if err != nil {
		return
	}

	err = json.NewEncoder(writer).Encode(&rm)
	if err != nil {
		return
	}

	// Save the thumbnails
	image, ok := printable.Preview(uv3dp.PreviewTypeTiny)
	if !ok {
		image, ok = printable.Preview(uv3dp.PreviewTypeHuge)
	}
	if ok {
		writer, err = archive.Create("Preview.png")
		if err != nil {
			return
		}

		err = png.Encode(writer, image)
		if err != nil {
			return
		}
	}

	return
}

func loadJSON(filemap map[string](*zip.File), filename string, msg interface{}) (err error) {
	zfile, found := filemap[filename]
	if !found {
		err = fmt.Errorf("%v not found in archive", filename)
		return
	}

	reader, err := zfile.Open()
	if err != nil {
		return
	}
	defer reader.Close()

	err = json.NewDecoder(reader).Decode(msg)

	return
}

func (sf *ZcodexFormat) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	archive, err := zip.NewReader(reader, filesize)
	if err != nil {
		return
	}

	fileMap := make(map[string](*zip.File))

	for _, file := range archive.File {
		fileMap[file.Name] = file
	}

	gcode_file, ok := fileMap["ResinGCodeData"]
	if !ok {
		err = fmt.Errorf("ResinGCodeData not in archive")
		return
	}

	gcode_reader, err := gcode_file.Open()
	if err != nil {
		return
	}
	defer gcode_reader.Close()

	sliceMap := []int{}

	gcode_scanner := bufio.NewScanner(gcode_reader)
	var slice int
	for gcode_scanner.Scan() {
		text := gcode_scanner.Text()
		switch {
		case text == "<Delay_model>":
			sliceMap = append(sliceMap, slice)
		case strings.HasPrefix(text, "<Slice> "):
			var rest string
			var n int
			n, err = fmt.Sscanf(text[8:], "%d%s", &slice, &rest)
			if n != 1 || err != io.EOF {
				err = fmt.Errorf("ResinGCodeData: Invalid slice load: '%v'", text)
				return
			}
		}
	}

	var us UserSettingsData
	err = loadJSON(fileMap, "UserSettingsData", &us)
	if err != nil {
		return
	}

	var rm ResinMetadata
	err = loadJSON(fileMap, "ResinMetadata", &rm)
	if err != nil {
		return
	}

	// Collect the layer files
	layerPng := make([]([]byte), len(rm.Layers))
	for n := 0; n < cap(layerPng); n++ {
		name := fmt.Sprintf("ResinSlicesData/Slice%05d.png", sliceMap[n])
		file, ok := fileMap[name]
		if !ok {
			err = fmt.Errorf("%s: Missing from archive", name)
			return
		}
		var reader io.ReadCloser
		reader, err = file.Open()
		if err != nil {
			return
		}
		defer reader.Close()

		layerPng[n], err = io.ReadAll(reader)
		if err != nil {
			return
		}
	}

	// Collect the thumbnails
	thumbs := map[uv3dp.PreviewType]string{
		uv3dp.PreviewTypeTiny: "Preview.png",
		uv3dp.PreviewTypeHuge: "Preview.png", // There's only one preview in the archive...
	}

	thumbImage := make(map[uv3dp.PreviewType]image.Image)
	for pt, pn := range thumbs {
		file, ok := fileMap[pn]
		if !ok {
			continue
		}

		var reader io.ReadCloser
		reader, err = file.Open()
		if err != nil {
			return
		}
		defer func() { reader.Close() }()

		var thumb image.Image
		thumb, err = png.Decode(reader)
		if err != nil {
			return
		}
		thumbImage[pt] = thumb
	}

	prop := uv3dp.Properties{}

	size := &prop.Size
	size.X = defaultPixelsX
	size.Y = defaultPixelsY
	size.Layers = len(rm.Layers)

	size.Millimeter.X = float32(size.X) * mmPerPixel
	size.Millimeter.Y = float32(size.Y) * mmPerPixel
	size.LayerHeight = rm.LayerThickness

	exp := &prop.Exposure
	exp.LightOnTime = float32(rm.LayerTime) / 1000.0
	exp.LightOffTime = float32(rm.BlankingLayerTime) / 1000.0
	exp.LightPWM = 255

	exp.LiftHeight = us.ZLiftDistance
	exp.RetractSpeed = us.ZLiftRetractRate
	exp.LiftSpeed = us.ZLiftFeedRate

	bot := &prop.Bottom
	bot.Exposure = *exp

	bot.Exposure.LightOnTime = float32(rm.BottomLayersTime) / 1000.0
	bot.Exposure.LightPWM = 255

	bot.Count = rm.BottomLayersNumber

	prop.Preview = thumbImage

	prop.Metadata = make(map[string](interface{}))
	prop.Metadata["zcodex/UserSettingsData"] = &us
	prop.Metadata["zcodex/ResinMetadata"] = &rm

	zcodex := &Zcodex{
		Print:    uv3dp.Print{Properties: prop},
		layerPng: layerPng,
	}

	printable = zcodex

	return
}

func (zcodex *Zcodex) Close() {
}

func asGray(in image.Image) (out *image.Gray) {
	bounds := in.Bounds()
	out = image.NewGray(bounds)

	for y := 0; y < bounds.Size().Y; y++ {
		for x := 0; x < bounds.Size().X; x++ {
			out.Set(x, y, color.GrayModel.Convert(in.At(x, y)))
		}
	}

	return
}

func (zcodex *Zcodex) LayerImage(index int) (grayImage *image.Gray) {
	pngImage, err := png.Decode(bytes.NewReader(zcodex.layerPng[index]))
	if err != nil {
		panic(err)
	}

	grayImage, ok := pngImage.(*image.Gray)
	if !ok {
		grayImage = asGray(pngImage)
	}

	return
}
