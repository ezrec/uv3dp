//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package lgs handles input and output of Longer Orange 10 print files
package lgs

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_lgs = map[string]uv3dp.Machine{
		"orange10": {Vendor: "Longer", Model: "Orange 10", Size: uv3dp.MachineSize{480, 854, 55.44, 98.64}},
	}
	machines_lgs30 = map[string]uv3dp.Machine{
		"orange30": {Vendor: "Longer", Model: "Orange 30", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
	}
)

func init() {
	newFormatter_10 := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix, 10) }
	newFormatter_30 := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix, 30) }

	uv3dp.RegisterFormatter(".lgs", newFormatter_10)
	uv3dp.RegisterFormatter(".lgs30", newFormatter_30)

	uv3dp.RegisterMachines(machines_lgs, ".lgs")
	uv3dp.RegisterMachines(machines_lgs30, ".lgs30")
}
