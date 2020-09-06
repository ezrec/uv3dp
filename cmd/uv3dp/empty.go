//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type EmptyFormatter struct {
	*pflag.FlagSet

	Pixels      []int
	Millimeters []float32
	Machine     string
}

func NewEmptyFormatter() (ef *EmptyFormatter) {
	ef = &EmptyFormatter{
		FlagSet: pflag.NewFlagSet("empty", pflag.ContinueOnError),
	}

	defaultMachine := uv3dp.MachineFormats["photon"]
	size := &defaultMachine.Machine.Size

	ef.IntSliceVarP(&ef.Pixels, "pixels", "p", []int{size.X, size.Y}, "Empty size, in pixels")
	ef.Float32SliceVarP(&ef.Millimeters, "millimeters", "m", []float32{size.Xmm, size.Ymm}, "Empty size, in millimeters")

	ef.StringVarP(&ef.Machine, "machine", "M", "photon", "Size preset by machine type")
	ef.SetInterspersed(false)

	return
}

func (ef *EmptyFormatter) Decode(file uv3dp.Reader, filesize int64) (printable uv3dp.Printable, err error) {
	var prop uv3dp.Properties

	size := &prop.Size

	msize := uv3dp.MachineFormats[ef.Machine].Machine.Size
	size.X = msize.X
	size.Y = msize.Y
	size.Millimeter.X = msize.Xmm
	size.Millimeter.Y = msize.Ymm

	if ef.Changed("pixels") {
		size.X = ef.Pixels[0]
		size.Y = ef.Pixels[1]
	}

	if ef.Changed("millimeters") {
		size.Millimeter.X = ef.Millimeters[0]
		size.Millimeter.Y = ef.Millimeters[1]
	}

	printable = uv3dp.NewEmptyPrintable(prop)

	return
}

func (ef *EmptyFormatter) Encode(writer uv3dp.Writer, p uv3dp.Printable) (err error) {
	return
}
