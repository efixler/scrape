package cmd

import (
	"errors"
	"flag"
	"fmt"
	"regexp"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/internal/storage/mysql"
	"github.com/efixler/scrape/internal/storage/sqlite"
	"github.com/efixler/scrape/store"
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
	Create   bool
}

func AddDatabaseFlags(baseEnv string, flags *flag.FlagSet, createFlag bool) *DatabaseFlags {
	dbFlags := &DatabaseFlags{
		database: NewDatabaseValue(baseEnv, DefaultDatabase),
		username: envflags.NewString(baseEnv+"_USER", ""),
		password: envflags.NewString(baseEnv+"_PASSWORD", ""),
	}
	dbFlags.database.AddTo(flags, "database", "Database type:path")
	dbFlags.username.AddTo(flags, "db-user", "Database user")
	dbFlags.password.AddTo(flags, "db-password", "Database password")
	if createFlag {
		flags.BoolVar(&dbFlags.Create, "create", false, "Create the database and exit")
	}
	return dbFlags
}

func (f DatabaseFlags) String() DatabaseSpec {
	return f.database.Get()
}

func (f DatabaseFlags) Database() (store.Factory, error) {
	return database(f.database.Get(), f.username.Get(), f.password.Get(), f.Create)
}

func (f DatabaseFlags) MustDatabase() store.Factory {
	dbF, err := f.Database()
	if err != nil {
		panic(fmt.Sprintf("error making database factory from flags: %v", err))
	}
	return dbF
}

func database(spec DatabaseSpec, username string, password string, noSchema bool) (store.Factory, error) {
	switch spec.Type {
	case "sqlite3":
		fallthrough
	case "sqlite":
		return sqlite.Factory(sqlite.File(spec.Path)), nil
	case "mysql":
		options := []mysql.Option{
			mysql.NetAddress(spec.Path),
			mysql.Username(username),
			mysql.Password(password),
		}
		if noSchema {
			options = append(options, mysql.WithoutSchema())
		}
		return mysql.Factory(options...), nil
	default:
		return nil, errors.New("no implementation for " + spec.Type)
	}
}
