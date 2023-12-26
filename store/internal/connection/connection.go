package connection

// TODO: This package is somewhat misconceived. And needs to be refactored or eliminated.
// There is only meant to be one *DB instance per DB, which generally meand one per app.
// This was written with the assumption that we'd parallelize fetches with multiple storage
// adapters but that's not really the right approach. We should have a single storage adapter
// per instance.
// Materially there are two things we want to fix or get rid of:
// 1. The open/close semantics and the locking they require, are brittle
// 2. The fact that we're using a map of sharedDBs is a bit of a code smell. We should
//    probably just have a single sharedDB instance per DB, and have the DB handle be
//    a member of the storage adapter struct. That would also make it easier to get rid
//    of the open/close semantics, which are not really necessary.
// 3. Statement handles are kept at the adaptor layer and they should really be kept
//	  wherever the DB handle is, since they pertain to the DB handle.
//  This isn't necessarily super urgent right now since the app code is only making one
//  storage adapter instance, but it's still a bit of a mess.
// Possibly we want to refactor this so that the storage adaprot embeds it, or get rid of in
// entirely and just have the storage adapter keep the DB handle and all the related stuff.

import (
	"database/sql"
	"sync"

	"log/slog"
)

type DriverName string

const (
	SQLite DriverName = "sqlite3"
)

var dbHandles map[string]*sharedDB

type sharedDB struct {
	db     *sql.DB
	driver DriverName
	dsn    string
	// stmts  map[T]*sql.Stmt
	count int
	mutex *sync.Mutex
}

// To be called from Open() e.g. the first time we need a db handle
// in this instance
func (s *sharedDB) acquire() (*sql.DB, error) {
	out := s.increment()
	if (out == 1) || (s.db == nil) {
		var err error
		s.db, err = sql.Open(string(s.driver), s.dsn)
		if err != nil {
			s.decrement()
			return nil, err
		}
	}
	return s.db, nil
}

// Returns the db handle, for methods wanting to use it
// after it's been Open()ed
func (s *sharedDB) get() *sql.DB {
	return s.db
}

// To be called from Close() e.g. when we're done with this db handle
// (and this instance)
// Returns true if we actually closed the db handle, false if someone
// else is still using it
func (s *sharedDB) release() bool {
	out := s.decrement()
	if out == 0 {
		err := s.db.Close()
		if err != nil {
			slog.Error("error closing db handle", "dsn", s.dsn, "error", err)
		}
		s.db = nil
		return true
	} else if out < 0 {
		// This can happen on a double close, not a big deal, but leave a log message
		// here so we can debug
		slog.Debug("db handle count went negative", "dsn", s.dsn, "count", out)
		s.mutex.Lock()
		defer s.mutex.Unlock()
		if s.count < 0 {
			s.count = 0
		}
	}
	return false
}

func (s *sharedDB) increment() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
	return s.count
}

func (s *sharedDB) decrement() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count--
	return s.count
}

func HasOpenedDB(dsn string) bool {
	if dbHandles == nil {
		return false
	}
	_, ok := dbHandles[dsn]
	return ok
}

// To the first time we need a db handle
func OpenDB(driver DriverName, dsn string) (*sql.DB, error) {
	if dbHandles == nil {
		// We usually only have one of these, so let's not allocate
		dbHandles = make(map[string]*sharedDB, 1)
	}
	// First let's see if we have a db handle for this dsn

	dbs, ok := dbHandles[dsn]
	if !ok {
		// lock the map while we create the handle
		var m sync.Mutex
		m.Lock()
		defer m.Unlock()
		if dbs, ok = dbHandles[dsn]; !ok {
			dbs = &sharedDB{dsn: dsn, driver: driver, mutex: &sync.Mutex{}}
			dbHandles[dsn] = dbs
		}
	}
	// Now we know we have a sharedDB entry, and that it's in the map,
	// so let's get the handle
	db, err := dbs.acquire()
	return db, err
}

// If the OpenDB() hasn't been called yet, or if it's been closed,
// this will return nil
func GetDB(dsn string) *sql.DB {
	if dbHandles == nil {
		return nil
	}
	dbs, ok := dbHandles[dsn]
	if !ok {
		return nil
	}
	return dbs.get()
}

func CloseDB(dsn string) bool {
	if dbHandles == nil {
		return false
	}
	dbs, ok := dbHandles[dsn]
	if !ok {
		return false
	}
	if !dbs.release() {
		return false
	}
	// We're done with this db handle, so let's remove it from the map
	var m sync.Mutex
	m.Lock()
	defer m.Unlock()
	delete(dbHandles, dsn)
	return true
}
