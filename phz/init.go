//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package ctb handle input and output of Chitubox DLP/LCD printables
package phz

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewPhzFormatter(suffix) }

	uv3dp.RegisterFormatter(".phz", newFormatter)
}
