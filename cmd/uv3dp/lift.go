//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type LiftCommand struct {
	*pflag.FlagSet

	LiftHeight float32
	LiftSpeed  float32
}

func NewLiftCommand() (cmd *LiftCommand) {
	cmd = &LiftCommand{
		FlagSet: pflag.NewFlagSet("lift", pflag.ContinueOnError),
	}

	cmd.Float32VarP(&cmd.LiftHeight, "height", "h", 0.0, "Lift height in mm")
	cmd.Float32VarP(&cmd.LiftSpeed, "speed", "s", 0.0, "Lift speed in mm/min")

	cmd.SetInterspersed(false)

	return
}

type liftModifier struct {
	uv3dp.Printable
	exposure uv3dp.Exposure
}

func (mod *liftModifier) Exposure() (exposure uv3dp.Exposure) {
	// Set the bottom and normal lift from the resins
	exposure = mod.exposure

	return
}

func (mod *liftModifier) LayerExposure(index int) (exposure uv3dp.Exposure) {
	exp := mod.exposure
	bot := mod.Printable.Bottom()

	if index < bot.Count {
		exposure = mod.Printable.LayerExposure(index)
	} else {
		exposure = exp
	}

	return
}

func (cmd *LiftCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	exp := input.Exposure()

	if cmd.Changed("height") {
		TraceVerbosef(VerbosityNotice, "  Setting default lift height to %v mm", cmd.LiftHeight)
		exp.LiftHeight = cmd.LiftHeight
	}

	if cmd.Changed("speed") {
		TraceVerbosef(VerbosityNotice, "  Setting default lift speed to %v mm/min", cmd.LiftSpeed)
		exp.LiftSpeed = cmd.LiftSpeed
	}

	mod = &liftModifier{
		Printable: input,
		exposure:  exp,
	}

	return
}
