//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package fdg handle input and output of Voxelab Polaris printers
package fdg

import (
	"github.com/ezrec/uv3dp"
)

var (
	machines_fdg = map[string]uv3dp.Machine{
		"polaris": {Vendor: "Voxelab", Model: "Polaris", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
	}
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".fdg", newFormatter)

	uv3dp.RegisterMachines(machines_fdg, ".fdg")
}
