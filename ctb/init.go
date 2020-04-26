//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package ctb handle input and output of Chitubox DLP/LCD printables
package ctb

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewCtbFormatter(suffix) }

	uv3dp.RegisterFormatter(".ctb", newFormatter)
}
