// Utilities for interpreting application-specific command line flags.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"regexp"
	"strings"

	"github.com/efixler/envflags"
	db "github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/mysql"
	"github.com/efixler/scrape/database/sqlite"
)

type MigrationCommand string

const (
	Up     MigrationCommand = "up"
	Reset  MigrationCommand = "reset"
	Status MigrationCommand = "status"
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

// An envflags implementationt to support type:path database specs.
func NewDatabaseValue(env string, def DatabaseSpec) *envflags.Value[DatabaseSpec] {
	converter := NewDatabaseSpec
	val := envflags.NewEnvFlagValue(env, def, converter)
	return val
}

type DatabaseFlags struct {
	database         *envflags.Value[DatabaseSpec]
	username         *envflags.Value[string]
	password         *envflags.Value[string]
	MigrationCommand MigrationCommand
}

// Add database flags to a flag set, optionally adding a migration flag.
func AddDatabaseFlags(baseEnv string, flags *flag.FlagSet, migrateFlag bool) *DatabaseFlags {
	dbFlags := &DatabaseFlags{
		database: NewDatabaseValue(baseEnv, DefaultDatabase),
		username: envflags.NewString(baseEnv+"_USER", ""),
		password: envflags.NewString(baseEnv+"_PASSWORD", ""),
	}
	dbFlags.database.AddTo(flags, "database", "Database type:path")
	dbFlags.username.AddTo(flags, "db-user", "Database user")
	dbFlags.password.AddTo(flags, "db-password", "Database password")
	if migrateFlag {
		flags.Func(
			"migrate",
			"Issue a db migration command: up, reset, or status",
			func(input string) error {
				cmd := MigrationCommand(strings.ToLower(input))
				switch cmd {
				case Reset:
					fallthrough
				case Up:
					fallthrough
				case Status:
					dbFlags.MigrationCommand = cmd
					return nil
				default:
					return fmt.Errorf("unsupported migration command: %s", cmd)
				}
			},
		)
	}
	return dbFlags
}

func (f DatabaseFlags) String() DatabaseSpec {
	return f.database.Get()
}

func (f DatabaseFlags) IsMigration() bool {
	return string(f.MigrationCommand) != ""
}

// Get a database handle for the db specified in the flags.
func (f DatabaseFlags) Database() (*db.DBHandle, error) {
	return database(f.database.Get(), f.username.Get(), f.password.Get(), f.MigrationCommand)
}

func (f DatabaseFlags) MustDatabase() *db.DBHandle {
	dbF, err := f.Database()
	if err != nil {
		panic(fmt.Sprintf("error making database factory from flags: %v", err))
	}
	return dbF
}

func database(spec DatabaseSpec, username string, password string, migration MigrationCommand) (*db.DBHandle, error) {
	// TODO: Have the DB implementations handle the connection nuances for migration cases.
	switch spec.Type {
	case "sqlite3":
		fallthrough
	case "sqlite":
		options := []sqlite.Option{sqlite.File(spec.Path)}
		switch migration {
		case MigrationCommand(""):
			// no migration command
		default:
			// on any migration command don't auto-create the schema.
			options = append(options, sqlite.WithoutAutoCreate())
		}
		engine, err := sqlite.New(options...)
		if err != nil {
			return nil, err
		}
		return db.New(engine), nil
	case "mysql":
		options := []mysql.Option{
			mysql.NetAddress(spec.Path),
			mysql.Username(username),
			mysql.Password(password),
		}
		if migration == Up {
			// For MySQL we need special handling only when it's possible
			// that the db doesn't exist yet.
			options = append(options, mysql.ForMigration())
		}
		engine, err := mysql.New(options...)
		if err != nil {
			return nil, err
		}
		return db.New(engine), nil
	default:
		return nil, errors.New("no implementation for " + spec.Type)
	}
}
