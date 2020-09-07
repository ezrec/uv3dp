//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type ExposureCommand struct {
	*pflag.FlagSet

	LightOnTime  float32
	LightOffTime float32
	LightPWM     uint8
}

func NewExposureCommand() (cmd *ExposureCommand) {
	cmd = &ExposureCommand{
		FlagSet: pflag.NewFlagSet("exposure", pflag.ContinueOnError),
	}

	cmd.Float32VarP(&cmd.LightOnTime, "light-on", "o", 0.0, "Normal layer light-on time in seconds")
	cmd.Float32VarP(&cmd.LightOffTime, "light-off", "f", 0.0, "Normal layer light-off time in seconds")
	cmd.Uint8VarP(&cmd.LightPWM, "pwm", "p", 255, "Light PWM rate (0..255)")

	cmd.SetInterspersed(false)

	return
}

type exposureModifier struct {
	uv3dp.Printable

	exposure uv3dp.Exposure
}

func (mod *exposureModifier) Exposure() (exposure uv3dp.Exposure) {
	// Set the normal exposure
	exposure = mod.exposure

	return
}

func (mod *exposureModifier) LayerExposure(index int) (exposure uv3dp.Exposure) {
	exp := mod.exposure
	bot := mod.Printable.Bottom()

	if index < bot.Count {
		exposure = mod.Printable.LayerExposure(index)
	} else {
		exposure = exp
	}

	return
}

func (cmd *ExposureCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	exp := input.Exposure()

	if cmd.Changed("light-on") {
		TraceVerbosef(VerbosityNotice, "  Setting default exposure time to %v", cmd.LightOnTime)
		exp.LightOnTime = cmd.LightOnTime
	}

	if cmd.Changed("light-off") {
		TraceVerbosef(VerbosityNotice, "  Setting default light off time to %v", cmd.LightOffTime)
		exp.LightOffTime = cmd.LightOffTime
	}

	if cmd.Changed("pwm") {
		TraceVerbosef(VerbosityNotice, "  Setting default light PWM to %v", cmd.LightPWM)
		exp.LightPWM = cmd.LightPWM
	}

	mod = &exposureModifier{
		Printable: input,
		exposure:  exp,
	}

	return
}
