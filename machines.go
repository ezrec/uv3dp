//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

import (
	"fmt"
)

type MachineSize struct {
	X, Y     int
	Xmm, Ymm float32
}

type Machine struct {
	Vendor string
	Model  string
	Size   MachineSize
}

type MachineFormat struct {
	Machine
	Extension string
	Args      []string
}

var (
	MachineFormats = map[string](*MachineFormat){}
)

func RegisterMachine(name string, machine Machine, extension string, args ...string) (err error) {
	_, ok := MachineFormats[name]
	if ok {
		err = fmt.Errorf("name already exists in Machine list")
		return
	}

	machineFormat := &MachineFormat{
		Machine:   machine,
		Extension: extension,
		Args:      args,
	}

	MachineFormats[name] = machineFormat

	return
}

func RegisterMachines(machineMap map[string]Machine, extension string, args ...string) (err error) {
	for name, machine := range machineMap {
		err = RegisterMachine(name, machine, extension, args...)
		if err != nil {
			return
		}
	}

	return
}
