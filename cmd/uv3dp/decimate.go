//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type DecimateCommand struct {
	*pflag.FlagSet

	Passes int
	Bottom bool
	Normal bool
}

func NewDecimateCommand() (cmd *DecimateCommand) {
	flagSet := pflag.NewFlagSet("cmd", pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cmd = &DecimateCommand{
		FlagSet: flagSet,
	}

	cmd.BoolVarP(&cmd.Bottom, "bottom", "b", false, "Decimate bottom layers only")
	cmd.BoolVarP(&cmd.Normal, "normal", "n", false, "Decimate normal layers only")
	cmd.IntVarP(&cmd.Passes, "passes", "p", 1, "Number of decimation passes")

	cmd.SetInterspersed(false)

	return
}

func (cmd *DecimateCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	if cmd.Bottom && cmd.Normal {
		err = fmt.Errorf("only one of --bottom and --normal may be selected")
		return
	}

	dec := uv3dp.NewDecimatedPrintable(input)

	dec.Passes = cmd.Passes

	layers := input.Properties().Size.Layers
	botCount := input.Properties().Bottom.Count

	if cmd.Bottom {
		dec.FirstLayer = 0
		dec.Layers = botCount
	}

	if cmd.Normal {
		dec.FirstLayer = botCount
		dec.Layers = layers - botCount
	}

	output = dec

	return
}
