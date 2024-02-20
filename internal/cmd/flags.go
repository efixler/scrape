package cmd

import (
	"errors"
	"flag"
	"regexp"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/store"
	"github.com/efixler/scrape/store/mysql"
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

type DatabaseFlags struct {
	database *envflags.Value[DatabaseSpec]
	username *envflags.Value[string]
	password *envflags.Value[string]
}

func AddDatabaseFlags(baseEnv string, flags *flag.FlagSet) *DatabaseFlags {
	dbFlags := &DatabaseFlags{
		database: NewDatabaseValue(baseEnv, DefaultDatabase),
		username: envflags.NewString(baseEnv+"_USER", ""),
		password: envflags.NewString(baseEnv+"_PASSWORD", ""),
	}
	dbFlags.database.AddTo(flags, "database", "Database type:path")
	dbFlags.username.AddTo(flags, "db-user", "Database user")
	dbFlags.password.AddTo(flags, "db-password", "Database password")
	return dbFlags
}

func (f DatabaseFlags) String() DatabaseSpec {
	return f.database.Get()
}

func (f DatabaseFlags) Database() (store.Factory, error) {
	return Database(f.database.Get(), f.username.Get(), f.password.Get())
}

func Database(spec DatabaseSpec, username string, password string) (store.Factory, error) {
	switch spec.Type {
	case "sqlite3":
		fallthrough
	case "sqlite":
		return sqlite.Factory(sqlite.File(spec.Path)), nil
	case "mysql":
		return mysql.Factory(mysql.NetAddress(spec.Path), mysql.Username(username), mysql.Password(password)), nil
	default:
		return nil, errors.New("no implementation for " + spec.Type)
	}
}
