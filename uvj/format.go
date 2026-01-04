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

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

type ErrConfigMissing string

func (e ErrConfigMissing) Error() string {
	return fmt.Sprintf("config.ini: Parameter '%s' missing", string(e))
}

type ErrConfigInvalid string

func (e ErrConfigInvalid) Error() string {
	return fmt.Sprintf("config.ini: Parameter '%s' invalid", string(e))
}

type UVJLayer struct {
	Z        float32
	Exposure uv3dp.Exposure
}

type UVJConfig struct {
	Properties uv3dp.Properties
	Layers     []UVJLayer
}

type UVJ struct {
	uv3dp.Print
	Layers   []UVJLayer
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

	prop := uv3dp.Properties{
		Size:     printable.Size(),
		Exposure: printable.Exposure(),
		Bottom:   printable.Bottom(),
	}

	// If LightPWM is set to 255, don't encode it
	if prop.Exposure.LightPWM == 255 {
		prop.Exposure.LightPWM = 0
	}
	if prop.Bottom.Exposure.LightPWM == 255 {
		prop.Bottom.Exposure.LightPWM = 0
	}

	config := UVJConfig{
		Properties: prop,
		Layers:     make([]UVJLayer, prop.Size.Layers),
	}

	// Create all the layers
	uv3dp.WithEachLayer(printable, func(p uv3dp.Printable, n int) {
		filename := fmt.Sprintf("slice/%08d.png", n)

		var writer io.Writer
		writer, err = archive.Create(filename)
		if err != nil {
			return
		}

		err = png.Encode(writer, p.LayerImage(n))
		if err != nil {
			return
		}

		exposure := p.LayerExposure(n)

		// Trigger the JSON 'omitdefault' as needed
		if exposure.LightPWM == 255 {
			exposure.LightPWM = 0
		}

		config.Layers[n] = UVJLayer{
			Z:        p.LayerZ(n),
			Exposure: exposure,
		}
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
	preview := []uv3dp.PreviewType{
		uv3dp.PreviewTypeTiny,
		uv3dp.PreviewTypeHuge,
	}

	for _, code := range preview {
		image, ok := printable.Preview(code)
		if !ok {
			continue
		}

		var name string
		switch code {
		case uv3dp.PreviewTypeTiny:
			name = "tiny"
		case uv3dp.PreviewTypeHuge:
			name = "huge"
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
	data, err := io.ReadAll(cfg_reader)
	if err != nil {
		return
	}

	var config UVJConfig

	// Set some non-zero defaults
	config.Properties.Exposure.LightPWM = 255
	config.Properties.Bottom.Exposure.LightPWM = 255

	err = json.Unmarshal(data, &config)
	if err != nil {
		return
	}

	// Check layers
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
		Print:    uv3dp.Print{Properties: config.Properties},
		Layers:   config.Layers,
		layerPng: layerPng,
	}

	printable = uvj

	return
}

func (uvj *UVJ) Close() {
}

func (uvj *UVJ) LayerZ(index int) (z float32) {
	if len(uvj.Layers) == 0 {
		z = uvj.Print.LayerZ(index)
	} else {
		z = uvj.Layers[index].Z
	}

	return
}

func (uvj *UVJ) LayerExposure(index int) (exposure uv3dp.Exposure) {
	if len(uvj.Layers) == 0 {
		exposure = uvj.Print.LayerExposure(index)
	} else {
		exposure = uvj.Layers[index].Exposure
	}

	return
}

func (uvj *UVJ) LayerImage(index int) (layerImage *image.Gray) {
	pngImage, err := png.Decode(bytes.NewReader(uvj.layerPng[index]))
	if err != nil {
		err = fmt.Errorf("layer %v: %w", index, err)
		panic(err)
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

	return
}
