package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"
)

type Options struct {
	Address      string
	Password     string
	DB           int
	MaxIdle      int
	IdleTimeout  time.Duration // 240 * time.Second
	MaxActive    int
	Wait         bool // pool中连接已达MaxActive且没有空闲连接则等待, 否则返回连接已耗尽ErrPoolExhausted
	TestOnBorrow func(c redis.Conn, t time.Time) error
}

func Pool(opt Options) *redis.Pool {
	// p := redis.NewPool(func() (redis.Conn, error) {
	// 	return redis.Dial("tcp", opt.Address, redis.DialPassword(opt.Password), redis.DialDatabase(opt.DB), redis.DialKeepAlive(5*time.Minute), redis.DialConnectTimeout(1*time.Minute))
	// }, opt.MaxIdle)
	p := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", opt.Address, redis.DialPassword(opt.Password), redis.DialDatabase(opt.DB))
		},
		MaxIdle:      opt.MaxIdle,
		MaxActive:    opt.MaxActive,
		IdleTimeout:  opt.IdleTimeout,
		Wait:         opt.Wait,
		TestOnBorrow: opt.TestOnBorrow,
	}
	return p
}
