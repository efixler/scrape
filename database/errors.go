package database

import "errors"

var (
	ErrCantCreateDatabase = errors.New("can't create the database")
	ErrDatabaseNotFound   = errors.New("database not found")
)
