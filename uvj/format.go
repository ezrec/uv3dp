//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uvj

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"time"

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

var (
	time_Now = time.Now
)

const (
	mmPerPixel       = 0.0472500
	defaultPixelsX   = 1440
	defaultPixelsY   = 2560
	defaultCacheSize = 16
)

type ErrConfigMissing string

func (e ErrConfigMissing) Error() string {
	return fmt.Sprintf("config.ini: Parameter '%s' missing", string(e))
}

type ErrConfigInvalid string

func (e ErrConfigInvalid) Error() string {
	return fmt.Sprintf("config.ini: Parameter '%s' invalid", string(e))
}

type UVJConfig struct {
	Properties uv3dp.Properties
	Layers     []uv3dp.Layer
}

type UVJ struct {
	Config   UVJConfig
	layerPng []([]byte)
}

type UVJFormat struct {
	*pflag.FlagSet
}

func NewUVJFormatter(suffix string) (sf *UVJFormat) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &UVJFormat{
		FlagSet: flagSet,
	}

	sf.SetInterspersed(false)

	return
}

func (sf *UVJFormat) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	archive := zip.NewWriter(writer)
	defer archive.Close()

	prop := printable.Properties()

	// Don't encode the preview images into the config file
	preview := prop.Preview
	prop.Preview = nil

	config := UVJConfig{
		Properties: prop,
		Layers:     make([]uv3dp.Layer, prop.Size.Layers),
	}

	// Create all the layers
	uv3dp.WithEachLayer(printable, func(n int, layer uv3dp.Layer) {
		filename := fmt.Sprintf("slice/%08d.png", n)

		var writer io.Writer
		writer, err = archive.Create(filename)
		if err != nil {
			return
		}

		err = png.Encode(writer, layer.Image)
		if err != nil {
			return
		}

		layer.Image = nil
		config.Layers[n] = layer
	})

	// Create the config file
	fileConfig, err := archive.Create("config.json")
	if err != nil {
		return
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return
	}

	fileConfig.Write(data)
	fileConfig.Write([]byte("\n"))

	// Save the thumbnails
	for code, image := range preview {
		imageSize := image.Bounds().Size()
		var name string
		switch code {
		case uv3dp.PreviewTypeTiny:
			name = "tiny"
		case uv3dp.PreviewTypeHuge:
			name = "huge"
		default:
			name = fmt.Sprintf("%dx%d", imageSize.X, imageSize.Y)
		}

		filename := "preview/" + name + ".png"

		var writer io.Writer
		writer, err = archive.Create(filename)
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

func (sf *UVJFormat) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	archive, err := zip.NewReader(reader, filesize)
	if err != nil {
		return
	}

	fileMap := make(map[string](*zip.File))

	for _, file := range archive.File {
		fileMap[file.Name] = file
	}

	cfg, found := fileMap["config.json"]
	if !found {
		err = errors.New("config.json not found in archive")
		return
	}

	cfg_reader, err := cfg.Open()
	if err != nil {
		return
	}
	defer func() { cfg_reader.Close() }()

	// Load the config file
	data, err := ioutil.ReadAll(cfg_reader)
	if err != nil {
		return
	}

	var config UVJConfig

	err = json.Unmarshal(data, &config)
	if err != nil {
		return
	}

	if len(config.Layers) > 0 && len(config.Layers) != config.Properties.Size.Layers {
		err = fmt.Errorf("config.json: expected %v layers, found %v layers", config.Properties.Size.Layers, config.Layers)
		return
	}

	// Collect the layer files
	layerPng := make([]([]byte), config.Properties.Size.Layers)
	for n := 0; n < cap(layerPng); n++ {
		name := fmt.Sprintf("slice/%08d.png", n)
		file, ok := fileMap[name]
		if !ok {
			err = errors.New(fmt.Sprintf("%s: Missing from archive", name))
			return
		}
		var reader io.ReadCloser
		reader, err = file.Open()
		if err != nil {
			return
		}
		defer reader.Close()

		layerPng[n], err = ioutil.ReadAll(reader)
		if err != nil {
			return
		}
	}

	// Collect the thumbnails
	thumbs := map[uv3dp.PreviewType]string{
		uv3dp.PreviewTypeTiny: "preview/tiny.png",
		uv3dp.PreviewTypeHuge: "preview/huge.png",
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
			err = fmt.Errorf("%s: %w", file.Name, err)
			return
		}
		thumbImage[pt] = thumb
	}

	config.Properties.Preview = thumbImage

	uvj := &UVJ{
		Config:   config,
		layerPng: layerPng,
	}

	printable = uvj

	return
}

func (uvj *UVJ) Close() {
}

func (uvj *UVJ) Properties() (prop uv3dp.Properties) {
	prop = uvj.Config.Properties
	return
}

func (uvj *UVJ) Layer(index int) (layer uv3dp.Layer) {
	pngImage, err := png.Decode(bytes.NewReader(uvj.layerPng[index]))
	if err != nil {
		err = fmt.Errorf("Layer %v: %w", index, err)
		panic(err)
	}

	if len(uvj.Config.Layers) == 0 {
		layer = uv3dp.Layer{
			Z:        float32(index) * uvj.Config.Properties.Size.LayerHeight,
			Exposure: uvj.Config.Properties.LayerExposure(index),
		}
	} else {
		layer = uvj.Config.Layers[index]
	}
	layerImage, ok := pngImage.(*image.Gray)
	if !ok {
		layerImage = image.NewGray(pngImage.Bounds())
		for y := pngImage.Bounds().Min.Y; y < pngImage.Bounds().Max.Y; y++ {
			for x := pngImage.Bounds().Min.X; x < pngImage.Bounds().Max.X; x++ {
				layerImage.Set(x, y, pngImage.At(x, y))
			}
		}
	}

	layer.Image = layerImage

	return
}
