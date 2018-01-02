package xgrpool

import (
	"github.com/ivpusic/grpool"
)

var xpool = grpool.NewPool(100000, 3000000)

func Default() *grpool.Pool {
	if xpool != nil {
		return xpool
	}
	xpool := grpool.NewPool(100000, 3000000)
	return xpool
}
