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

	ResinName string

	LightOnTime time.Duration

	BottomStyle       string // Style (either 'slow' or 'fade')
	BottomLightOnTime time.Duration
	BottomCount       int
}

func NewExposureCommand() (ec *ExposureCommand) {
	ec = &ExposureCommand{
		FlagSet: pflag.NewFlagSet("exposure", pflag.ContinueOnError),
	}

	ec.StringVarP(&ec.ResinName, "resin", "r", "", "Resin type [see 'Known resins' in help]")

	ec.DurationVarP(&ec.LightOnTime, "exposure", "e", time.Duration(0), "Normal layer light-on time")

	ec.IntVarP(&ec.BottomCount, "bottom-count", "c", 0, "Bottom layer count")
	ec.StringVarP(&ec.BottomStyle, "bottom-style", "s", "slow", "Bottom layer style - 'fade' or 'slow'")
	ec.DurationVarP(&ec.BottomLightOnTime, "bottom-exposure", "b", time.Duration(0), "Bottom layer light-on time")

	ec.SetInterspersed(false)

	return
}

type exposureModifier struct {
	uv3dp.Printable
	Resin
}

func (em *exposureModifier) Properties() (prop uv3dp.Properties) {
	prop = em.Printable.Properties()

	// Set the bottom and normal exposure from the resins
	prop.Bottom = em.Resin.Bottom
	prop.Exposure = em.Resin.Exposure

	return
}

func (em *exposureModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = em.Printable.Layer(index)

	exp := &em.Resin.Exposure
	bot := &em.Resin.Bottom.Exposure
	bottomCount := em.Resin.Bottom.Count

	if index < bottomCount {
		layer.Exposure = bot
	} else {
		layer.Exposure = exp
	}

	return
}

func (ec *ExposureCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	prop := input.Properties()

	// Clone the resin defaults from the source printable
	resin := &Resin{
		Exposure: prop.Exposure,
		Bottom:   prop.Bottom,
	}

	if ec.Changed("resin") {
		var ok bool
		resin, ok = ResinMap[ec.ResinName]
		if !ok {
			err = fmt.Errorf("unknown resin name \"%v\"", ec.ResinName)
			return
		}
		TraceVerbosef(VerbosityNotice, "  Setting default resin to %v", resin.Name)
	}

	if ec.Changed("exposure") {
		TraceVerbosef(VerbosityNotice, "  Setting default exposure time to %v", ec.LightOnTime)
		resin.Exposure.LightOnTime = ec.LightOnTime
	}

	if ec.Changed("bottom-count") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom layer count %v", ec.BottomCount)
		resin.Bottom.Count = int(ec.BottomCount)
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
		resin.Bottom.Style = style
	}

	if ec.Changed("bottom-exposure") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom time to %v", ec.BottomLightOnTime)
		resin.Bottom.Exposure.LightOnTime = ec.BottomLightOnTime
	}

	em := &exposureModifier{
		Printable: input,
		Resin:     *resin,
	}

	output = em

	return
}
