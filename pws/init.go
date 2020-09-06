//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package pws handles input and output of Anycubic Photons 2.0 (.pws) printables
package pws

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_pws = map[string]uv3dp.Machine{
		"photons": {Vendor: "Anycubic", Model: "Photon S", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
	}
	machines_pw0 = map[string]uv3dp.Machine{
		"photon0": {Vendor: "Anycubic", Model: "Photon Zero", Size: uv3dp.MachineSize{480, 854, 55.44, 98.64}},
	}
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".pws", newFormatter)
	uv3dp.RegisterFormatter(".pw0", newFormatter)

	uv3dp.RegisterMachines(machines_pws, ".pws")
	uv3dp.RegisterMachines(machines_pw0, ".pw0")
}
