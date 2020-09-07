//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"image"

	"github.com/spf13/pflag"

	"github.com/ezrec/uv3dp"
)

type SelectCommand struct {
	*pflag.FlagSet

	First int
	Count int
}

func NewSelectCommand() (cmd *SelectCommand) {
	flagSet := pflag.NewFlagSet("cmd", pflag.ContinueOnError)
	flagSet.SetInterspersed(false)

	cmd = &SelectCommand{
		FlagSet: flagSet,
		First:   0,
		Count:   -1,
	}

	cmd.IntVarP(&cmd.First, "first", "f", 0, "First layer to select")
	cmd.IntVarP(&cmd.Count, "count", "c", -1, "Count of layers to select (-1 for all layers after first)")

	return
}

type SelectPrintable struct {
	uv3dp.Printable

	first int
	count int
}

func (sp *SelectPrintable) LayerZ(index int) float32 {
	return sp.Printable.LayerZ(index + sp.first)
}

func (sp *SelectPrintable) LayerExposure(index int) uv3dp.Exposure {
	return sp.Printable.LayerExposure(index + sp.first)
}

func (sp *SelectPrintable) LayerImage(index int) *image.Gray {
	return sp.Printable.LayerImage(index + sp.first)
}

func (sp *SelectPrintable) Size() (size uv3dp.Size) {
	size = sp.Printable.Size()
	size.Layers = sp.count

	return
}

func (cmd *SelectCommand) Filter(input uv3dp.Printable) (output uv3dp.Printable, err error) {
	layers := input.Size().Layers

	first := cmd.First
	count := cmd.Count

	if layers == 0 {
		first = 0
		count = 0
	} else {
		if first >= layers {
			first = layers - 1
		}

		if count < 0 {
			count = layers - first
		}

		if first+count > layers {
			count = layers - first
		}
	}

	sp := &SelectPrintable{
		Printable: input,
		first:     first,
		count:     count,
	}

	output = sp

	return
}
