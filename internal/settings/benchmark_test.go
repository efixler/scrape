package settings

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/ua"
)

func populateTestDB(dbh *database.DBHandle, count int) ([]string, error) {
	dss := NewDomainSettingsStorage(dbh)
	domains := make([]string, count)
	for i := 0; i < count; i++ {
		d := randomDomain()
		domains[i] = d
		ds := &DomainSettings{
			Domain:      d,
			Sitename:    randomString(32),
			FetchClient: resource.FetchClient(rand.Intn(3)),
			UserAgent:   ua.UserAgent(randomString(64)),
			Headers: map[string]string{
				"x-token":      randomString(rand.Intn(128) + 127),
				"x-identifier": randomString(rand.Intn(64 + 63)),
			},
		}
		if err := dss.Save(ds); err != nil {
			return nil, err
		}
	}
	return domains, nil
}

func BenchmarkLoadDomainSetting(b *testing.B) {
	var tests = []struct {
		name string
		size int
		dbf  func(*testing.B) *database.DBHandle
	}{
		{"sqlite:memory:", 100, sqliteInMemoryDB},
		{"sqlite:memory:", 1000, sqliteInMemoryDB},
		{"sqlite:memory:", 10000, sqliteInMemoryDB},
		{"sqlite:tmpfile:", 100, sqliteFileDB},
		{"sqlite:tmpfile:", 1000, sqliteFileDB},
		{"sqlite:tmpfile:", 10000, sqliteFileDB},
	}

	for _, test := range tests {
		b.Run(fmt.Sprintf("%s%d", test.name, test.size), func(b *testing.B) {
			db := test.dbf(b)
			domains, err := populateTestDB(db, test.size)
			if err != nil {
				b.Fatalf("can't populate test database: %v", err)
			}
			benchmarkLoadDomainSetting(b, NewDomainSettingsStorage(db), domains)
		})
	}
}

func benchmarkLoadDomainSetting(b *testing.B, dss *DomainSettingsStorage, domains []string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dn := domains[rand.Intn(len(domains))]
		_, err := dss.Fetch(dn)
		if err != nil {
			b.Fatalf("can't fetch domain %q: %v", dn, err)
		}
	}
}

func sqliteInMemoryDB(b *testing.B) *database.DBHandle {
	db := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
	if err := db.Open(context.Background()); err != nil {
		b.Fatalf("Error opening database: %v", err)
	}
	b.Cleanup(func() {
		db.Close()
	})
	return db
}

func sqliteFileDB(b *testing.B) *database.DBHandle {
	tmpdb := filepath.Join(b.TempDir(), "scrape-domain-settings-test"+randomString(8))
	db := database.New(sqlite.MustNew(sqlite.File(tmpdb)))
	if err := db.Open(context.Background()); err != nil {
		b.Fatalf("Error opening database: %v", err)
	}
	b.Cleanup(func() {
		db.Close()
	})
	return db
}
