//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"fmt"
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
	LightOnTime   time.Duration // Exposure time
	LightOffTime  time.Duration // Cool down time
	LiftHeight    float32       // mm
	LiftSpeed     float32       // mm/min
	RetractHeight float32       // mm
	RetractSpeed  float32       // mm/min
}

// Total duration of an exposure
func (exp *Exposure) Duration() (total time.Duration) {
	total = exp.LightOnTime + exp.LightOffTime

	// Motion is lift; then retract -> move back to start at retract speed
	total += time.Duration(exp.LiftHeight / exp.LiftSpeed * float32(time.Second))
	total += time.Duration((exp.LiftHeight + exp.RetractHeight*2) / exp.RetractSpeed * float32(time.Second))
	return
}

// Interpolate scales settings between this and another Exposure
func (exp *Exposure) Interpolate(target Exposure, scale float32) (result Exposure) {
	result.LightOnTime = exp.LightOnTime + time.Duration(float64(target.LightOnTime-exp.LightOnTime)*float64(scale))
	result.LightOffTime = exp.LightOffTime + time.Duration(float64(target.LightOffTime-exp.LightOffTime)*float64(scale))
	result.LiftHeight = exp.LiftHeight + (target.LiftHeight-exp.LiftHeight)*scale
	result.LiftSpeed = exp.LiftSpeed + (target.LiftSpeed-exp.LiftSpeed)*scale
	result.RetractHeight = exp.RetractHeight + (target.RetractHeight-exp.RetractHeight)*scale
	result.RetractSpeed = exp.RetractSpeed + (target.RetractSpeed-exp.RetractSpeed)*scale

	return
}

// Bottom layer style
type BottomStyle int

const (
	BottomStyleSlow = BottomStyle(iota) // Abruptly transition from slow to normal exposure
	BottomStyleFade                     // Gradually transition for slow to normal layers
)

func (bs BottomStyle) String() string {
	switch bs {
	case BottomStyleSlow:
		return "slow"
	case BottomStyleFade:
		return "fade"
	default:
		return "unknown"
	}
}

// Bottom layer exposure
type Bottom struct {
	Exposure             // Exposure
	Count    int         // Number of bottom layers
	Style    BottomStyle // Transition style
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
	Metadata map[string](interface{})
}

// Get image bounds
func (prop *Properties) Bounds() image.Rectangle {
	return image.Rect(0, 0, prop.Size.X, prop.Size.Y)
}

// Duration returns total printing time
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

// LayerExposure gets the default exposure by layer index
func (prop *Properties) LayerExposure(index int) (exposure Exposure) {
	if index >= prop.Bottom.Count {
		exposure = prop.Exposure
		return
	}

	switch prop.Bottom.Style {
	case BottomStyleSlow:
		exposure = prop.Bottom.Exposure
	case BottomStyleFade:
		exposure = prop.Bottom.Exposure.Interpolate(prop.Exposure, float32(index)/float32(prop.Bottom.Count))
	default:
		panic(fmt.Sprintf("Unknown Properties.Bottom.Style: %+v", prop.Bottom.Style))
	}

	return
}
