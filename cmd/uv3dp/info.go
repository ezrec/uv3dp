//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"sort"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type InfoCommand struct {
	*pflag.FlagSet

	SizeSummary     bool
	LayerDetail     bool
	ExposureSummary bool
}

func NewInfoCommand() (info *InfoCommand) {
	flagSet := pflag.NewFlagSet("info", pflag.ContinueOnError)

	info = &InfoCommand{
		FlagSet: flagSet,
	}

	info.SetInterspersed(false)
	info.BoolVarP(&info.SizeSummary, "size", "s", true, "Show size summary")
	info.BoolVarP(&info.ExposureSummary, "exposure", "e", true, "Show summary of the exposure settings")
	info.BoolVarP(&info.LayerDetail, "layer", "l", false, "Show layer detail")

	return
}

func (info *InfoCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	prop := input.Properties()

	if info.SizeSummary {
		size := &prop.Size
		fmt.Printf("Layers: %v, %vx%v slices, %.2f x %.2f x %.2f mm bed required\n",
			size.Layers, size.X, size.Y,
			size.Millimeter.X, size.Millimeter.Y, float32(size.Layers)*size.LayerHeight)
	}

	if info.ExposureSummary {
		exp := &prop.Exposure
		bot := &prop.Bottom
		var bottomStyle string
		switch bot.Style {
		case uv3dp.BottomStyleSlow:
			bottomStyle = "slow"
		case uv3dp.BottomStyleFade:
			bottomStyle = "fade"
		default:
			bottomStyle = fmt.Sprintf("%+v", bot.Style)
		}

		fmt.Printf("Exposure: %.2gs on, %.2gs off",
			exp.LightOnTime,
			exp.LightOffTime)
		if exp.LightPWM != 255 {
			fmt.Printf(", PWM %v", exp.LightPWM)
		}
		fmt.Println()
		fmt.Printf("Bottom: %.2gs on, %.2gs off",
			bot.Exposure.LightOnTime,
			bot.Exposure.LightOffTime)
		if bot.Exposure.LightPWM != 255 {
			fmt.Printf(", PWM %v", bot.Exposure.LightPWM)
		}
		fmt.Printf(" (%v %v layers)\n", bot.Count, bottomStyle)
		fmt.Printf("Lift: %v mm, %v mm/min\n",
			exp.LiftHeight, exp.LiftSpeed)
		fmt.Printf("Retract: %v mm, %v mm/min\n",
			exp.RetractHeight, exp.RetractSpeed)

		keys := []string{}
		for k := range prop.Metadata {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("%v: %v\n", k, prop.Metadata[k])
		}
	}

	if info.LayerDetail {
		for n := 0; n < prop.Size.Layers; n++ {
			layer := input.Layer(n)
			fmt.Printf("%d: @%.2f %+v\n", n, layer.Z, layer.Exposure)
		}
	}

	output = input

	return
}
