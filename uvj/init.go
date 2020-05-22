//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package uvj handles input and output of UV3DP 'generic' zip files (JSON slice description and images)
package uvj

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) uv3dp.Formatter { return NewUVJFormatter(suffix) }

	uv3dp.RegisterFormatter(".uvj", newFormatter)
}
