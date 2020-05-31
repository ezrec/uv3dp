//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package pws handles input and output of Anycubic Photons 2.0 (.pws) printables
package pws

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewFormatter(suffix) }

	uv3dp.RegisterFormatter(".pws", newFormatter)
	uv3dp.RegisterFormatter(".pw0", newFormatter)
}
