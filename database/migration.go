package database

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pressly/goose/v3"
)

// Execute an up migration using goose. migrationDir is the path within
// migrationFS where the migration files are stored.
// The environment variables are set before running the migration.
// This may be used in conjunction with Goose's EnvSubOn directive.
func (d DBHandle[T]) DoMigrateUp(
	migrationFS fs.FS,
	migrationDir string,
	env ...string,
) error {
	if err := goose.SetDialect(string(d.Driver)); err != nil {
		return err
	}
	if (len(env) % 2) != 0 {
		return fmt.Errorf("environment variables must be key-value pairs")
	}
	goose.SetBaseFS(migrationFS)

	envRestore := make(map[string]string, len(env)/2)
	defer func() {
		for k, v := range envRestore {
			switch v {
			case "":
				os.Unsetenv(k)
			default:
				os.Setenv(k, v)
			}
		}
	}()

	for i := 0; i < len(env); i += 2 {
		envRestore[env[i]] = os.Getenv(env[i])
		if err := os.Setenv(env[i], env[i+1]); err != nil {
			delete(envRestore, env[i])
			return err
		}
	}
	if err := goose.Up(d.DB, migrationDir); err != nil {
		return err
	}
	return nil
}

func (d DBHandle[T]) PrintMigrationStatus(migrationFS fs.FS, migrationDir string) error {
	if err := goose.SetDialect(string(d.Driver)); err != nil {
		return err
	}
	goose.SetBaseFS(migrationFS)
	if err := goose.Status(d.DB, migrationDir); err != nil {
		return err
	}
	return nil
}
