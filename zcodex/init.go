//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package zcodex handles input and output of Prusa SL1 DLP/LCD printables
package zcodex

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewZcodexFormatter(suffix) }

	uv3dp.RegisterFormatter(".zcodex", newFormatter)
}
