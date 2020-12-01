//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"math"

	"image"
	"image/color"

	"github.com/spf13/pflag"
	"golang.org/x/image/draw"

	"github.com/ezrec/uv3dp"
)

type BedCommand struct {
	*pflag.FlagSet

	Pixels      []int
	Millimeters []float32
	Machine     string
	Reflect     bool
}

func NewBedCommand() (bc *BedCommand) {
	bc = &BedCommand{
		FlagSet: pflag.NewFlagSet("bed", pflag.ContinueOnError),
	}

	bc.IntSliceVarP(&bc.Pixels, "pixels", "p", []int{1440, 2560}, "Bed size, in pixels")
	bc.Float32SliceVarP(&bc.Millimeters, "millimeters", "m", []float32{68.04, 120.96}, "Bed size, in millimeters")

	bc.StringVarP(&bc.Machine, "machine", "M", "EPAX-X1", "Size preset by machine type")
	bc.BoolVarP(&bc.Reflect, "reflect", "r", false, "Mirror image along the X axis")
	bc.SetInterspersed(false)

	return
}

func (bc *BedCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	srcSize := input.Size()
	dstSize := srcSize
	rotate := false

	if bc.Changed("machine") {
		machine, found := uv3dp.MachineFormats[bc.Machine]
		if !found {
			err = fmt.Errorf("machine '%s' is not a known machine type", bc.Machine)
			return
		}
		size := machine.Machine.Size
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

	// Determine if we need to rotate
	origSize := srcSize
	if (dstSize.X > dstSize.Y) != (srcSize.X > srcSize.Y) {
		rotate = true
		srcSize.X = origSize.Y
		srcSize.Y = origSize.X
		srcSize.Millimeter.X = origSize.Millimeter.Y
		srcSize.Millimeter.Y = origSize.Millimeter.X
	}

	// Compute the X & Y scaling
	dstXPpm := dstSize.Millimeter.X / float32(dstSize.X)
	dstYPpm := dstSize.Millimeter.Y / float32(dstSize.Y)

	// Compute desitination rectange

	// First, get the size of the src bed, scaled to the size in dest pixels
	dstRect := image.Rect(0, 0, int(math.Round(float64(srcSize.Millimeter.X/dstXPpm))), int(math.Round(float64(srcSize.Millimeter.Y/dstYPpm))))

	// Center on bed
	dstRect = dstRect.Add(image.Point{
		X: (dstSize.X - dstRect.Max.X) / 2,
		Y: (dstSize.Y - dstRect.Max.Y) / 2,
	})

	var action string
	if rotate {
		action = " => rotate"
	}

	if bc.Reflect {
		action += " => reflect"
	}

	fmt.Printf("Transformation: %dx%d (%.3gx%.3g mm)%s => [%d,%d - %d,%d]\n",
		origSize.X, origSize.Y, origSize.Millimeter.X, origSize.Millimeter.Y,
		action,
		dstRect.Min.X, dstRect.Min.Y, dstRect.Max.X, dstRect.Max.Y)

	bm := &bedModifier{
		Printable: input,
		size:      dstSize,
		rotate:    rotate,
		dstRect:   dstRect,
		reflect:   bc.Reflect,
	}

	output = bm

	return
}

// rotateImage returns an image rotated 90 degrees
type rotateImage struct {
	image.Image
}

func (ri *rotateImage) At(x, y int) color.Color {
	return ri.Image.At(y, x)
}

func (ri *rotateImage) Bounds() image.Rectangle {
	rect := ri.Image.Bounds()
	return image.Rect(rect.Min.Y, rect.Min.X, rect.Max.Y, rect.Max.X)
}

// reflectImage returns an image reflected along the X axis
type reflectImage struct {
	image.Image
	dX int
}

func (ri *reflectImage) At(x, y int) color.Color {
	return ri.Image.At(ri.dX-x, y)
}

// bedModifier modifies the given printable to have the new size
type bedModifier struct {
	uv3dp.Printable

	size    uv3dp.Size
	dstRect image.Rectangle
	rotate  bool
	reflect bool
}

func (bm *bedModifier) Size() (size uv3dp.Size) {
	size = bm.size

	return
}

func (bm *bedModifier) LayerImage(index int) (newImage *image.Gray) {
	srcImage := image.Image(bm.Printable.LayerImage(index))

	// Re-bed the layer to the new size
	newImage = image.NewGray(image.Rect(0, 0, bm.size.X, bm.size.Y))

	reflect := bm.reflect

	// Our trivial rotation also causes a reflection, so invert the reflect operand
	if bm.rotate {
		srcImage = &rotateImage{Image: srcImage}
		reflect = !reflect
	}

	if reflect {
		bounds := srcImage.Bounds()
		dX := bounds.Min.X + (bounds.Max.X - 1)
		srcImage = &reflectImage{Image: srcImage, dX: dX}
	}

	draw.NearestNeighbor.Scale(newImage, bm.dstRect, srcImage, srcImage.Bounds(), draw.Src, nil)

	return
}
