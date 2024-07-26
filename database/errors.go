package database

import "errors"

var (
	ErrCantCreateDatabase   = errors.New("can't create the database")
	ErrDatabaseNotFound     = errors.New("database not found")
	ErrDatabaseNotOpen      = errors.New("database not opened")
	ErrDatabaseAlreadyOpen  = errors.New("database already opened")
	ErrDatabaseClosed       = errors.New("database closed")
	ErrCantResetMaintenance = errors.New("can't reset maintenance ticker")
	ErrInvalidDuration      = errors.New("invalid duration for maintenance ticker")
	ErrCloseListenersFull   = errors.New("close listeners are at full capacity")
)
