package database

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"

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
func (d *DBHandle) MigrateUp(env ...string) error {
	clearF, err := d.prepareForMigration(env...)
	if err != nil {
		return err
	}
	defer clearF()
	if bm, ok := d.Engine.(BeforeMigrateUpHook); ok {
		err := bm.BeforeMigrateUp(d)
		if err != nil {
			return err
		}
	}
	return goose.Up(d.DB, ".")

}

func (d DBHandle) MigrateReset(env ...string) error {
	clearF, err := d.prepareForMigration(env...)
	if err != nil {
		return err
	}
	defer clearF()
	return goose.Reset(d.DB, ".")
}

func (d DBHandle) prepareForMigration(env ...string) (func(), error) {
	if d.Engine.MigrationFS() == nil {
		return nil, ErrNoMigrationFS
	}

	if err := goose.SetDialect(string(d.Engine.Driver())); err != nil {
		return nil, err
	}
	if mes, ok := d.Engine.(MigrationEnvSetter); ok {
		env = append(mes.MigrationEnv(), env...)
	}

	if (len(env) % 2) != 0 {
		return nil, fmt.Errorf("environment variables must be key-value pairs")
	}
	migFS, _, err := extractMigrationFS(d.Engine.MigrationFS())
	if err != nil {
		return nil, err
	}

	goose.SetBaseFS(migFS)

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

func (d DBHandle) PrintMigrationStatus() error {
	if err := goose.SetDialect(string(d.Engine.Driver())); err != nil {
		return err
	}
	migFS, dirName, err := extractMigrationFS(d.Engine.MigrationFS())
	if err != nil {
		return err
	}
	goose.SetBaseFS(migFS)
	if err := goose.Status(d.DB, dirName); err != nil {
		return err
	}
	return nil
}

func extractMigrationFS(migrationFS *embed.FS) (fs.FS, string, error) {
	if migrationFS == nil {
		return nil, "", ErrNoMigrationFS
	}
	return findMigrationsDir(migrationFS, ".")
}

func findMigrationsDir(fsys fs.FS, currentPath string) (fs.FS, string, error) {
	entries, err := fs.ReadDir(fsys, currentPath)
	if err != nil {
		return nil, "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() == MigrationDir {
				fullPath := path.Join(currentPath, entry.Name())
				subFS, err := fs.Sub(fsys, fullPath)
				if err != nil {
					return nil, "", err
				}
				return subFS, fullPath, nil
			}
			// Recursively search in subdirectories
			foundFS, foundPath, err := findMigrationsDir(fsys, path.Join(currentPath, entry.Name()))
			if err == nil && foundFS != nil {
				return foundFS, foundPath, nil
			}
		}
	}

	return nil, "", ErrNoMigrationFS
}
