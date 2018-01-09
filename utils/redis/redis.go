package redis

import (
	"github.com/garyburd/redigo/redis"
)

type Options struct {
	MaxIdle  int
	Address  string
	Password string
	DB       int
}

func Pool(opt Options) *redis.Pool {
	p := redis.NewPool(func() (redis.Conn, error) {
		return redis.Dial("tcp", opt.Address, redis.DialPassword(opt.Password), redis.DialDatabase(opt.DB))
	}, opt.MaxIdle)
	return p
}
