//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"sort"
	"time"

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

func printExposure(mode string, exp *uv3dp.Exposure) {
	fmt.Printf("%v:\n", mode)
	fmt.Printf("  Exposure: %.2gs on, %.2gs off", exp.LightOnTime, exp.LightOffTime)
	if exp.LightPWM != 255 {
		fmt.Printf(", PWM %v", exp.LightPWM)
	}
	fmt.Printf("  Lift: %v mm, %v mm/min\n",
		exp.LiftHeight, exp.LiftSpeed)
	fmt.Printf("  Retract: %v mm, %v mm/min\n",
		exp.RetractHeight, exp.RetractSpeed)
}

func (info *InfoCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	exp := input.Exposure()
	bot := input.Bottom()

	if info.SizeSummary {
		size := input.Size()
		fmt.Printf("Layers: %v, %vx%v slices, %.2f x %.2f x %.2f mm bed required\n",
			size.Layers, size.X, size.Y,
			size.Millimeter.X, size.Millimeter.Y, float32(size.Layers)*size.LayerHeight)
		exposureTime := time.Duration(0)
		for n := 0; n < size.Layers; n++ {
			exposureTime += time.Duration(input.LayerExposure(n).LightOnTime * float32(time.Second))
		}
		exposureTime = exposureTime.Truncate(time.Second)
		totalTime := uv3dp.PrintDuration(input).Truncate(time.Second)

		fmt.Printf("Total time: %v (%v exposure, %v motion)\n",
			totalTime, exposureTime, totalTime-exposureTime)
	}

	if info.ExposureSummary {
		printExposure(fmt.Sprintf("Bottom (%v layers)", bot.Count), &bot.Exposure)
		printExposure("Normal", &exp)

		keys := input.MetadataKeys()

		sort.Strings(keys)

		for _, k := range keys {
			data, _ := input.Metadata(k)
			fmt.Printf("%v: %v\n", k, data)
		}
	}

	if info.LayerDetail {
		size := input.Size()
		for n := 0; n < size.Layers; n++ {
			layerZ := input.LayerZ(n)
			layerExposure := input.LayerExposure(n)
			fmt.Printf("%d: @%.2f %+v\n", n, layerZ, layerExposure)
		}
	}

	output = input

	return
}
