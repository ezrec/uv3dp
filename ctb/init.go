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
		"ld-002r": {Vendor: "Creality", Model: "LD-002R", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
		"x1n":     {Vendor: "EPAX", Model: "X1N", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
		"x1k":     {Vendor: "EPAX", Model: "X1K", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
		"x10n":    {Vendor: "EPAX", Model: "X10", Size: uv3dp.MachineSize{1600, 2560, 135.0, 216.0}},
	}

	machines_ctb_3 = map[string]uv3dp.Machine{
		"mars2-pro":     {Vendor: "Elegoo", Model: "Mars 2 Pro", Size: uv3dp.MachineSize{1620, 2560, 82.62, 130.56}},
		"sonic-mini-4k": {Vendor: "Phrozen", Model: "Sonic Mini 4K", Size: uv3dp.MachineSize{3840, 2160, 134.4, 75.6}},
		"e6":            {Vendor: "EPAX", Model: "E6 mono", Size: uv3dp.MachineSize{1620, 2560, 81.0, 128.0}},
		"e10-4k":        {Vendor: "EPAX", Model: "E10 mono 4K", Size: uv3dp.MachineSize{2400, 3840, 120.0, 192.0}},
		"e10-5k":        {Vendor: "EPAX", Model: "E10 mono 5K", Size: uv3dp.MachineSize{2880, 4920, 135.0, 216.0}},
	}
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".ctb", newFormatter)

	uv3dp.RegisterMachines(machines_ctb_2, ".ctb", "--version=2")
	uv3dp.RegisterMachines(machines_ctb_3, ".ctb", "--version=3")
}
