//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package main

import (
	"math/rand"
	"time"

	"github.com/ezrec/uv3dp"
)

func init() {
	// Initialize the rand package
	rand.Seed(time.Now().UnixNano())

	newEmptyFormatter := func(suffix string) uv3dp.Formatter { return NewEmptyFormatter() }

	uv3dp.RegisterFormatter("empty", newEmptyFormatter)
}
