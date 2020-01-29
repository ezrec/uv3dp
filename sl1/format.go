//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package sl1

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

const (
	mmPerPixel       = 0.0472500
	defaultPixelsX   = 1440
	defaultPixelsY   = 2560
	defaultCacheSize = 16
)

type sl1Config struct {
	jobDir       string
	expTime      float32
	expTimeFirst float32
	layerHeight  float32
	numFade      uint
	numFast      uint
	numSlow      uint
	printTime    float32
	usedMaterial float32
}

type ErrConfigMissing string

func (e ErrConfigMissing) Error() string {
	return fmt.Sprintf("config.ini: Parameter '%s' missing", string(e))
}

type ErrConfigInvalid string

func (e ErrConfigInvalid) Error() string {
	return fmt.Sprintf("config.ini: Parameter '%s' invalid", string(e))
}

func (cfg *sl1Config) unmap(items map[string]string) (err error) {
	jobDir, ok := items["jobDir"]
	if !ok {
		err = ErrConfigMissing("jobDir")
		return
	}
	cfg.jobDir = jobDir

	floats := map[string](*float32){
		"expTime":      &cfg.expTime,
		"expTimeFirst": &cfg.expTimeFirst,
		"layerHeight":  &cfg.layerHeight,
		"printTime":    &cfg.printTime,
		"usedMaterial": &cfg.usedMaterial,
	}
	for attr, ptr := range floats {
		item, ok := items[attr]
		if !ok {
			err = ErrConfigMissing(attr)
		}
		var val float64
		val, err = strconv.ParseFloat(item, 32)
		if err != nil {
			return ErrConfigInvalid(attr)
		}
		*ptr = float32(val)
	}

	uints := map[string](*uint){
		"numFade": &cfg.numFade,
		"numFast": &cfg.numFast,
		"numSlow": &cfg.numSlow,
	}
	for attr, ptr := range uints {
		item, ok := items[attr]
		if !ok {
			err = ErrConfigMissing(attr)
		}
		var val uint64
		val, err = strconv.ParseUint(item, 10, 32)
		if err != nil {
			return ErrConfigInvalid(attr)
		}
		*ptr = uint(val)
	}

	return
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Sl1 struct {
	config     sl1Config
	properties uv3dp.Properties
	layerPng   []([]byte)
}

type Sl1Format struct {
	*pflag.FlagSet

	MaterialName string
}

func NewSl1Formatter(suffix string) (sf *Sl1Format) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &Sl1Format{
		FlagSet: flagSet,
	}

	sf.StringVarP(&sf.MaterialName, "material-name", "m", "3DM-ABS @", "config.init entry 'materialName'")
	sf.SetInterspersed(false)

	return
}

func sl1Timestamp() (stamp string) {
	now := time.Now().UTC()

	stamp = fmt.Sprintf("%d-%02d-%02d at %02d:%02d:%02d UTC", now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())
	return
}

func (sf *Sl1Format) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	archive := zip.NewWriter(writer)
	defer archive.Close()

	prop := printable.Properties()

	size := &prop.Size
	exp := &prop.Exposure
	bot := &prop.Bottom.Exposure

	layerHeight := fmt.Sprintf("%.3g", size.LayerHeight)
	materialName := sf.MaterialName
	if strings.HasSuffix(materialName, " @") {
		materialName += layerHeight
	}

	numFade := 0
	numSlow := prop.Bottom.Count
	if prop.Bottom.Style == uv3dp.BottomStyleFade {
		numFade = prop.Bottom.Count
		numSlow = 0
	}

	config_ini := map[string]string{
		"action":                "print",
		"jobDir":                "uv3dp",
		"expTime":               fmt.Sprintf("%.3g", float64(exp.LightExposure)/float64(time.Second)),
		"expTimeFirst":          fmt.Sprintf("%.3g", float64(bot.LightExposure)/float64(time.Second)),
		"fileCreationTimestamp": sl1Timestamp(),
		"layerHeight":           layerHeight,
		"materialName":          materialName,
		"numFade":               fmt.Sprintf("%v", numFade),
		"numFast":               fmt.Sprintf("%v", size.Layers),
		"numSlow":               fmt.Sprintf("%v", numSlow),
		"printProfile":          layerHeight + " Normal",
		"printTime":             fmt.Sprintf("%.3f", float64(prop.Duration())/float64(time.Second)),
		"printerModel":          "SL1",
		"printerProfile":        "Original Prusa SL1",
		"prusaSlicerVersion":    "uv3dp",
		"usedMaterial":          "0.0", // TODO: Calculate this properly!
	}

	// Create the config file
	fileConfig, err := archive.Create("config.ini")
	if err != nil {
		return
	}

	attrs := []string{}
	for attr, _ := range config_ini {
		attrs = append(attrs, attr)
	}
	sort.Strings(attrs)

	for _, attr := range attrs {
		fmt.Fprintf(fileConfig, "%v = %v\n", attr, config_ini[attr])
	}

	// Create all the layers
	for n := 0; n < size.Layers; n++ {
		layer := printable.Layer(n)

		filename := fmt.Sprintf("%s%05d.png", config_ini["jobDir"], n)

		var writer io.Writer
		writer, err = archive.Create(filename)
		if err != nil {
			return
		}

		err = png.Encode(writer, layer.Image)
		if err != nil {
			return
		}
	}

	// Save the thumbnails
	for _, image := range prop.Preview {
		imageSize := image.Bounds().Size()
		filename := fmt.Sprintf("thumbnail/thumbnail%dx%d.png", imageSize.X, imageSize.Y)

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

func (sf *Sl1Format) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	archive, err := zip.NewReader(reader, filesize)
	if err != nil {
		return
	}

	fileMap := make(map[string](*zip.File))

	for _, file := range archive.File {
		fileMap[file.Name] = file
	}

	cfg, found := fileMap["config.ini"]
	if !found {
		err = errors.New("config.ini not found in archive")
		return
	}

	cfg_reader, err := cfg.Open()
	if err != nil {
		return
	}
	defer func() { cfg_reader.Close() }()

	// Load the config file
	config_map := make(map[string]string)
	scanner := bufio.NewScanner(cfg_reader)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, " = ", 2)
		config_map[fields[0]] = fields[1]
	}

	var config sl1Config
	err = config.unmap(config_map)
	if err != nil {
		return
	}

	// Collect the layer files
	layerPng := make([]([]byte), config.numFast)
	for n := 0; n < cap(layerPng); n++ {
		name := fmt.Sprintf("%s%05d.png", config.jobDir, n)
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
		uv3dp.PreviewTypeTiny: "thumbnail/thumbnail400x400.png",
		uv3dp.PreviewTypeHuge: "thumbnail/thumbnail800x480.png",
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
	size.Layers = int(config.numFast)

	size.Millimeter.X = float32(size.X) * mmPerPixel
	size.Millimeter.Y = float32(size.Y) * mmPerPixel
	size.LayerHeight = config.layerHeight

	bot := &prop.Bottom
	bot.Exposure.LightExposure = time.Duration(config.expTimeFirst*1000) * time.Millisecond

	if config.numFade > 0 {
		bot.Count = int(config.numFade)
		bot.Style = uv3dp.BottomStyleFade
	} else {
		bot.Count = int(config.numSlow)
		bot.Style = uv3dp.BottomStyleSlow
	}

	exp := &prop.Exposure
	exp.LightExposure = time.Duration(config.expTime*1000) * time.Millisecond

	// Calculate layer off time based off of total print time
	bottomExposure := config.expTimeFirst * float32(config.numFade)
	restExposure := config.expTime * float32(config.numFast-config.numFade)
	totalOffTime := config.printTime - bottomExposure - restExposure
	layerOffTime := totalOffTime / float32(config.numFast)

	exp.LightOffTime = time.Duration(layerOffTime*1000) * time.Millisecond
	bot.LightOffTime = exp.LightOffTime

	prop.Preview = thumbImage

	sl1 := &Sl1{
		properties: prop,
		layerPng:   layerPng,
	}

	printable = sl1

	return
}

func (sl1 *Sl1) Close() {
}

func (sl1 *Sl1) Properties() (prop uv3dp.Properties) {
	prop = sl1.properties
	return
}

func (sl1 *Sl1) Layer(index int) (layer uv3dp.Layer) {
	pngImage, err := png.Decode(bytes.NewReader(sl1.layerPng[index]))
	if err != nil {
		panic(err)
	}

	layer.Z = float32(index) * sl1.properties.Size.LayerHeight
	layer.Image = pngImage.(*image.Gray)
	exposure := sl1.properties.LayerExposure(index)
	layer.Exposure = &exposure

	return
}
