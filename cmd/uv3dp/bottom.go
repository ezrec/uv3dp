//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type BottomCommand struct {
	*pflag.FlagSet

	Style        string // Style (either 'slow' or 'fade')
	LightOnTime  time.Duration
	LightOffTime time.Duration
	Count        int
}

func NewBottomCommand() (cmd *BottomCommand) {
	cmd = &BottomCommand{
		FlagSet: pflag.NewFlagSet("bottom", pflag.ContinueOnError),
	}

	cmd.IntVarP(&cmd.Count, "count", "c", 0, "Bottom layer count")
	cmd.StringVarP(&cmd.Style, "style", "s", "slow", "Bottom layer style - 'fade' or 'slow'")
	cmd.DurationVarP(&cmd.LightOnTime, "light-on", "o", time.Duration(0), "Bottom layer light-on time")
	cmd.DurationVar(&cmd.LightOffTime, "light-off", time.Duration(0), "Bottom layer light-off time")

	cmd.SetInterspersed(false)

	return
}

type bottomModifier struct {
	uv3dp.Printable
	Bottom uv3dp.Bottom
}

func (mod *bottomModifier) Properties() (prop uv3dp.Properties) {
	prop = mod.Printable.Properties()

	// Set the bottom exposure
	prop.Bottom = mod.Bottom

	return
}

func (mod *bottomModifier) Layer(index int) (layer uv3dp.Layer) {
	layer = mod.Printable.Layer(index)

	if index < mod.Bottom.Count {
		layer.Exposure = &mod.Bottom.Exposure
	}

	return
}

func (cmd *BottomCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	prop := input.Properties()

	bot := prop.Bottom

	if cmd.Changed("count") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom layer count %v", cmd.Count)
		bot.Count = int(cmd.Count)
	}

	if cmd.Changed("style") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom layer style %v", cmd.Style)
		styleMap := map[string]uv3dp.BottomStyle{
			"slow": uv3dp.BottomStyleSlow,
			"fade": uv3dp.BottomStyleFade,
		}
		style, found := styleMap[cmd.Style]
		if !found {
			panic(fmt.Sprintf("exposure: Invalid --style=%v", cmd.Style))
		}
		bot.Style = style
	}

	if cmd.Changed("light-on") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom time to %v", cmd.LightOnTime)
		bot.Exposure.LightOnTime = cmd.LightOnTime
	}

	if cmd.Changed("light-off") {
		TraceVerbosef(VerbosityNotice, "  Setting default bottom off time to %v", cmd.LightOffTime)
		bot.Exposure.LightOffTime = cmd.LightOffTime
	}

	mod := &bottomModifier{
		Printable: input,
		Bottom:    bot,
	}

	output = mod

	return
}
