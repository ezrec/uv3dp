//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

// Package cbddlp handle input and output of Chitubox DLP/LCD printables
package cbddlp

import (
	"github.com/ezrec/uv3dp"
)

func init() {
	newFormatter := func(suffix string) (format uv3dp.Formatter) { return NewCbddlpFormatter(suffix) }

	uv3dp.RegisterFormatter(".cbddlp", newFormatter)
	uv3dp.RegisterFormatter(".photon", newFormatter)
}
