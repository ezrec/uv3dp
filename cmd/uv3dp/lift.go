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

	cmd.Float32VarP(&cmd.LiftHeight, "height", "h", 0.0, "Lift height")
	cmd.Float32VarP(&cmd.LiftSpeed, "speed", "s", 0.0, "Lift height")

	cmd.SetInterspersed(false)

	return
}

type liftModifier struct {
	uv3dp.Printable
	Exposure    uv3dp.Exposure
	BottomCount int
}

func (mod *liftModifier) Properties() (prop uv3dp.Properties) {
	prop = mod.Printable.Properties()

	// Set the bottom and normal lift from the resins
	prop.Exposure = mod.Exposure

	return
}

func (mod *liftModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = mod.Printable.Layer(index)

	if index >= mod.BottomCount {
		layer.Exposure = &mod.Exposure
	}

	return
}

func (cmd *LiftCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	prop := input.Properties()

	exp := prop.Exposure

	if cmd.Changed("height") {
		TraceVerbosef(VerbosityNotice, "  Setting default lift height to %v mm", cmd.LiftHeight)
		exp.LiftHeight = cmd.LiftHeight
	}

	if cmd.Changed("speed") {
		TraceVerbosef(VerbosityNotice, "  Setting default lift speed to %v mm/s", cmd.LiftSpeed)
		exp.LiftSpeed = cmd.LiftSpeed
	}

	mod = &liftModifier{
		Printable:   input,
		Exposure:    exp,
		BottomCount: prop.Bottom.Count,
	}

	return
}
