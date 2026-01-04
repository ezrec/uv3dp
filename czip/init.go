//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package 'czip' handles ChiTuBox '.zip' printers (ie Kelant S400 and Phrozen Shuffle)
package czip

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_zip = map[string]uv3dp.Machine{
		"s400":    {Vendor: "Kelant", Model: "S400", Size: uv3dp.MachineSize{X: 2560, Y: 1600, Xmm: 192.0, Ymm: 120.0}},
		"shuffle": {Vendor: "Phrozen", Model: "Shuffle", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 67.68, Ymm: 120.32}},
	}
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".zip", newFormatter)

	uv3dp.RegisterMachines(machines_zip, ".zip")
}
