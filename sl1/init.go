//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package sl1 handles input and output of Prusa SL1 DLP/LCD printables
package sl1

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_sl1 = map[string]uv3dp.Machine{
		"sl1": {Vendor: "Prusa", Model: "SL1", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
	}
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".sl1", newFormatter)

	uv3dp.RegisterMachines(machines_sl1, ".sl1")
}
