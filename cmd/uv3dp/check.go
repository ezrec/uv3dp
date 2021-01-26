//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"

	"github.com/ezrec/uv3dp"
)

type checkModifier struct {
	uv3dp.Printable
}

func (mod *checkModifier) LayerZ(index int) (z float32) {
	z = mod.Printable.LayerZ(index)

	if z < 0.001 {
		panic(fmt.Sprintf("Layer %d: Z value of %.02fmm is too close to the screen", index, z))
	}

    if index > 0 {
        prev_z := mod.Printable.LayerZ(index - 1)
        if z < prev_z {
            panic(fmt.Sprintf("Layer %d: Z value of %.02fmm is below the previous layer at %.02fmm", index, z, prev_z))
        }

        nominal_dz := mod.Printable.Size().LayerHeight

        if (z - prev_z) > nominal_dz*1.5 {
            panic(fmt.Sprintf("Layer %d: Layer height of %.02fmm is too far from nominal of %.02fmm", index, z-prev_z, nominal_dz))
        }
    }

	return
}

func CheckFilter(input uv3dp.Printable) (mod uv3dp.Printable, err error) {
	mod = &checkModifier{
		Printable: input,
	}

	return
}
