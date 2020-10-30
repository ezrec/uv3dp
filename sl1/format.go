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

var (
	time_Now = time.Now
)

const (
	mmPerPixel       = 0.0472500
	defaultPixelsX   = 1440
	defaultPixelsY   = 2560
	defaultCacheSize = 16
)

var (
	defaultExposure = uv3dp.Exposure{
		LiftHeight:    4,
		LiftSpeed:     60,
		RetractHeight: 4,
		RetractSpeed:  60,
	}

	defaultBottomExposure = uv3dp.Exposure{
		LiftHeight:    4,
		LiftSpeed:     60,
		RetractHeight: 4,
		RetractSpeed:  60,
	}
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
	pixelsX      uint
	pixelsY      uint
	MillimeterY  float32
	MillimeterX  float32
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
		"expTime":        &cfg.expTime,
		"expTimeFirst":   &cfg.expTimeFirst,
		"layerHeight":    &cfg.layerHeight,
		"printTime":      &cfg.printTime,
		"usedMaterial":   &cfg.usedMaterial,
		"display_height": &cfg.MillimeterX,
		"display_width":  &cfg.MillimeterY,
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
		"numFade":          &cfg.numFade,
		"numFast":          &cfg.numFast,
		"numSlow":          &cfg.numSlow,
		"display_pixels_x": &cfg.pixelsY,
		"display_pixels_y": &cfg.pixelsX,
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

type Print struct {
	uv3dp.Print
	config   sl1Config
	layerPng []([]byte)
}

type Format struct {
	*pflag.FlagSet

	MaterialName string
}

func NewFormatter(suffix string) (sf *Format) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &Format{
		FlagSet: flagSet,
	}

	sf.StringVarP(&sf.MaterialName, "material-name", "m", "3DM-ABS @", "config.init entry 'materialName'")
	sf.SetInterspersed(false)

	return
}

func sl1Timestamp() (stamp string) {
	now := time_Now().UTC()

	stamp = fmt.Sprintf("%d-%02d-%02d at %02d:%02d:%02d UTC", now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())
	return
}

func (sf *Format) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	archive := zip.NewWriter(writer)
	defer archive.Close()

	size := printable.Size()
	exp := printable.Exposure()
	bot := printable.Bottom().Exposure
	bot_slow := printable.Bottom().Count
	bot_fade := printable.Bottom().Transition

	layerHeight := fmt.Sprintf("%.3g", size.LayerHeight)
	materialName := sf.MaterialName
	if strings.HasSuffix(materialName, " @") {
		materialName += layerHeight
	}

	config_ini := map[string]string{
		"action":                "print",
		"jobDir":                "uv3dp",
		"expTime":               fmt.Sprintf("%.3g", exp.LightOnTime),
		"expTimeFirst":          fmt.Sprintf("%.3g", bot.LightOnTime),
		"fileCreationTimestamp": sl1Timestamp(),
		"layerHeight":           layerHeight,
		"materialName":          materialName,
		"numFade":               fmt.Sprintf("%v", bot_fade),
		"numFast":               fmt.Sprintf("%v", size.Layers),
		"numSlow":               fmt.Sprintf("%v", bot_slow),
		"printProfile":          layerHeight + " Normal",
		"printTime":             fmt.Sprintf("%.3f", float32(uv3dp.PrintDuration(printable))/float32(time.Second)),
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
	for attr := range config_ini {
		attrs = append(attrs, attr)
	}
	sort.Strings(attrs)

	for _, attr := range attrs {
		fmt.Fprintf(fileConfig, "%v = %v\n", attr, config_ini[attr])
	}

	// Create all the layers
	uv3dp.WithEachLayer(printable, func(p uv3dp.Printable, n int) {
		filename := fmt.Sprintf("%s%05d.png", config_ini["jobDir"], n)

		var writer io.Writer
		writer, err = archive.Create(filename)
		if err != nil {
			return
		}

		err = png.Encode(writer, p.LayerImage(n))
		if err != nil {
			return
		}
	})

	// Save the thumbnails
	previews := []uv3dp.PreviewType{
		uv3dp.PreviewTypeTiny,
		uv3dp.PreviewTypeHuge,
	}

	for _, code := range previews {
		image, ok := printable.Preview(code)
		if !ok {
			continue
		}
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

func read_ini(file *zip.File) (map[string]string, error) {
	retmap := make(map[string]string)
	err := error(nil)

	prusacfg_reader, err := file.Open()
	if err != nil {
		return retmap, err
	}
	defer func() { prusacfg_reader.Close() }()

	scanner := bufio.NewScanner(prusacfg_reader)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, " = ", 2)
		retmap[fields[0]] = fields[1]
	}

	return retmap, err
}

func (sf *Format) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
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

	// Load the config file
	config_map, err := read_ini(cfg)
	if err != nil {
		return
	}

	prusacfg, found := fileMap["prusaslicer.ini"]
	if !found {
		err = errors.New("prusaslicer.ini not found in archive")
		return
	}

	prusacfg_map, err := read_ini(prusacfg)
	if err != nil {
		return
	}

	for key := range prusacfg_map {
		_, contained := config_map[key]
		if !contained {
			config_map[key] = prusacfg_map[key]
		}
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
	size.X = int(config.pixelsX)
	size.Y = int(config.pixelsY)
	size.Layers = int(config.numFast)

	size.Millimeter.X = config.MillimeterX
	size.Millimeter.Y = config.MillimeterY
	size.LayerHeight = config.layerHeight

	bot := &prop.Bottom
	bot.Exposure = defaultBottomExposure
	bot.Exposure.LightOnTime = config.expTimeFirst

	bot.Transition = int(config.numFade)
	bot.Count = int(config.numSlow)

	exp := &prop.Exposure
	*exp = defaultExposure
	exp.LightOnTime = config.expTime

	// Calculate layer off time based off of total print time
	bottomExposure := config.expTimeFirst * float32(config.numFade)
	restExposure := config.expTime * float32(config.numFast-config.numFade)
	totalOffTime := config.printTime - bottomExposure - restExposure
	layerOffTime := totalOffTime / float32(config.numFast)

	exp.LightOffTime = float32(layerOffTime) / 1000.0
	bot.LightOffTime = exp.LightOffTime

	exp.LightPWM = 255
	bot.LightPWM = 255

	prop.Preview = thumbImage

	sl1 := &Print{
		Print:    uv3dp.Print{Properties: prop},
		layerPng: layerPng,
	}

	printable = sl1

	return
}

func (sl1 *Print) Close() {
}

func (sl1 *Print) LayerImage(index int) (imageGray *image.Gray) {
	pngImage, err := png.Decode(bytes.NewReader(sl1.layerPng[index]))
	if err != nil {
		panic(err)
	}

	imageGray = pngImage.(*image.Gray)

	return
}
