//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type ResinCommand struct {
	*pflag.FlagSet

	ResinName string
}

func NewResinCommand() (cmd *ResinCommand) {
	cmd = &ResinCommand{
		FlagSet: pflag.NewFlagSet("resin", pflag.ContinueOnError),
	}

	cmd.StringVarP(&cmd.ResinName, "type", "t", "", "Resin type [see 'Known resins' in help]")

	cmd.SetInterspersed(false)

	return
}

type resinModifier struct {
	uv3dp.Printable
	Resin
}

func (mod *resinModifier) Properties() (prop uv3dp.Properties) {
	prop = mod.Printable.Properties()

	// Set the bottom and normal resin from the resins
	prop.Bottom = mod.Resin.Bottom
	prop.Exposure = mod.Resin.Exposure

	return
}

func (mod *resinModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = mod.Printable.Layer(index)

	exp := mod.Resin.Exposure
	bot := mod.Resin.Bottom.Exposure
	bottomCount := mod.Resin.Bottom.Count

	if index < bottomCount {
		layer.Exposure = bot
	} else {
		layer.Exposure = exp
	}

	return
}

func (cmd *ResinCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	prop := input.Properties()

	// Clone the resin defaults from the source printable
	resin := &Resin{
		Exposure: prop.Exposure,
		Bottom:   prop.Bottom,
	}

	if cmd.Changed("type") {
		var ok bool
		resin, ok = ResinMap[cmd.ResinName]
		if !ok {
			err = fmt.Errorf("unknown resin name \"%v\"", cmd.ResinName)
			return
		}
		TraceVerbosef(VerbosityNotice, "  Setting default resin to %v", resin.Name)
	}

	mod = &resinModifier{
		Printable: input,
		Resin:     *resin,
	}

	return
}
