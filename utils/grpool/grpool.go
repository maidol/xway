package xgrpool

import (
	"github.com/ivpusic/grpool"
)

func Default() *grpool.Pool {
	xpool := grpool.NewPool(10, 100)
	return xpool
}

func New(numWorkers int, jobQueueLen int) *grpool.Pool {
	return grpool.NewPool(numWorkers, jobQueueLen)
}
