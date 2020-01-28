//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/pflag"
	"golang.org/x/image/draw"
	"image"

	"github.com/ezrec/uv3dp"
)

// Predefined bed layouts
var (
	machineMap = map[string]struct {
		X, Y     int
		Xmm, Ymm float32
	}{
		"Anycubic-Photon": {1440, 2560, 68.04, 120.96},
		"Elogoo-Mars":     {1440, 2560, 68.04, 120.96},
		"EPAX-X1":         {1440, 2560, 68.04, 120.96},
		"EPAX-X9":         {1600, 2560, 120.0, 192.0},
		"EPAX-X10":        {1600, 2560, 135.0, 216.0},
		"EPAX-X133":       {2160, 3840, 165.0, 293.0},
		"EPAX-X156":       {2160, 3840, 194.0, 345.0},
	}
)

type BedCommand struct {
	*pflag.FlagSet

	Pixels      []int
	Millimeters []float32
	Machine     string
}

func NewBedCommand() (bc *BedCommand) {
	bc = &BedCommand{
		FlagSet: pflag.NewFlagSet("bed", pflag.ContinueOnError),
	}

	bc.IntSliceVarP(&bc.Pixels, "pixels", "p", []int{1440, 2560}, "Bed size, in pixels")
	bc.Float32SliceVarP(&bc.Millimeters, "millimeters", "m", []float32{68.04, 120.96}, "Bed size, in millimeters")

	bc.StringVarP(&bc.Machine, "machine", "M", "EPAX-X1", "Size preset by machine type")
	bc.SetInterspersed(false)

	return
}

func (bc *BedCommand) PrintDefaults() {
	bc.FlagSet.PrintDefaults()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Machines:")
	fmt.Fprintln(os.Stderr)

	keys := []string{}
	for key, _ := range machineMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		item := machineMap[key]
		fmt.Fprintf(os.Stderr, "    %-20s %dx%d, %.3gx%.3g mm\n", key, item.X, item.Y, item.Xmm, item.Ymm)
	}
}

func (bc *BedCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	srcSize := input.Properties().Size
	dstSize := srcSize

	if bc.Changed("machine") {
		size := machineMap[bc.Machine]
		dstSize.X = size.X
		dstSize.Y = size.Y
		dstSize.Millimeter.X = size.Xmm
		dstSize.Millimeter.Y = size.Ymm
	}

	if bc.Changed("pixels") {
		dstSize.X = bc.Pixels[0]
		dstSize.Y = bc.Pixels[1]
	}

	if bc.Changed("millimeters") {
		dstSize.Millimeter.X = bc.Millimeters[0]
		dstSize.Millimeter.Y = bc.Millimeters[1]
	}

	// Compute the X & Y scaling
	dstXPpm := dstSize.Millimeter.X / float32(dstSize.X)
	dstYPpm := dstSize.Millimeter.Y / float32(dstSize.Y)

	// Compute desitination rectange

	// First, get the size of the src bed, scaled to the size in dest pixels
	dstRect := image.Rect(0, 0, int(srcSize.Millimeter.X/dstXPpm), int(srcSize.Millimeter.Y/dstYPpm))

	// Center on bed
	dstRect = dstRect.Add(image.Point{
		X: (dstSize.X - dstRect.Max.X) / 2,
		Y: (dstSize.Y - dstRect.Max.Y) / 2,
	})

	fmt.Printf("Transformation: %dx%d (%.3gx%.3g mm) => [%d,%d - %d,%d]\n",
		srcSize.X, srcSize.Y, srcSize.Millimeter.X, srcSize.Millimeter.Y,
		dstRect.Min.X, dstRect.Min.Y, dstRect.Max.X, dstRect.Max.Y)

	bm := &bedModifier{
		Printable: input,
		size:      dstSize,
		dstRect:   dstRect,
	}

	output = bm

	return
}

// bedModifier modifies the given printable to have the new size
type bedModifier struct {
	uv3dp.Printable

	size    uv3dp.Size
	dstRect image.Rectangle
}

func (bm *bedModifier) Properties() (prop uv3dp.Properties) {
	prop = bm.Printable.Properties()

	prop.Size = bm.size

	return
}

func (bm *bedModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = bm.Printable.Layer(index)

	// Re-bed the layer to the new size
	newImage := image.NewGray(image.Rect(0, 0, bm.size.X, bm.size.Y))
	draw.NearestNeighbor.Scale(newImage, bm.dstRect, layer.Image, layer.Image.Bounds(), draw.Src, nil)

	layer.Image = newImage

	return
}
