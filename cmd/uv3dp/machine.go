//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"os"
	"sort"
)

type MachineSize struct {
	X, Y     int
	Xmm, Ymm float32
}

type MachineInfo struct {
	Size MachineSize
}

// Predefined bed layouts
var (
	MachineMap = map[string]MachineInfo{
		"Anycubic-Photon":  {Size: MachineSize{1440, 2560, 68.04, 120.96}},
		"Elogoo-Mars":      {Size: MachineSize{1440, 2560, 68.04, 120.96}},
		"EPAX-X1":          {Size: MachineSize{1440, 2560, 68.04, 120.96}},
		"EPAX-X9":          {Size: MachineSize{1600, 2560, 120.0, 192.0}},
		"EPAX-X10":         {Size: MachineSize{1600, 2560, 135.0, 216.0}},
		"EPAX-X133":        {Size: MachineSize{2160, 3840, 165.0, 293.0}},
		"EPAX-X156":        {Size: MachineSize{2160, 3840, 194.0, 345.0}},
		"Zortrax-Inkspire": {Size: MachineSize{1440, 2560, 72.0, 128.0}},
	}
)

func PrintMachines() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Known machines:")
	fmt.Fprintln(os.Stderr)

	keys := []string{}
	for key := range MachineMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		item := MachineMap[key]
		size := &item.Size
		fmt.Fprintf(os.Stderr, "    %-20s %dx%d, %.3gx%.3g mm\n", key, size.X, size.Y, size.Xmm, size.Ymm)
	}
}
