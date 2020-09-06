//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package zcodex handles input and output of Prusa SL1 DLP/LCD printables
package zcodex

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_zcodex = map[string]uv3dp.Machine{
		"inkspire": {Vendor: "Zortrax", Model: "Inkspire", Size: uv3dp.MachineSize{1440, 2560, 72.0, 128.0}},
	}
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewZcodexFormatter(suffix) }

	uv3dp.RegisterFormatter(".zcodex", newFormatter)

	uv3dp.RegisterMachines(machines_zcodex, ".zcodex")
}
