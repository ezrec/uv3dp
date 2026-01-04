//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newEmptyFormatter := func(suffix string) uv3dp.Formatter { return NewEmptyFormatter() }

	uv3dp.RegisterFormatter("empty", newEmptyFormatter)
}
