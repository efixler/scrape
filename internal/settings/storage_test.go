package settings

import (
	"context"
	"fmt"
	"log/slog"
	"net/textproto"
	"sort"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/ua"
	"github.com/pressly/goose/v3"
)

func getDatabase(t *testing.T) *database.DBHandle {
	engine := testEngine()
	db := database.New(engine)
	if err := db.Open(context.TODO()); err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	if err := db.MigrateUp(); err != nil {
		t.Fatalf("Error migrating database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.MigrateReset(); err != nil {
			t.Errorf("Error resetting test db: %v", err)
		}
		db.Close()
	})
	return db
}

func TestStoreAndRetrieve(t *testing.T) {
	db := getDatabase(t)
	dss := NewDomainSettingsStorage(db)

	tests := []struct {
		name      string
		settings  DomainSettings
		expectErr bool
	}{
		{
			name:      "empty",
			settings:  DomainSettings{},
			expectErr: true,
		},
		{
			name: "populated",
			settings: DomainSettings{
				Domain:      "example.com",
				Sitename:    "example",
				FetchClient: resource.DefaultClient,
				UserAgent:   ua.UserAgent("Mozilla/5.0"),
				Headers:     MIMEHeader{"x-special": "special"},
			},
		},
		{
			name: "nil headers",
			settings: DomainSettings{
				Domain:      "example.com",
				Sitename:    "example",
				FetchClient: resource.DefaultClient,
				UserAgent:   ua.UserAgent("Mozilla/5.0"),
				Headers:     nil,
			},
		},
		{
			name: "empty headers",
			settings: DomainSettings{
				Domain:      "example.com",
				Sitename:    "example",
				FetchClient: resource.DefaultClient,
				UserAgent:   ua.UserAgent("Mozilla/5.0"),
				Headers:     MIMEHeader{},
			},
		},
	}
	for _, test := range tests {
		if err := dss.Save(&test.settings); (err != nil) != test.expectErr {
			if !test.expectErr {
				t.Fatalf("%s: can't save: %v", test.name, err)
			}
			continue
		} else if err != nil {
			continue
		}
		ds, err := dss.Fetch(test.settings.Domain)
		if (err != nil) != test.expectErr {
			if test.expectErr {
				t.Errorf("%s: expected error on fetch, got none", test.name)
			} else {
				t.Errorf("%s: unexpected error on fetch: %v", test.name, err)
			}
			continue
		}
		if err != nil {
			continue
		}
		if ds.Sitename != test.settings.Sitename {
			t.Errorf("%s: Sitename: got %q, want %q", test.name, ds.Sitename, test.settings.Sitename)
		}
		if ds.FetchClient != test.settings.FetchClient {
			t.Errorf("%s: FetchClient: got %v, want %v", test.name, ds.FetchClient, test.settings.FetchClient)
		}
		if ds.UserAgent != test.settings.UserAgent {
			t.Errorf("%s: UserAgent: got %v, want %v", test.name, ds.UserAgent, test.settings.UserAgent)
		}
		if len(ds.Headers) != len(test.settings.Headers) {
			t.Errorf("%s: Headers: got %v, want %v", test.name, ds.Headers, test.settings.Headers)
			continue
		}
		for k := range test.settings.Headers {
			if test.settings.Headers[textproto.CanonicalMIMEHeaderKey(k)] != ds.Headers[k] {
				t.Errorf(
					"%s: Headers[%q]: got %q, want %q",
					test.name,
					k,
					ds.Headers[k],
					test.settings.Headers[k],
				)
			}
		}
	}
}

func TestFetchRange(t *testing.T) {
	db := getDatabase(t)
	dss := NewDomainSettingsStorage(db)

	domains, err := populateTestDB(db, 100)
	if err != nil {
		t.Fatalf("can't populate test database: %v", err)
	}
	sort.Strings(domains)
	limit := 10
	for i := 0; i < len(domains); i += limit {
		ds, err := dss.FetchRange(i, limit, "")
		if err != nil {
			t.Fatalf("can't fetch range: %v", err)
		}
		for j := i; j < 10; j++ {
			if ds[j].Domain != domains[j] {
				t.Errorf("expected %q, got %q", domains[j], ds[j].Domain)
			}
		}
	}
	// now check a set that's smaller than limit
	domains = domains[len(domains)-5:]
	ds, err := dss.FetchRange(95, limit, "")
	if err != nil {
		t.Fatalf("can't fetch range: %v", err)
	}
	if len(ds) != len(domains) {
		t.Fatalf("expected %d domains, got %d", len(domains), len(ds))
	}
	for i := range ds {
		if ds[i].Domain != domains[i] {
			t.Errorf("expected %q, got %q", domains[i], ds[i].Domain)
		}
	}
}

func TestDelete(t *testing.T) {
	db := getDatabase(t)

	domains, err := populateTestDB(db, 1)
	if err != nil {
		t.Fatalf("can't populate test database: %v", err)
	}
	dss := NewDomainSettingsStorage(db)

	if deleted, err := dss.Delete(domains[0]); err != nil {
		t.Fatalf("can't delete domain: %v", err)
	} else if !deleted {
		t.Errorf("expected domain %v to be deleted", domains[0])
	}

	if _, err = dss.Fetch(domains[0]); err != storage.ErrResourceNotFound {
		t.Errorf("expected domain %v to be deleted, it wasn't", domains[0])
	}

	if deleted, err := dss.Delete(domains[0]); err != nil {
		t.Fatalf("can't delete domain: %v", err)
	} else if deleted {
		t.Errorf("expected domain %v to already be deleted", domains[0])
	}
}

func TestFetchRangeWithQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectCount int
	}{
		{"empty", "", 260},
		{"*", "*", 260},
		{"a*", "a*", 10},
		{"-1", "-1", 26},
		{"c-1*", "c-1*", 1},
		{"*.com", "*.com", 260},
	}

	db := getDatabase(t)
	dss := NewDomainSettingsStorage(db)
	runes := []rune("abcdefghijklmnopqrstuvwxyz")
	for rune := range runes {
		for i := 0; i < 10; i++ {
			domain := fmt.Sprintf("%c-%d.com", runes[rune], i)
			ds := &DomainSettings{
				Domain:      domain,
				Sitename:    "example",
				FetchClient: resource.DefaultClient,
				UserAgent:   ua.UserAgent("Mozilla/5.0"),
				Headers:     MIMEHeader{"x-special": "special"},
			}
			if err := dss.Save(ds); err != nil {
				t.Fatalf("can't save domain: %v", err)
			}
		}
	}

	for _, test := range tests {
		ds, err := dss.FetchRange(0, 1000, test.query)
		if err != nil {
			t.Fatalf("[%s]: can't fetch range: %v", test.name, err)
		}
		if len(ds) != test.expectCount {
			t.Errorf("[%s]: expected %d domains, got %d", test.name, test.expectCount, len(ds))
		}
	}
}

func init() {
	goose.SetLogger(goose.NopLogger())
	slog.SetLogLoggerLevel(slog.LevelWarn)
}
