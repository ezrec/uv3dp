//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package ctb handle input and output of Chitubox DLP/LCD printables
package ctb

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_ctb_2 = map[string]uv3dp.Machine{
		"ld-002r": {Vendor: "Creality", Model: "LD-002R", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 68.04, Ymm: 120.96}},
		"x1n":     {Vendor: "EPAX", Model: "X1N", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 68.04, Ymm: 120.96}},
		"x1k":     {Vendor: "EPAX", Model: "X1K", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 68.04, Ymm: 120.96}},
		"x10n":    {Vendor: "EPAX", Model: "X10", Size: uv3dp.MachineSize{X: 1600, Y: 2560, Xmm: 135.0, Ymm: 216.0}},
	}

	machines_ctb_3 = map[string]uv3dp.Machine{
		"mars2-pro":     {Vendor: "Elegoo", Model: "Mars 2 Pro", Size: uv3dp.MachineSize{X: 1620, Y: 2560, Xmm: 82.62, Ymm: 130.56}},
		"sonic-mini-4k": {Vendor: "Phrozen", Model: "Sonic Mini 4K", Size: uv3dp.MachineSize{X: 3840, Y: 2160, Xmm: 134.4, Ymm: 75.6}},
		"e6":            {Vendor: "EPAX", Model: "E6 mono", Size: uv3dp.MachineSize{X: 1620, Y: 2560, Xmm: 81.0, Ymm: 128.0}},
		"e10-4k":        {Vendor: "EPAX", Model: "E10 mono 4K", Size: uv3dp.MachineSize{X: 2400, Y: 3840, Xmm: 120.0, Ymm: 192.0}},
		"e10-5k":        {Vendor: "EPAX", Model: "E10 mono 5K", Size: uv3dp.MachineSize{X: 2880, Y: 4920, Xmm: 135.0, Ymm: 216.0}},
	}
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".ctb", newFormatter)

	uv3dp.RegisterMachines(machines_ctb_2, ".ctb", "--version=2")
	uv3dp.RegisterMachines(machines_ctb_3, ".ctb", "--version=3")
}
