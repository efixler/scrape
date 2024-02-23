package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInMemoryOption(t *testing.T) {
	t.Parallel()
	imopt := InMemoryDB()
	c := &config{}
	err := imopt(c)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if c.filename != InMemoryDBName {
		t.Errorf("Unexpected filename: %s", c.filename)
	}
	// Not testing every single config parameter here, focusing on
	// the ones that are most likely to be not be changed for an InMemoryDB
	if c.accessMode != AccessModeMemory {
		t.Errorf("Unexpected accessMode: %s", c.accessMode)
	}
	if c.journalMode != JournalModeOff {
		t.Errorf("Unexpected journalMode: %s", c.journalMode)
	}
	if c.synchronous != SyncNormal {
		t.Errorf("Unexpected synchronous: %s", c.synchronous)
	}
}

func TestDefaultsOption(t *testing.T) {
	t.Parallel()
	dopt := Defaults()
	c := &config{}
	err := dopt(c)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if c.accessMode != AccessModeRWC {
		t.Errorf("Unexpected accessMode: %s", c.accessMode)
	}
	if c.journalMode != JournalModeWAL {
		t.Errorf("Unexpected journalMode: %s", c.journalMode)
	}
	if c.synchronous != SyncOff {
		t.Errorf("Unexpected synchronous: %s", c.synchronous)
	}
}

func TestWithFileOption(t *testing.T) {
	t.Parallel()
	type data struct {
		name             string
		filename         string
		expectedFilename string
		expectErr        bool
	}
	// We expect filename to be resolved to an absolute path

	cwd, _ := os.Getwd()
	tests := []data{
		{"empty", "", filepath.Join(cwd, DefaultDatabase), false},
		{"in memory", InMemoryDBName, InMemoryDBName, false},
		{"relative", "foo.db", filepath.Join(cwd, "foo.db"), false},
		{"absolute", "/tmp/foo.db", "/tmp/foo.db", false},
	}
	for _, test := range tests {
		dir := filepath.Dir(test.expectedFilename)
		_, err := os.Stat(dir) // :memory: gets resolved to .
		cleanUp := os.IsNotExist(err)

		c := &config{}
		wopt := File(test.filename)
		err = wopt(c)
		if err != nil {
			if !test.expectErr {
				t.Fatalf("%s: unexpected error: %s", test.name, err)
			}
			continue
		}
		if c.filename != test.expectedFilename {
			t.Fatalf("%s: unexpected filename: %s", test.name, c.filename)
		}
		if cleanUp {
			file := filepath.Dir(test.expectedFilename)
			if err = os.Remove(file); err != nil {
				t.Fatalf("Error removing directory: %s", err)
			}
		}
	}
}
