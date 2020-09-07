//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package cbddlp handle input and output of Chitubox DLP/LCD printables
package cbddlp

import (
	"github.com/ezrec/uv3dp"
)

var machines_photon = map[string]uv3dp.Machine{
	"photon": {Vendor: "Anycubic", Model: "Photon", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
}

var machines_cbddlp = map[string]uv3dp.Machine{
	"mars": {Vendor: "Elegoo", Model: "Mars", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
	"x1":   {Vendor: "EPAX", Model: "X1", Size: uv3dp.MachineSize{1440, 2560, 68.04, 120.96}},
	"x9":   {Vendor: "EPAX", Model: "X9", Size: uv3dp.MachineSize{1600, 2560, 120.0, 192.0}},
	"x10":  {Vendor: "EPAX", Model: "X10", Size: uv3dp.MachineSize{1600, 2560, 135.0, 216.0}},
	"x133": {Vendor: "EPAX", Model: "X133", Size: uv3dp.MachineSize{2160, 3840, 165.0, 293.0}},
	"x156": {Vendor: "EPAX", Model: "X156", Size: uv3dp.MachineSize{2160, 3840, 194.0, 345.0}},
}

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".cbddlp", newFormatter)
	uv3dp.RegisterFormatter(".photon", newFormatter)

	uv3dp.RegisterMachines(machines_photon, ".photon")
	uv3dp.RegisterMachines(machines_cbddlp, ".cbddlp")
}
