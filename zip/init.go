//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package 'kelant' handles Kelant S400 style .zip files
package kelant

import (
	"github.com/ezrec/uv3dp"
)

var (
	machine_zip = map[string]uv3dp.Machine{
		"s400": {Vendor: "Kelant", Model: "S400", Size: uv3dp.MachineSize{2560, 1600, 192.0, 120.0}},
	}
)

func init() {
	// Register zip formatter

	// Bind zip formatter to Kelant S400 machine
}
