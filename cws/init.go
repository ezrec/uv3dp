//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package cws handles input and output of NOVA32 CWS printables
package cws

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_cws = map[string]uv3dp.Machine{
		"elfin": {Vendor: "Nova3D", Model: "Elfin", Size: uv3dp.MachineSize{1410, 2550, 73.0, 132.0}},
	}
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewCWSFormatter(suffix) }

	uv3dp.RegisterFormatter(".cws", newFormatter)

	uv3dp.RegisterMachines(machines_cws, ".cws")
}
