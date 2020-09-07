//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"image"
	"math"
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
	LightOnTime   float32 // Exposure time
	LightOffTime  float32 // Cool down time
	LightPWM      uint8   `json:",omitempty"` // PWM from 1..255
	LiftHeight    float32 // mm
	LiftSpeed     float32 // mm/min
	RetractHeight float32 `json:",omitempty"` // mm
	RetractSpeed  float32 `json:",omitempty"` // mm/min
}

// Total duration of an exposure
func (exp *Exposure) Duration() (total time.Duration) {
	totalSec := exp.LightOnTime + exp.LightOffTime

	// Motion is lift; then retract -> move back to start at retract speed
	totalSec += exp.LiftHeight / exp.LiftSpeed * 60
	totalSec += (exp.LiftHeight + exp.RetractHeight*2) / exp.RetractSpeed * 60

	total = time.Duration(totalSec * float32(time.Second))

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

func (prop *Properties) MetadataKeys() (keys []string) {
	for key := range prop.Metadata {
		keys = append(keys, key)
	}

	return
}

// Get image bounds
func (prop *Properties) Bounds() image.Rectangle {
	return image.Rect(0, 0, prop.Size.X, prop.Size.Y)
}

// Exposure gets the default exposure by layer index
func (prop *Properties) LayerExposure(index int) (exposure Exposure) {
	if index >= prop.Bottom.Count {
		exposure = prop.Exposure
	} else {
		exposure = prop.Bottom.Exposure
	}

	// Validate LightPWM
	if exposure.LightPWM == 0 {
		exposure.LightPWM = 255
	}

	return
}

// Z get the default Z height at a layer index
func (prop *Properties) LayerZ(index int) (z float32) {
	return float32(math.Round(float64(prop.Size.LayerHeight)*float64(index+1)*100) / 100.0)
}
