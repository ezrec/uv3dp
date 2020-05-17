//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package cws handles input and output of NOVA32 CWS printables
package cws

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewCWSFormatter(suffix) }

	uv3dp.RegisterFormatter(".cws", newFormatter)
}
