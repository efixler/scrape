package database

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pressly/goose/v3"
)

const (
	MigrationDir = "migrations" // Default directory for migration files (in the embed.FS)
)

var (
	ErrNoMigrationFS = fmt.Errorf("migration filesystem is not set")
)

// Execute an up migration using goose.
// The environment variables are set before running the migration.
// This may be used in conjunction with Goose's EnvSubOn directive.
func (d *DBHandle) DoMigrateUp(env ...string) error {
	if clearF, err := d.prepareForMigration(env...); err != nil {
		return err
	} else {
		defer clearF()
	}
	if err := goose.Up(d.DB, MigrationDir); err != nil {
		return err
	}
	return nil
}

func (d DBHandle) DoMigrateReset(env ...string) error {
	if clearF, err := d.prepareForMigration(env...); err != nil {
		return err
	} else {
		defer clearF()
	}
	return goose.Reset(d.DB, MigrationDir)
}

func (d DBHandle) prepareForMigration(env ...string) (func(), error) {
	if d.engine.MigrationFS() == nil {
		return nil, ErrNoMigrationFS
	}

	if err := goose.SetDialect(string(d.engine.Driver())); err != nil {
		return nil, err
	}
	if (len(env) % 2) != 0 {
		return nil, fmt.Errorf("environment variables must be key-value pairs")
	}
	goose.SetBaseFS(*d.engine.MigrationFS())

	envRestore := make(map[string]string, len(env)/2)
	clearF := func() {
		for k, v := range envRestore {
			switch v {
			case "":
				os.Unsetenv(k)
			default:
				os.Setenv(k, v)
			}
		}
	}
	for i := 0; i < len(env); i += 2 {
		envRestore[env[i]] = os.Getenv(env[i])
		if err := os.Setenv(env[i], env[i+1]); err != nil {
			delete(envRestore, env[i])
			return nil, err
		}
	}
	return clearF, nil
}

func (d DBHandle) PrintMigrationStatus(migrationFS fs.FS, migrationDir string) error {
	if err := goose.SetDialect(string(d.engine.Driver())); err != nil {
		return err
	}
	if d.engine.MigrationFS() == nil {
		return ErrNoMigrationFS
	}
	goose.SetBaseFS(*d.engine.MigrationFS())
	if err := goose.Status(d.DB, MigrationDir); err != nil {
		return err
	}
	return nil
}
