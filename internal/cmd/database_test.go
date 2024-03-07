package cmd

import (
	"os"
	"testing"
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
