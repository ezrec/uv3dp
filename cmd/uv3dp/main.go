//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ezrec/uv3dp"
	"github.com/ezrec/uv3dp/cbddlp"
	"github.com/ezrec/uv3dp/sl1"

	"github.com/spf13/pflag"
)

const (
	defaultCachedLayers = 64
)

var param struct {
	decimate bool
	input    string
	output   string
}

func init() {
	pflag.BoolVarP(&param.decimate, "decimate", "D", false, "Decimate layers of the file")
	pflag.StringVarP(&param.input, "input", "i", "", "Input file")
	pflag.StringVarP(&param.output, "output", "o", "", "Output file")
}

func decoderBySuffix(file string) (decoder uv3dp.PrintableDecoder, err error) {
	switch {
	case strings.HasSuffix(file, ".cbddlp") || strings.HasSuffix(file, ".photon"):
		decoder = cbddlp.Decoder
	case strings.HasSuffix(file, ".sl1"):
		decoder = sl1.Decoder
	default:
		err = errors.New(fmt.Sprintf("File '%s' not a recognized format", file))
		return
	}

	return
}

func encoderBySuffix(file string) (encoder uv3dp.PrintableEncoder, err error) {
	switch {
	case strings.HasSuffix(file, ".cbddlp") || strings.HasSuffix(file, ".photon"):
		encoder = cbddlp.Encoder
	case strings.HasSuffix(file, ".sl1"):
		encoder = sl1.Encoder
	default:
		err = errors.New(fmt.Sprintf("File '%s' not a recognized format", file))
		return
	}

	return
}

func evaluate() (err error) {
	if len(param.input) == 0 {
		err = errors.New("-input: Required parameter missing")
		return
	}

	decoder, err := decoderBySuffix(param.input)
	if err != nil {
		return
	}

	var reader *os.File
	reader, err = os.Open(param.input)
	if err != nil {
		return
	}
	defer func() { reader.Close() }()

	filesize, err := reader.Seek(0, io.SeekEnd)
	if err != nil {
		return
	}

	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		return
	}

	input, err := decoder(reader, filesize)
	if err != nil {
		return
	}

	input = uv3dp.NewCachedPrintable(input, defaultCachedLayers)

	prop := input.Properties()
	size := &prop.Size
	fmt.Printf("Layers: %v, %vx%v slices, %.2f x %.2f x %.2f mm bed required\n",
		size.Layers, size.X, size.Y,
		size.Millimeter.X, size.Millimeter.Y, float32(size.Layers)*size.LayerHeight)

	exp := &prop.Exposure
	bot := &prop.Bottom
	fmt.Printf("Exposure: %v on, %v off nominal, %v bottom (%v layers)\n",
		exp.LightExposure, exp.LightOffTime,
		bot.Exposure.LightExposure, bot.Count)

	if param.decimate {
		input = uv3dp.NewDecimatedPrintable(input)
	}

	if len(param.output) > 0 {
		var encoder uv3dp.PrintableEncoder
		encoder, err = encoderBySuffix(param.output)
		if err != nil {
			return
		}

		var writer *os.File
		writer, err = os.Create(param.output)
		if err != nil {
			return
		}
		defer func() { writer.Close() }()

		err = encoder(writer, input)
		if err != nil {
			return
		}
	}

	return
}

func main() {
	pflag.Parse()

	args := pflag.Args()
	if len(args) != 0 {
		pflag.Usage()
		os.Exit(1)
	}

	err := evaluate()
	if err != nil {
		panic(err)
	}
}
