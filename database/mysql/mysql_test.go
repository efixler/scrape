package mysql

import (
	"errors"
	"testing"
)

func TestOptionError(t *testing.T) {
	errOpt := func(c *Config) error {
		return errors.New("test error")
	}
	_, err := New(errOpt)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestMigrationEnvContainsTargetSchema(t *testing.T) {
	e := MustNew()
	env := e.MigrationEnv()
	for k, v := range env {
		if v == "TargetSchema" {
			if len(env) < k+1 {
				t.Fatalf("TargetSchema key found in env without value")
			}
			if env[k+1] == "" {
				t.Fatalf("TargetSchema value is empty")
			}
			return
		}
	}
	t.Errorf("TargetSchema not found in env")
}
