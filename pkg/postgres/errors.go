package postgres

import "errors"

var (
	ErrPostgresConnectionFailed = errors.New("database connection failed")
	ErrPostgresPingFailed       = errors.New("database ping failed")
)
