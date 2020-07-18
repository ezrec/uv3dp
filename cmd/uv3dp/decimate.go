//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/ezrec/uv3dp"
	"github.com/spf13/pflag"
)

type DecimateCommand struct {
	*pflag.FlagSet

	Bottom int
	Normal int
}

func NewDecimateCommand() (cmd *DecimateCommand) {
	flagSet := pflag.NewFlagSet("cmd", pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cmd = &DecimateCommand{
		FlagSet: flagSet,
	}

	cmd.IntVarP(&cmd.Bottom, "bottom", "b", 0, "Number of bottom layer passes")
	cmd.IntVarP(&cmd.Normal, "normal", "n", 1, "Number of normal layer passes")

	cmd.SetInterspersed(false)

	return
}

func (cmd *DecimateCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	layers := input.Properties().Size.Layers
	botCount := input.Properties().Bottom.Count

	if cmd.Bottom > 0 {
		dec := uv3dp.NewDecimatedPrintable(input)

		dec.Passes = cmd.Bottom
		dec.FirstLayer = 0
		dec.Layers = botCount

		input = dec
	}

	if cmd.Normal > 0 {
		dec := uv3dp.NewDecimatedPrintable(input)

		dec.Passes = cmd.Normal
		dec.FirstLayer = botCount
		dec.Layers = layers - botCount

		input = dec
	}

	output = input

	return
}
