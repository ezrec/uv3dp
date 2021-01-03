//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package czip

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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

var (
	time_Now = time.Now
)

type czipConfig struct {
	FileName                string
	MachineType             string
	EstimatedPrintTime      float32
	Volume                  float32
	Resin                   string
	Weight                  float32
	Price                   float32
	LayerHeight             float32
	ResolutionX             int
	ResolutionY             int
	MachineX                float32
	MachineY                float32
	MachineZ                float32
	ProjectType             string
	NormalExposureTime      float32
	BottomLayExposureTime   float32
	BottomLayerExposureTime float32
	NormalDropSpeed         float32
	NormalLayerLiftHeight   float32
	ZSlowUpDistance         float32
	NormalLayerLiftSpeed    float32
	BottomLayCount          int
	BottomLayerCount        int
	Mirror                  int
	TotalLayer              int
	BottomLayerLiftHeight   float32
	BottomLayerLiftSpeed    float32
	BottomLightOffTime      float32
	LightOffTime            float32
}

type ErrConfigMissing string

func (e ErrConfigMissing) Error() string {
	return fmt.Sprintf("run.gcode: Parameter '%s' missing", string(e))
}

type ErrConfigInvalid string

func (e ErrConfigInvalid) Error() string {
	return fmt.Sprintf("run.gcode: Parameter '%s' invalid", string(e))
}

func (cfg *czipConfig) Marshal() (out string) {
	t := reflect.TypeOf(cfg).Elem()
	s := reflect.ValueOf(cfg).Elem()

	for n := 0; n < t.NumField(); n++ {
		sf := t.Field(n)
		name := strings.ToLower(sf.Name[:1]) + sf.Name[1:]
		line := fmt.Sprintf(";%v:%v\n", name, s.Field(n).Interface())
		out += line
	}

	return
}

func (cfg *czipConfig) Unmarshal(in string) (ok bool) {
	line := in

	if len(line) < 4 || line[0] != ';' {
		return false
	}
	line = line[1:]
	var attr string
	i := 0
	for ; i < len(line); i++ {
		if line[i] == ':' {
			attr = line[:i]
			line = line[i+1:]
			i = 0
		} else if line[i] == ' ' || line[i] == '\t' || line[i] == '\r' || line[i] == '\n' || line[i] == '#' {
			break
		}
	}
	val := line[:i]

	fmt.Printf(": '%s' => '%s' '%s'\n", in, attr, val)
	if attr == "" || val == "" {
		return false
	}

	name := strings.ToUpper(attr[:1]) + attr[1:]

	t := reflect.TypeOf(cfg).Elem()
	s := reflect.ValueOf(cfg).Elem()

	sf, ok := t.FieldByName(name)
	if !ok {
		return false
	}

	vf := s.FieldByIndex(sf.Index)

	switch sf.Type.String() {
	case "string":
		vf.SetString(val)
	case "int":
		ival, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return false
		}
		vf.SetInt(ival)
	case "float32":
		fval, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return false
		}
		vf.SetFloat(fval)
	default:
		panic(name + ": unknown type " + sf.Type.String())
	}

	return true
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Print struct {
	uv3dp.Print
	config   czipConfig
	layerPng []([]byte)
}

type Format struct {
	*pflag.FlagSet
}

func NewFormatter(suffix string) (sf *Format) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &Format{
		FlagSet: flagSet,
	}

	sf.SetInterspersed(false)

	return
}

func (sf *Format) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	archive := zip.NewWriter(writer)
	defer archive.Close()

	size := printable.Size()
	exp := printable.Exposure()
	bot := printable.Bottom().Exposure
	bot_count := printable.Bottom().Count

	cfg := czipConfig{
		MachineType:             "default",
		EstimatedPrintTime:      float32(uv3dp.PrintDuration(printable) / time.Second),
		Volume:                  0.0,
		Resin:                   "default",
		Weight:                  0.0,
		Price:                   0.0,
		LayerHeight:             size.LayerHeight,
		ResolutionX:             size.X,
		ResolutionY:             size.Y,
		MachineX:                size.Millimeter.X,
		MachineY:                size.Millimeter.Y,
		MachineZ:                printable.LayerZ(size.Layers - 1),
		ProjectType:             "mirror_LCD",
		NormalExposureTime:      exp.LightOnTime,
		LightOffTime:            exp.LightOffTime,
		BottomLightOffTime:      bot.LightOffTime,
		BottomLayExposureTime:   bot.LightOnTime,
		BottomLayerExposureTime: bot.LightOnTime,
		NormalDropSpeed:         exp.RetractSpeed,
		NormalLayerLiftHeight:   exp.LiftHeight,
		ZSlowUpDistance:         exp.RetractHeight,
		NormalLayerLiftSpeed:    exp.LiftSpeed,
		BottomLayCount:          bot_count,
		BottomLayerCount:        bot_count,
		Mirror:                  1,
		TotalLayer:              size.Layers,
		BottomLayerLiftHeight:   bot.LiftHeight,
		BottomLayerLiftSpeed:    bot.LiftSpeed,
	}

	// Create all the layers
	uv3dp.WithEachLayer(printable, func(p uv3dp.Printable, n int) {
		filename := fmt.Sprintf("%d.png", n+1)

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

	gcode := cfg.Marshal()
	gcode += `;START_GCODE_BEGIN
G21;
G90;
M106 S0;
G28 Z0;

;START_GCODE_END
`

	for n := 0; n < size.Layers; n++ {
		filename := fmt.Sprintf("%d.png", n+1)

		z := printable.LayerZ(n)
		exp := printable.LayerExposure(n)

		layer_code := fmt.Sprintf(`
;LAYER_START:%d
;currPos:%.2f
M6054 "%s";show Image
G0 Z%.2f F%d;
G0 Z%.2f F%d;
G4 P%d;
M106 S%d;light on
G4 P%d;
M106 S0; light off

;LAYER_END
`, n, z, filename, z+exp.LiftHeight, int(exp.LiftSpeed), z, int(exp.RetractSpeed),
			int(exp.LightOnTime*1000),
			exp.LightPWM,
			int(exp.LightOffTime*1000))

		gcode += layer_code
	}

	gcode += fmt.Sprintf(`
;END_GCODE_BEGIN
M106 S0;
G1 Z%.2f F25
M18;

;END_GCODE_END
`, cfg.MachineZ+cfg.NormalLayerLiftHeight)

	// Create the gcode file
	fileConfig, err := archive.Create("run.gcode")
	if err != nil {
		return
	}

	_, err = fileConfig.Write([]byte(gcode))
	if err != nil {
		return
	}

	// Save the thumbnails
	previews := map[uv3dp.PreviewType]string{
		uv3dp.PreviewTypeTiny: "preview_cropping.png",
		uv3dp.PreviewTypeHuge: "preview.png",
	}

	for code, filename := range previews {
		image, ok := printable.Preview(code)
		if !ok {
			continue
		}

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

func (sf *Format) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	archive, err := zip.NewReader(reader, filesize)
	if err != nil {
		return
	}

	fileMap := make(map[string](*zip.File))

	for _, file := range archive.File {
		fileMap[file.Name] = file
	}

	run, found := fileMap["run.gcode"]
	if !found {
		err = errors.New("run.gcode not found in archive")
		return
	}

	run_reader, err := run.Open()
	if err != nil {
		return
	}
	defer func() { run_reader.Close() }()

	// Load the gcode file
	header := czipConfig{}
	scanner := bufio.NewScanner(run_reader)
	for scanner.Scan() {
		line := scanner.Text()
		ok := header.Unmarshal(line)
		if !ok {
			break
		}
	}

	// Collect the layer files
	layerPng := make([]([]byte), header.TotalLayer)
	for n := 0; n < cap(layerPng); n++ {
		name := fmt.Sprintf("%d.png", n+1)
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
		uv3dp.PreviewTypeTiny: "preview_cropping.png",
		uv3dp.PreviewTypeHuge: "preview.png",
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
	size.X = header.ResolutionX
	size.Y = header.ResolutionY
	size.Layers = header.TotalLayer

	size.Millimeter.X = header.MachineX
	size.Millimeter.Y = header.MachineY
	size.LayerHeight = header.LayerHeight

	exp := &prop.Exposure
	exp.LightOnTime = header.NormalExposureTime
	exp.LightOffTime = header.LightOffTime
	exp.LiftSpeed = header.NormalLayerLiftSpeed
	exp.LiftHeight = header.NormalLayerLiftHeight
	exp.RetractSpeed = header.NormalDropSpeed
	exp.RetractHeight = header.ZSlowUpDistance
	exp.LightPWM = 255

	bot := &prop.Bottom
	bot.Count = header.BottomLayerCount
	bot.Exposure.LightOnTime = header.BottomLayerExposureTime
	bot.Exposure.LightOffTime = header.BottomLightOffTime
	bot.Exposure.LiftSpeed = header.BottomLayerLiftSpeed
	bot.Exposure.LiftHeight = header.BottomLayerLiftHeight
	bot.Exposure.RetractSpeed = header.NormalDropSpeed
	bot.Exposure.RetractHeight = header.ZSlowUpDistance
	bot.Exposure.LightPWM = 255

	prop.Preview = thumbImage

	czip := &Print{
		Print:    uv3dp.Print{Properties: prop},
		layerPng: layerPng,
	}

	printable = czip

	return
}

func (czip *Print) Close() {
}

func (czip *Print) LayerImage(index int) (imageGray *image.Gray) {
	pngImage, err := png.Decode(bytes.NewReader(czip.layerPng[index]))
	if err != nil {
		panic(err)
	}

	imageGray = pngImage.(*image.Gray)

	return
}
