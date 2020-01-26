//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package sl1 handles input and output of Prusa SL1 DLP/LCD printables
package sl1

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewSl1Formatter(suffix) }

	uv3dp.RegisterFormatter(".sl1", newFormatter)
}
