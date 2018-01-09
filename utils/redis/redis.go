package redis

import (
	"github.com/garyburd/redigo/redis"
)

type Options struct {
	MaxIdle   int
	MaxActive int
	Address   string
	Password  string
	DB        int
}

func Pool(opt Options) *redis.Pool {
	// p := redis.NewPool(func() (redis.Conn, error) {
	// 	return redis.Dial("tcp", opt.Address, redis.DialPassword(opt.Password), redis.DialDatabase(opt.DB), redis.DialKeepAlive(5*time.Minute), redis.DialConnectTimeout(1*time.Minute))
	// }, opt.MaxIdle)
	p := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", opt.Address, redis.DialPassword(opt.Password), redis.DialDatabase(opt.DB))
		},
		MaxIdle: opt.MaxIdle,
		// MaxActive: 100,
		// IdleTimeout: 240 * time.Second,
		// Wait:        true,
		// TestOnBorrow
	}
	return p
}
