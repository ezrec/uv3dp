//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package ctb handle input and output of Chitubox DLP/LCD printables
package phz

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_phz = map[string]uv3dp.Machine{
		"sonic-mini": {Vendor: "Phrozen", Model: "Sonic Mini", Size: uv3dp.MachineSize{1080, 1920, 68.04, 120.96}},
	}
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewPhzFormatter(suffix) }

	uv3dp.RegisterFormatter(".phz", newFormatter)

	uv3dp.RegisterMachines(machines_phz, ".phz")
}
