//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type InfoCommand struct {
	*pflag.FlagSet

	LayerSummary    bool
	ExposureSummary bool
}

func NewInfoCommand() (info *InfoCommand) {
	flagSet := pflag.NewFlagSet("info", pflag.ContinueOnError)

	info = &InfoCommand{
		FlagSet: flagSet,
	}

	info.SetInterspersed(false)
	info.BoolVarP(&info.LayerSummary, "layers", "l", true, "Show summary of the layers")
	info.BoolVarP(&info.ExposureSummary, "exposure", "e", true, "Show summary of the exposure settings")

	return
}

func (info *InfoCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	prop := input.Properties()

	if info.LayerSummary {
		size := &prop.Size
		fmt.Printf("Layers: %v, %vx%v slices, %.2f x %.2f x %.2f mm bed required\n",
			size.Layers, size.X, size.Y,
			size.Millimeter.X, size.Millimeter.Y, float32(size.Layers)*size.LayerHeight)
	}

	if info.ExposureSummary {
		exp := &prop.Exposure
		bot := &prop.Bottom
		fmt.Printf("Exposure: %v on, %v off nominal, %v bottom (%v layers)\n",
			exp.LightExposure, exp.LightOffTime,
			bot.Exposure.LightExposure, bot.Count)
	}

	output = input

	return
}
