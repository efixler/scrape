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
			FetchClient: resource.ClientIdentifier(rand.Intn(3)),
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

// Results: (saving here to reference on testings some possible improvements)
// BenchmarkLoadDomainSetting
// BenchmarkLoadDomainSetting/sqlite:memory:100
// BenchmarkLoadDomainSetting/sqlite:memory:100-10         	  177280	      5765 ns/op	    2474 B/op	      51 allocs/op
// BenchmarkLoadDomainSetting/sqlite:memory:1000
// BenchmarkLoadDomainSetting/sqlite:memory:1000-10        	  189783	      5909 ns/op	    2502 B/op	      51 allocs/op
// BenchmarkLoadDomainSetting/sqlite:memory:10000
// BenchmarkLoadDomainSetting/sqlite:memory:10000-10       	  195798	      6128 ns/op	    2503 B/op	      51 allocs/op
// BenchmarkLoadDomainSetting/sqlite:tmpfile:100
// BenchmarkLoadDomainSetting/sqlite:tmpfile:100-10        	  165363	      6763 ns/op	    2504 B/op	      51 allocs/op
// BenchmarkLoadDomainSetting/sqlite:tmpfile:1000
// BenchmarkLoadDomainSetting/sqlite:tmpfile:1000-10       	  169336	      6967 ns/op	    2507 B/op	      51 allocs/op
// BenchmarkLoadDomainSetting/sqlite:tmpfile:10000
// BenchmarkLoadDomainSetting/sqlite:tmpfile:10000-10      	  165699	      7159 ns/op	    2502 B/op	      51 allocs/op

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

func benchmarkLoadDomainSetting(b *testing.B, dss *domainSettingsStorage, domains []string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dn := domains[rand.Intn(len(domains))]
		_, err := dss.Fetch(dn)
		if err != nil {
			b.Fatalf("can't fetch domain %q: %v", dn, err)
		}
	}
}

// Results: (saving here to reference on testings some possible improvements)
// BenchmarkLoadDomainBatch
// BenchmarkLoadDomainBatch/sqlite:memory:100
// BenchmarkLoadDomainBatch/sqlite:memory:100-10         	    3529	    334731 ns/op	  180754 B/op	    2426 allocs/op
// BenchmarkLoadDomainBatch/sqlite:memory:500
// BenchmarkLoadDomainBatch/sqlite:memory:500-10         	     712	   1691428 ns/op	  898205 B/op	   12019 allocs/op
// BenchmarkLoadDomainBatch/sqlite:memory:1000
// BenchmarkLoadDomainBatch/sqlite:memory:1000-10        	     343	   3455747 ns/op	 1781367 B/op	   24010 allocs/op
// BenchmarkLoadDomainBatch/sqlite:memory:10000
// BenchmarkLoadDomainBatch/sqlite:memory:10000-10       	      32	  38905829 ns/op	17881840 B/op	  239885 allocs/op
// BenchmarkLoadDomainBatch/sqlite:tmpfile:100
// BenchmarkLoadDomainBatch/sqlite:tmpfile:100-10        	    3344	    332339 ns/op	  177914 B/op	    2428 allocs/op
// BenchmarkLoadDomainBatch/sqlite:tmpfile:500
// BenchmarkLoadDomainBatch/sqlite:tmpfile:500-10        	     712	   1712945 ns/op	  899061 B/op	   12020 allocs/op
// BenchmarkLoadDomainBatch/sqlite:tmpfile:1000
// BenchmarkLoadDomainBatch/sqlite:tmpfile:1000-10       	     346	   3436954 ns/op	 1782174 B/op	   24013 allocs/op
// BenchmarkLoadDomainBatch/sqlite:tmpfile:10000
// BenchmarkLoadDomainBatch/sqlite:tmpfile:10000-10      	      32	  37236333 ns/op	17836775 B/op	  239868 allocs/op

func BenchmarkLoadDomainBatch(b *testing.B) {
	var tests = []struct {
		name string
		size int
		dbf  func(*testing.B) *database.DBHandle
	}{
		{"sqlite:memory:", 100, sqliteInMemoryDB},
		{"sqlite:memory:", 500, sqliteInMemoryDB},
		{"sqlite:memory:", 1000, sqliteInMemoryDB},
		{"sqlite:memory:", 10000, sqliteInMemoryDB},
		{"sqlite:tmpfile:", 100, sqliteFileDB},
		{"sqlite:tmpfile:", 500, sqliteFileDB},
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
			tcount := len(domains)
			domains = nil
			dss := NewDomainSettingsStorage(db)
			if dss.maxBatchSize < tcount {
				dss.maxBatchSize = tcount
			}
			benchmarkLoadDomainBatch(b, dss, tcount)
		})
	}
}

func benchmarkLoadDomainBatch(b *testing.B, dss *domainSettingsStorage, count int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := dss.FetchRange(0, count, "")
		if err != nil {
			b.Fatalf("can't fetch domains: %v", err)
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
