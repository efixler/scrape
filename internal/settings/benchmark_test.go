package settings

import (
	"context"
	"fmt"
	"math/rand"
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
		size int
	}{
		{100},
		{1000},
		{10000},
	}

	for _, test := range tests {
		b.Run(fmt.Sprintf("size=%d", test.size), func(b *testing.B) {
			db := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
			if err := db.Open(context.Background()); err != nil {
				b.Fatalf("Error opening database: %v", err)
			}
			b.Cleanup(func() {
				db.Close()
			})
			domains, err := populateTestDB(db, test.size)
			if err != nil {
				b.Fatalf("can't populate test database: %v", err)
			}
			dss := NewDomainSettingsStorage(db)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				dn := domains[rand.Intn(test.size)]
				_, err := dss.Fetch(dn)
				if err != nil {
					b.Fatalf("can't fetch domain %q: %v", dn, err)
				}
			}
		})
	}
}
