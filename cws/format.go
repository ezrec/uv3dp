//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package cws

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
	"unicode"

	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

var (
	time_Now = time.Now
)

const (
	defaultName = "uv3dp"
)

type cwsHeader struct {
	Vendor        string
	SlicerName    string
	SlicerVersion string
	SlicerArch    string
	Timestamp     time.Time
}

func (ch *cwsHeader) String() string {
	now := ch.Timestamp
	return fmt.Sprintf("%s %s %s %s %d-%02d-%02d %02d:%02d:%02d",
		ch.Vendor,
		ch.SlicerName,
		ch.SlicerVersion,
		ch.SlicerArch,
		now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())
}

type cwsConfig struct {
	Header              cwsHeader
	Xppm                float32 `name:"Pix per mm X"`
	Yppm                float32 `name:"Pix per mm Y"`
	XResolution         int
	YResolution         int
	LayerThickness      float32 `units:"mm"`
	LayerTime           int     `units:"ms"`
	RenderOutlines      bool
	OutlineWidthInset   int
	OutlineWidthOutset  int // Nominally 0
	BottomLayersTime    int `units:"ms"`
	BottomLayers        int `name:"Number of Bottom Layers"`
	BlankingLayerTime   int `units:"ms"`
	BuildDirection      string
	LiftDistance        int  `units:"mm"`
	SlideTileValue      int  `name:"Slide/Tilt Value"`
	UseMainliftGCodeTab bool `name:"Use Mainlift GCode Tab"`
	AntiAliasing        bool
	AntiAliasingValue   float32
	ZLiftFeedRate       float32 `units:"mm/min"`
	ZBottomLiftFeedRate float32 `units:"mm/min"`
	ZLiftRetractRate    float32 `units:"mm/min"`
	FlipX               bool
	FlipY               bool
	Layers              int `name:"Number of Slices"`
}

func (cc *cwsConfig) FieldAttr(n int) (value reflect.Value, name string, units string) {
	v := reflect.ValueOf(cc).Elem()
	t := reflect.TypeOf(*cc)

	value = v.Field(n)
	field := t.Field(n)

	var ok bool
	name, ok = field.Tag.Lookup("name")
	if !ok {
		name = ""
		for n, c := range field.Name {
			if n != 0 && unicode.IsUpper(c) {
				name += " "
			}
			name += string([]rune{c})
		}
	}
	units, ok = field.Tag.Lookup("units")
	if ok {
		units = " " + units
	}

	return
}

func (cc *cwsConfig) Save(gcode io.Writer) (err error) {
	_, err = fmt.Fprintf(gcode, "; %v\n", cc.Header.String())
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(****Build and Slicing Parameters****)\n")
	if err != nil {
		return
	}

	// Start at 1, skip the header field
	t := reflect.TypeOf(*cc)

	for n := 1; n < t.NumField(); n++ {
		value, name, units := cc.FieldAttr(n)

		var repr string
		switch t.Field(n).Type.String() {
		case "int":
			repr = fmt.Sprintf("%v", value.Int())
		case "float32":
			repr = fmt.Sprintf("%1.3f", value.Float())
		case "string":
			repr = value.String()
		case "bool":
			if value.Bool() {
				repr = "True"
			} else {
				repr = "False"
			}
		default:
			panic(fmt.Sprintf("unknown type '%v' for field '%v'", t.Field(n).Type.String(), t.Field(n).Name))
		}

		_, err = fmt.Fprintf(gcode, ";(%-24v= %v%v )\n", name, repr, units)
		if err != nil {
			return
		}
	}

	_, err = fmt.Fprintf(gcode, "(****Machine Configuration ******)\n")
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %1.2fmm )\n", "Platform X Size", float32(cc.XResolution)/cc.Xppm)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %1.2fmm )\n", "Platform Y Size", float32(cc.YResolution)/cc.Yppm)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %1.2fmm )\n", "Platform Z Size", float32(cc.Layers)*cc.LayerThickness)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %vmm/min )\n", "Max X Feedrate", 200)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %vmm/min )\n", "Max Y Feedrate", 200)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %vmm/min )\n", "Max Z Feedrate", 200)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(gcode, ";(%-24v= %s )\n", "Machine Type", "UV_LCD")
	if err != nil {
		return
	}

	return
}

func (cc *cwsConfig) ValueByName(name string) (value reflect.Value, ok bool) {
	t := reflect.TypeOf(*cc)

	for n := 1; n < t.NumField(); n++ {
		fieldV, fieldN, _ := cc.FieldAttr(n)
		if fieldN == name {
			value = fieldV
			ok = true
			return
		}
	}

	return
}

func (cc *cwsConfig) Load(gcodeFile io.Reader) (err error) {
	scanner := bufio.NewScanner(gcodeFile)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 2 {
			if line[0] == ';' && strings.Contains(line, " = ") {
				fields := strings.SplitN(line[1:], " = ", 2)
				if len(fields) == 2 {
					name := strings.Trim(fields[0], " (")
					value := strings.Trim(strings.Split(fields[1], " ")[0], " )")
					ccValue, ok := cc.ValueByName(name)
					if ok {
						switch ccValue.Type().String() {
						case "int":
							var ivalue int64
							ivalue, err = strconv.ParseInt(value, 10, 64)
							if err != nil {
								return
							}
							ccValue.SetInt(ivalue)
						case "float32":
							var fvalue float64
							fvalue, err = strconv.ParseFloat(value, 64)
							if err != nil {
								return
							}
							ccValue.SetFloat(fvalue)
						case "string":
							ccValue.SetString(value)
						case "bool":
							var bvalue bool
							bvalue, err = strconv.ParseBool(value)
							ccValue.SetBool(bvalue)
						default:
							panic(fmt.Sprintf("unknown data type '%s'", ccValue.Type().String()))
						}
					}
				}
			}
		}
	}

	return
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type CWS struct {
	properties uv3dp.Properties
	config     cwsConfig
	layerPng   []([]byte)
}

type CWSFormat struct {
	*pflag.FlagSet
}

func NewCWSFormatter(suffix string) (sf *CWSFormat) {
	flagSet := pflag.NewFlagSet(suffix, pflag.ContinueOnError)

	sf = &CWSFormat{
		FlagSet: flagSet,
	}

	sf.SetInterspersed(false)

	return
}

func (sf *CWSFormat) Encode(writer uv3dp.Writer, printable uv3dp.Printable) (err error) {
	jobName := defaultName

	archive := zip.NewWriter(writer)
	defer archive.Close()

	prop := printable.Properties()

	size := &prop.Size
	exp := &prop.Exposure
	bot := &prop.Bottom.Exposure

	uv3dp.WithEachLayer(printable, func(n int, layer uv3dp.Layer) {
		filename := fmt.Sprintf("%s%04d.png", jobName, n)

		var writer io.Writer
		writer, err = archive.Create(filename)
		if err != nil {
			return
		}

		err = png.Encode(writer, layer.Image)
		if err != nil {
			return
		}
	})

	config := cwsConfig{
		Header: cwsHeader{
			Vendor:        "github.com/ezrec/uv3dp",
			SlicerName:    "uv3dp",
			SlicerVersion: "v0.0.0",
			SlicerArch:    "64-bits",
			Timestamp:     time_Now(),
		},
		Xppm:                size.Millimeter.X / float32(size.X),
		Yppm:                size.Millimeter.Y / float32(size.Y),
		XResolution:         size.X,
		YResolution:         size.Y,
		LayerThickness:      size.LayerHeight,
		LayerTime:           int(exp.LightOnTime * 1000.0),
		OutlineWidthInset:   2,
		BottomLayersTime:    int(bot.LightOnTime * 1000.0),
		BottomLayers:        prop.Bottom.Count,
		BlankingLayerTime:   int(exp.LightOffTime * 1000.0),
		BuildDirection:      "Bottom_Up",
		LiftDistance:        int(exp.LiftHeight),
		AntiAliasing:        true,
		AntiAliasingValue:   2.0,
		ZLiftFeedRate:       exp.LiftSpeed,
		ZBottomLiftFeedRate: bot.LiftSpeed,
		ZLiftRetractRate:    exp.RetractSpeed,
		FlipX:               true,
		FlipY:               true,
		Layers:              size.Layers,
	}

	// Create the gcode file
	gcode, err := archive.Create(jobName + ".gcode")
	if err != nil {
		return
	}

	// Save the config
	err = config.Save(gcode)
	if err != nil {
		return
	}

	// Emit the GCode header
	fmt.Fprintf(gcode, `
G28
G21 ;Set units to be mm
G91 ;Relative Positioning
M17 ;Enable motors
<Slice> Blank
M106 S0
`)

	// Create all the layer movement gcode
	priorZ := float32(0.0)
	for n := 0; n < size.Layers; n++ {
		layer := printable.Layer(n)
		if n > 0 {
			thickness := layer.Z - priorZ
			fmt.Fprintf(gcode, "G1 Z%1.3f F%v\n", -(layer.Exposure.LiftHeight - thickness), int(layer.Exposure.LiftSpeed))
			// This is just a guess here
			fmt.Fprintf(gcode, ";<Delay> %v\n", 720000/int(layer.Exposure.LiftSpeed))
		}

		// Create all the layers
		fmt.Fprintf(gcode, "\n;<Slice> %v\n", n)
		fmt.Fprintf(gcode, "M106 S255\n;<Delay> %v\n", int(layer.Exposure.LightOnTime*1000.0))
		fmt.Fprintf(gcode, "M106 S0\n;<Slice> Blank\n")
		fmt.Fprintf(gcode, "G1 Z%1.3f F%v\n", layer.Exposure.LiftHeight, int(layer.Exposure.LiftSpeed))
		priorZ = layer.Z
	}

	// Emit the GCode trailer
	fmt.Fprintf(gcode, "\n")
	fmt.Fprintf(gcode, "M18 ;Disable Motors\n")
	fmt.Fprintf(gcode, "M106 S0\n")
	fmt.Fprintf(gcode, "G1 Z80\n")
	fmt.Fprintf(gcode, ";<Completed>\n")

	return
}

func (sf *CWSFormat) Decode(reader uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	archive, err := zip.NewReader(reader, filesize)
	if err != nil {
		return
	}

	fileMap := make(map[string](*zip.File))

	jobName := defaultName
	for _, file := range archive.File {
		fileMap[file.Name] = file
		if strings.HasSuffix(file.Name, ".gcode") {
			jobName = file.Name[:len(file.Name)-len(".gcode")]
		}
	}

	filename := jobName + ".gcode"
	gcodeFile, found := fileMap[filename]
	if !found {
		err = errors.New(filename + " not found in archive")
		return
	}

	gcodeReader, err := gcodeFile.Open()
	if err != nil {
		return
	}
	defer func() { gcodeReader.Close() }()

	// Load the config file
	var config cwsConfig

	err = config.Load(gcodeReader)
	if err != nil {
		return
	}

	// Collect the layer files
	layerPng := make([]([]byte), config.Layers)
	for n := 0; n < cap(layerPng); n++ {
		name := fmt.Sprintf("%s%04d.png", jobName, n)
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
	size.X = config.XResolution
	size.Y = config.YResolution
	size.Layers = config.Layers

	size.Millimeter.X = float32(size.X) * config.Xppm
	size.Millimeter.Y = float32(size.Y) * config.Yppm
	size.LayerHeight = config.LayerThickness

	bot := &prop.Bottom
	bot.Exposure.LightOnTime = float32(config.BottomLayersTime) / 1000.0

	bot.Count = config.BottomLayers

	exp := &prop.Exposure
	exp.LightOnTime = float32(config.LayerTime) / 1000.0
	exp.LightOffTime = float32(config.BlankingLayerTime) / 1000.0
	exp.LiftHeight = float32(config.LiftDistance)
	exp.LiftSpeed = config.ZLiftFeedRate
	exp.RetractSpeed = config.ZLiftRetractRate

	bot.LightOffTime = exp.LightOffTime
	bot.LiftSpeed = exp.LiftSpeed
	bot.RetractSpeed = exp.RetractSpeed

	cws := &CWS{
		properties: prop,
		layerPng:   layerPng,
	}

	printable = cws

	return
}

func (cws *CWS) Close() {
}

func (cws *CWS) Properties() (prop uv3dp.Properties) {
	prop = cws.properties
	return
}

func (cws *CWS) Layer(index int) (layer uv3dp.Layer) {
	pngImage, err := png.Decode(bytes.NewReader(cws.layerPng[index]))
	if err != nil {
		panic(err)
	}

	layer.Z = float32(index) * cws.properties.Size.LayerHeight
	layer.Image = pngImage.(*image.Gray)
	layer.Exposure = cws.properties.LayerExposure(index)

	return
}
