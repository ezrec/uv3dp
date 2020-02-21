//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"time"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type ExposureCommand struct {
	*pflag.FlagSet

	LightOnTime  time.Duration
	LightOffTime time.Duration
}

func NewExposureCommand() (cmd *ExposureCommand) {
	cmd = &ExposureCommand{
		FlagSet: pflag.NewFlagSet("exposure", pflag.ContinueOnError),
	}

	cmd.DurationVarP(&cmd.LightOnTime, "light-on", "o", time.Duration(0), "Normal layer light-on time")
	cmd.DurationVar(&cmd.LightOffTime, "light-off", time.Duration(0), "Normal layer light-off time")

	cmd.SetInterspersed(false)

	return
}

type exposureModifier struct {
	uv3dp.Printable

	Exposure    uv3dp.Exposure
	BottomCount int
}

func (mod *exposureModifier) Properties() (prop uv3dp.Properties) {
	prop = mod.Printable.Properties()

	// Set the normal exposure
	prop.Exposure = mod.Exposure

	return
}

func (mod *exposureModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = mod.Printable.Layer(index)

	if index >= mod.BottomCount {
		layer.Exposure = &mod.Exposure
	}

	return
}

func (cmd *ExposureCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	prop := input.Properties()

	exp := prop.Exposure

	if cmd.Changed("light-on") {
		TraceVerbosef(VerbosityNotice, "  Setting default exposure time to %v", cmd.LightOnTime)
		exp.LightOnTime = cmd.LightOnTime
	}

	if cmd.Changed("light-off") {
		TraceVerbosef(VerbosityNotice, "  Setting default light off time to %v", cmd.LightOffTime)
		exp.LightOffTime = cmd.LightOffTime
	}

	mod = &exposureModifier{
		Printable:   input,
		Exposure:    exp,
		BottomCount: prop.Bottom.Count,
	}

	return
}
