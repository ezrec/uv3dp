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
	exposure uv3dp.Exposure
}

func (mod *retractModifier) Exposure() (exposure uv3dp.Exposure) {
	// Set the bottom and normal retract from the resins
	exposure = mod.exposure

	return
}

func (mod *retractModifier) LayerExposure(index int) (exposure uv3dp.Exposure) {
	bot := mod.Printable.Bottom()
	exp := mod.exposure

	if index < bot.Count {
		exposure = mod.Printable.LayerExposure(index)
	} else {
		exposure = exp
	}

	return
}

func (cmd *RetractCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	exp := input.Exposure()

	if cmd.Changed("height") {
		TraceVerbosef(VerbosityNotice, "  Setting default retract height to %v mm", cmd.RetractHeight)
		exp.RetractHeight = cmd.RetractHeight
	}

	if cmd.Changed("speed") {
		TraceVerbosef(VerbosityNotice, "  Setting default retract speed to %v mm/min", cmd.RetractSpeed)
		exp.RetractSpeed = cmd.RetractSpeed
	}

	mod = &retractModifier{
		Printable: input,
		exposure:  exp,
	}

	return
}
