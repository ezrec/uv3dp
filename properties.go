//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
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
	LightOnTime   float32 // Exposure time
	LightOffTime  float32 // Cool down time
	LightPWM      uint8   // PWM from 1..255
	LiftHeight    float32 // mm
	LiftSpeed     float32 // mm/min
	RetractHeight float32 // mm
	RetractSpeed  float32 // mm/min
}

// Total duration of an exposure
func (exp *Exposure) Duration() (total float32) {
	total = exp.LightOnTime + exp.LightOffTime

	// Motion is lift; then retract -> move back to start at retract speed
	total += exp.LiftHeight / exp.LiftSpeed
	total += (exp.LiftHeight + exp.RetractHeight*2) / exp.RetractSpeed
	return
}

// Interpolate scales settings between this and another Exposure
func (exp *Exposure) Interpolate(target Exposure, scale float32) (result Exposure) {
	result.LightOnTime = exp.LightOnTime + float32(float64(target.LightOnTime-exp.LightOnTime)*float64(scale))
	result.LightOffTime = exp.LightOffTime + float32(float64(target.LightOffTime-exp.LightOffTime)*float64(scale))
	result.LiftHeight = exp.LiftHeight + (target.LiftHeight-exp.LiftHeight)*scale
	result.LiftSpeed = exp.LiftSpeed + (target.LiftSpeed-exp.LiftSpeed)*scale
	result.RetractHeight = exp.RetractHeight + (target.RetractHeight-exp.RetractHeight)*scale
	result.RetractSpeed = exp.RetractSpeed + (target.RetractSpeed-exp.RetractSpeed)*scale

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
	Preview  map[PreviewType]image.Image `json:",omitempty"`
	Metadata map[string](interface{})    `json:",omitempty"`
}

// Get metadata
func (prop *Properties) GetMetadataUint8(attr string, defValue uint8) (value uint8) {
	value = defValue

	tmp, found := prop.Metadata[attr]
	if found {
		u8, ok := tmp.(uint8)
		if ok {
			value = u8
		}
	}

	return
}

// Get image bounds
func (prop *Properties) Bounds() image.Rectangle {
	return image.Rect(0, 0, prop.Size.X, prop.Size.Y)
}

// Duration returns total printing time
func (prop *Properties) Duration() (duration float32) {
	size := &prop.Size
	bot := &prop.Bottom.Exposure
	botCount := prop.Bottom.Count
	exp := &prop.Exposure
	botTime := bot.Duration() * float32(botCount)
	expTime := exp.Duration() * float32(size.Layers-int(botCount))

	duration = botTime + expTime

	return
}

// LayerExposure gets the default exposure by layer index
func (prop *Properties) LayerExposure(index int) (exposure Exposure) {
	if index >= prop.Bottom.Count {
		exposure = prop.Exposure
		return
	}

	exposure = prop.Bottom.Exposure

	return
}
