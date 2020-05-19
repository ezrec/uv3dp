//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type RetractCommand struct {
	*pflag.FlagSet

	RetractHeight float32
	RetractSpeed  float32
}

func NewRetractCommand() (cmd *RetractCommand) {
	cmd = &RetractCommand{
		FlagSet: pflag.NewFlagSet("retract", pflag.ContinueOnError),
	}

	cmd.Float32VarP(&cmd.RetractHeight, "height", "h", 0.0, "Retract height in mm")
	cmd.Float32VarP(&cmd.RetractSpeed, "speed", "s", 0.0, "Retract speed in mm/min")

	cmd.SetInterspersed(false)

	return
}

type retractModifier struct {
	uv3dp.Printable
	Exposure    uv3dp.Exposure
	BottomCount int
}

func (mod *retractModifier) Properties() (prop uv3dp.Properties) {
	prop = mod.Printable.Properties()

	// Set the bottom and normal retract from the resins
	prop.Exposure = mod.Exposure

	return
}

func (mod *retractModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = mod.Printable.Layer(index)

	if index >= mod.BottomCount {
		layer.Exposure = &mod.Exposure
	}

	return
}

func (cmd *RetractCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	prop := input.Properties()

	exp := prop.Exposure

	if cmd.Changed("height") {
		TraceVerbosef(VerbosityNotice, "  Setting default retract height to %v mm", cmd.RetractHeight)
		exp.RetractHeight = cmd.RetractHeight
	}

	if cmd.Changed("speed") {
		TraceVerbosef(VerbosityNotice, "  Setting default retract speed to %v mm/min", cmd.RetractSpeed)
		exp.RetractSpeed = cmd.RetractSpeed
	}

	mod = &retractModifier{
		Printable:   input,
		Exposure:    exp,
		BottomCount: prop.Bottom.Count,
	}

	return
}
