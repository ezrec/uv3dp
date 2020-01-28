//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
	"time"
)

type SizeMillimeter struct {
	X, Y float32
}

type Size struct {
	X, Y        int            // Printable size in pixels (x,y)
	Millimeter  SizeMillimeter // Printable size in mm
	Layers      int
	LayerHeight float32 // Height of an individual layer
}

// Per-layer exposure
type Exposure struct {
	LightExposure time.Duration // Exposure time
	LightOffTime  time.Duration // Cool down time
	LiftHeight    float32       // mm
	LiftSpeed     float32       // mm/sec
	RetractHeight float32       // mm
	RetractSpeed  float32       // mm/sec
}

// Total duration of an exposure
func (exp *Exposure) Duration() (total time.Duration) {
	total = exp.LightExposure + exp.LightOffTime

	// Motion is lift; then retract -> move back to start at retract speed
	total += time.Duration(exp.LiftHeight / exp.LiftSpeed * float32(time.Second))
	total += time.Duration((exp.LiftHeight + exp.RetractHeight*2) / exp.RetractSpeed * float32(time.Second))
	return
}

// Bottom layer exposure
type Bottom struct {
	Exposure     // Exposure
	Count    int // Number of bottom layers
}

type PreviewType uint

const (
	PreviewTypeTiny = PreviewType(iota)
	PreviewTypeHuge
)

type Properties struct {
	Size     Size
	Exposure Exposure
	Bottom   Bottom
	Preview  map[PreviewType]image.Image
}

// Get image bounds
func (prop *Properties) Bounds() image.Rectangle {
	return image.Rect(0, 0, prop.Size.X, prop.Size.Y)
}

// Get total printing time
func (prop *Properties) Duration() (duration time.Duration) {
	size := &prop.Size
	bot := &prop.Bottom.Exposure
	botCount := prop.Bottom.Count
	exp := &prop.Exposure
	botTime := bot.Duration() * time.Duration(botCount)
	expTime := exp.Duration() * time.Duration(size.Layers-int(botCount))

	duration = botTime + expTime

	return
}

// Everything needed to print a single layer
type Layer struct {
	Z        float32     // Z height in mm
	Exposure *Exposure   // Layer exposure settings
	Image    *image.Gray // Image mask
}

type Printable interface {
	Properties() (prop Properties)
	Layer(index int) (layer Layer)
}
