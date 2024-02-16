package cmd

import (
	"errors"
	"regexp"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/sqlite"
)

var (
	DefaultDatabase        = DatabaseSpec{Type: "sqlite", Path: "scrape_data/scrape.db"}
	ErrDatabaseFormat      = errors.New("database spec must be in the format <type>:<path spec>")
	ErrUnsupportedDatabase = errors.New("unsupported database type")
	dsnRegex               = regexp.MustCompile(`^(\w{5,10}):(.+)`)
)

type DatabaseSpec struct {
	Type string
	Path string
}

func (d DatabaseSpec) String() string {
	return d.Type + ":" + d.Path
}

// Format: <type>:<path spec>
// Example: sqlite:scrape_data/scrape.db
// Example: sqlite::memory:
func NewDatabaseSpec(s string) (DatabaseSpec, error) {
	matches := dsnRegex.FindStringSubmatch(s)
	if matches == nil {
		return DatabaseSpec{}, ErrDatabaseFormat
	}
	spec := DatabaseSpec{
		Type: matches[1],
		Path: matches[2],
	}
	return spec, nil
}

func NewDatabaseValue(env string, def DatabaseSpec) *envflags.Value[DatabaseSpec] {
	converter := NewDatabaseSpec
	val := envflags.NewEnvFlagValue(env, def, converter)
	return val
}

func Database(spec DatabaseSpec) (store.Factory, error) {
	switch spec.Type {
	case "sqlite3":
		fallthrough
	case "sqlite":
		return sqlite.Factory(sqlite.WithFile(spec.Path)), nil
	case "mysql":
		return nil, errors.New("no implementation for mysql")
	default:
		return nil, errors.New("no implementation for " + spec.Type)
	}
}
