//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ezrec/uv3dp"
)

func PrintMachines() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Known machines:")
	fmt.Fprintln(os.Stderr)

	keys := []string{}
	for key := range uv3dp.MachineFormats {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		item := uv3dp.MachineFormats[key]
		size := &item.Machine.Size
		fmt.Fprintf(os.Stderr, "    %-16s %10s %-16s Size: %dx%d, %.3gx%.3g mm,\t", key,
			item.Machine.Vendor, item.Machine.Model, size.X, size.Y, size.Xmm, size.Ymm)
		fmt.Fprintf(os.Stderr, "Format: %s %v\n", item.Extension, strings.Join(item.Args, " "))
	}
}
