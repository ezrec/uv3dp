//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type ExposureCommand struct {
	*pflag.FlagSet

	Exposure time.Duration // Time to expose a normal layer

	BottomCount    uint // Number of bottom layers
	BottomExposure time.Duration
	BottomStyle    string // Style (either 'slow' or 'fade')
}

func NewExposureCommand() (ec *ExposureCommand) {
	ec = &ExposureCommand{
		FlagSet: pflag.NewFlagSet("exposure", pflag.ContinueOnError),
	}

	ec.DurationVarP(&ec.Exposure, "exposure", "e", time.Duration(0), "Normal layer light-on time")
	ec.UintVarP(&ec.BottomCount, "bottom-count", "c", 0, "Bottom layer count")
	ec.StringVarP(&ec.BottomStyle, "bottom-style", "s", "slow", "Bottom layer style - 'fade' or 'slow'")
	ec.DurationVarP(&ec.BottomExposure, "bottom-exposure", "b", time.Duration(0), "Bottom layer light-on time")
	ec.SetInterspersed(false)

	return
}

type exposureModifier struct {
	uv3dp.Printable
	properties uv3dp.Properties
}

func (em *exposureModifier) Properties() (prop uv3dp.Properties) {
	prop = em.properties
	return
}

func (em *exposureModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = em.Printable.Layer(index)

	exp := &em.properties.Exposure
	bot := &em.properties.Bottom.Exposure
	bottomCount := em.properties.Bottom.Count

	if index < bottomCount {
		layer.Exposure = bot
	} else {
		layer.Exposure = exp
	}

	return
}

func (ec *ExposureCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	em := &exposureModifier{
		Printable:  input,
		properties: input.Properties(),
	}

	if ec.Changed("exposure") {
		TraceVerbosef(VerbosityNotice, "  Setting default exposure time to %v", ec.Exposure)
		em.properties.Exposure.LightExposure = ec.Exposure
	}

	if ec.Changed("bottom-count") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom layer count %v", ec.BottomCount)
		em.properties.Bottom.Count = int(ec.BottomCount)
	}

	if ec.Changed("bottom-style") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom layer style %v", ec.BottomStyle)
		styleMap := map[string]uv3dp.BottomStyle{
			"slow": uv3dp.BottomStyleSlow,
			"fade": uv3dp.BottomStyleFade,
		}
		style, found := styleMap[ec.BottomStyle]
		if !found {
			panic(fmt.Sprintf("exposure: Invalid --bottom-style=%v", ec.BottomStyle))
		}
		em.properties.Bottom.Style = style
	}

	if ec.Changed("bottom-exposure") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom time to %v", ec.BottomExposure)
		em.properties.Bottom.Exposure.LightExposure = ec.BottomExposure
	}

	output = em

	return
}
