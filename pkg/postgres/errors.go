package postgres

import "errors"

var (
	ErrConnectionFailed = errors.New("database connection failed")
	ErrPingFailed       = errors.New("database ping failed")
)
