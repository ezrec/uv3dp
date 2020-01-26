//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type DecimateCommand struct {
	*pflag.FlagSet
}

func NewDecimateCommand() (info *DecimateCommand) {
	flagSet := pflag.NewFlagSet("info", pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	info = &DecimateCommand{
		FlagSet: flagSet,
	}

	return
}

func (info *DecimateCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	output = uv3dp.NewDecimatedPrintable(input)

	return
}
