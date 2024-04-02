package cmd

import (
	"flag"
	"os"
	"testing"

	"github.com/efixler/envflags"
)

func TestDatabaseSpec(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue DatabaseSpec
		setValue     string
		want         DatabaseSpec
		expectError  bool
	}{
		{
			name:         "env, no explicit value",
			envValue:     "sqlite:env.db",
			defaultValue: DatabaseSpec{"sqlite", "default.db"},
			setValue:     "_",
			want:         DatabaseSpec{"sqlite", "env.db"},
			expectError:  false,
		},
		{
			name:         "env not set, no explicit value",
			envValue:     "",
			defaultValue: DatabaseSpec{"sqlite", "default.db"},
			setValue:     "_",
			want:         DatabaseSpec{"sqlite", "default.db"},
			expectError:  false,
		},
		{
			name:         "env not set, explicit value",
			envValue:     "",
			defaultValue: DatabaseSpec{"sqlite", "default.db"},
			setValue:     "sqlite:set.db",
			want:         DatabaseSpec{"sqlite", "set.db"},
			expectError:  false,
		},
		{
			name:         "env, explicit value",
			envValue:     "sqlite:scrape_data/scrape.db",
			defaultValue: DatabaseSpec{"sqlite", "default.db"},
			setValue:     "sqlite::memory:",
			want:         DatabaseSpec{"sqlite", ":memory:"},
			expectError:  false,
		},
		{
			name:         "env, empty explicit value",
			envValue:     "sqlite:env.db",
			defaultValue: DatabaseSpec{"sqlite", "default.db"},
			setValue:     "",
			want:         DatabaseSpec{"sqlite", "env.db"},
			expectError:  true,
		},
		{
			name:         "malformed env, no explicit value",
			envValue:     "xyzabc",
			defaultValue: DatabaseSpec{"sqlite", "default.db"},
			setValue:     "_",
			want:         DatabaseSpec{"sqlite", "default.db"},
			expectError:  false,
		},
	}
	envKey := "ENVFLAGS_TEST"
	for _, test := range tests {
		os.Setenv(envKey, test.envValue)
		pflag := NewDatabaseValue(envKey, test.defaultValue)
		if test.setValue != "_" {
			err := pflag.Set(test.setValue)
			if (err != nil) != test.expectError {
				t.Errorf(
					"%s: Set(%s) returned error %v, expected error %t",
					test.name,
					test.setValue,
					err,
					test.expectError,
				)
			}
		}
		got := pflag.Get()
		if got != test.want {
			t.Errorf("%s: Get() = %q, want %q", test.name, got, test.want)
		}
		os.Setenv("ENVFLAGS_TEST", "")
	}
}

func TestFlagInput(t *testing.T) {
	tests := []struct {
		name          string
		envFlagPrefix string
		args          []string
		env           []string
		expected      []string
	}{
		{
			name:          "mysql all args",
			envFlagPrefix: "SCRAPE_",
			args:          []string{"--database=mysql:127.0.0.1:3306", "--db-user", "scrape", "--db-password", "password"},
			env:           []string{},
			expected:      []string{"mysql", "127.0.0.1:3306", "scrape", "password"},
		},
		{
			name:          "mysql all args, localhost",
			envFlagPrefix: "SCRAPE_",
			args:          []string{"--database=mysql:localhost:3306", "--db-user", "scrape", "--db-password", "password"},
			env:           []string{},
			expected:      []string{"mysql", "localhost:3306", "scrape", "password"},
		},
		{
			name:          "mysql all env",
			envFlagPrefix: "SCRAPE_",
			args:          []string{},
			env:           []string{"SCRAPE_DB", "mysql:localhost:3306", "SCRAPE_DB_USER", "scrape", "SCRAPE_DB_PASSWORD", "password"},
			expected:      []string{"mysql", "localhost:3306", "scrape", "password"},
		},
	}
	for _, test := range tests {
		envflags.EnvPrefix = test.envFlagPrefix
		envMap := make(map[string]string)
		for i := 0; i < len(test.env); i += 2 {
			envMap[test.env[i]] = test.env[i+1]
			os.Setenv(test.env[i], test.env[i+1])
		}
		flags := flag.NewFlagSet(test.name, flag.ContinueOnError)
		dbFlags := AddDatabaseFlags("DB", flags, false)
		flags.Parse(test.args)
		spec := dbFlags.database.Get()
		if spec.Type != test.expected[0] {
			t.Errorf("[%s]: Type = %v, want %v", test.name, spec.Type, test.expected[0])
		}
		if spec.Path != test.expected[1] {
			t.Errorf("[%s]: Path = %v, want %v", test.name, spec.Path, test.expected[1])
		}
		if dbFlags.username.Get() != test.expected[2] {
			t.Errorf("[%s]: Username = %v, want %v", test.name, dbFlags.username.Get(), test.expected[2])
		}
		if dbFlags.password.Get() != test.expected[3] {
			t.Errorf("[%s]: Password = %v, want %v", test.name, dbFlags.password.Get(), test.expected[3])
		}
		for k := range envMap {
			os.Unsetenv(k)
		}
	}
}
