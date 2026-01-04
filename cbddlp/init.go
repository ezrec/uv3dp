//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package cbddlp handle input and output of Chitubox DLP/LCD printables
package cbddlp

import (
	"github.com/ezrec/uv3dp"
)

var machines_photon = map[string]uv3dp.Machine{
	"photon": {Vendor: "Anycubic", Model: "Photon", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 68.04, Ymm: 120.96}},
}

var machines_cbddlp = map[string]uv3dp.Machine{
	"mars": {Vendor: "Elegoo", Model: "Mars", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 68.04, Ymm: 120.96}},
	"x1":   {Vendor: "EPAX", Model: "X1", Size: uv3dp.MachineSize{X: 1440, Y: 2560, Xmm: 68.04, Ymm: 120.96}},
	"x9":   {Vendor: "EPAX", Model: "X9", Size: uv3dp.MachineSize{X: 1600, Y: 2560, Xmm: 120.0, Ymm: 192.0}},
	"x10":  {Vendor: "EPAX", Model: "X10", Size: uv3dp.MachineSize{X: 1600, Y: 2560, Xmm: 135.0, Ymm: 216.0}},
	"x133": {Vendor: "EPAX", Model: "X133", Size: uv3dp.MachineSize{X: 2160, Y: 3840, Xmm: 165.0, Ymm: 293.0}},
	"x156": {Vendor: "EPAX", Model: "X156", Size: uv3dp.MachineSize{X: 2160, Y: 3840, Xmm: 194.0, Ymm: 345.0}},
}

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".cbddlp", newFormatter)
	uv3dp.RegisterFormatter(".photon", newFormatter)

	uv3dp.RegisterMachines(machines_photon, ".photon")
	uv3dp.RegisterMachines(machines_cbddlp, ".cbddlp")
}
