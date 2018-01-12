package mysql

import (
	"database/sql"
	"time"
)

type Options struct {
	// dsn: user:pwd@tcp(address)/dbname?charset=utf8
	DSN         string
	UserName    string
	Password    string
	Address     string
	DBName      string
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
}

// NewPool creates db pool
func NewPool(opt Options) (*sql.DB, error) {
	var dsn string
	if opt.DSN != "" {
		dsn = opt.DSN
	} else {
		dsn = opt.UserName + ":" + opt.Password + "@tcp(" + opt.Address + ")/" + opt.DBName + "?collation=utf8mb4_unicode_ci"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(opt.MaxIdle)
	db.SetMaxOpenConns(opt.MaxOpen)
	db.SetConnMaxLifetime(opt.MaxLifetime)
	return db, err
}
