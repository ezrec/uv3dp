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

func (mod *resinModifier) Exposure() (exposure uv3dp.Exposure) {
	exposure = mod.Resin.Exposure

	return
}

func (mod *resinModifier) Bottom() (bottom uv3dp.Bottom) {
	bottom = mod.Resin.Bottom

	return
}

func (mod *resinModifier) LayerExposure(index int) (exposure uv3dp.Exposure) {
	exp := mod.Resin.Exposure
	bot := mod.Resin.Bottom.Exposure
	bottomCount := mod.Resin.Bottom.Count

	if index < bottomCount {
		exposure = bot
	} else {
		exposure = exp
	}

	return
}

func (cmd *ResinCommand) Filter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	// Clone the resin defaults from the source printable
	resin := &Resin{
		Exposure: input.Exposure(),
		Bottom:   input.Bottom(),
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
