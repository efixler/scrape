package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/ua"
)

func TestJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name              string
		data              string
		expectErr         bool
		expectSitename    string
		expectFetchClient resource.FetchClient
		expectUserAgent   ua.UserAgent
		expectHeaders     map[string]string
	}{
		{
			name:              "empty",
			data:              `{}`,
			expectErr:         false,
			expectSitename:    "",
			expectFetchClient: resource.Unspecified,
			expectUserAgent:   ua.Zero,
			expectHeaders:     map[string]string{},
		},
		{
			name:              "fully populated",
			data:              `{"sitename":"example","fetch_client":"chromium-headless","user_agent":"Mozilla/5.0","headers":{"x-special":"special"}}`,
			expectErr:         false,
			expectSitename:    "example",
			expectFetchClient: resource.HeadlessChromium,
			expectUserAgent:   ua.UserAgent("Mozilla/5.0"),
			expectHeaders:     map[string]string{"x-special": "special"},
		},
	}
	for _, test := range tests {
		ds := &DomainSettings{}
		if err := json.Unmarshal([]byte(test.data), ds); err != nil {
			if !test.expectErr {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}
		if ds.Sitename != test.expectSitename {
			t.Errorf("%s: Sitename: got %q, want %q", test.name, ds.Sitename, test.expectSitename)
		}
		if ds.FetchClient != test.expectFetchClient {
			t.Errorf("%s: FetchClient: got %v, want %v", test.name, ds.FetchClient, test.expectFetchClient)
		}
		if ds.UserAgent != test.expectUserAgent {
			t.Errorf("%s: UserAgent: got %v, want %v", test.name, ds.UserAgent, test.expectUserAgent)
		}
		if len(ds.Headers) != len(test.expectHeaders) {
			t.Errorf("%s: Headers: got %v, want %v", test.name, ds.Headers, test.expectHeaders)
			continue
		}
		for k := range ds.Headers {
			if test.expectHeaders[k] != ds.Headers[k] {
				t.Errorf(
					"%s: Headers[%q]: got %q, want %q",
					test.name,
					k,
					ds.Headers[k],
					test.expectHeaders[k],
				)
			}
		}
	}
}

func TestJSONMarshal(t *testing.T) {
	tests := []struct {
		name              string
		data              *DomainSettings
		expectErr         bool
		expectJSON        string
		expectSitename    string
		expectFetchClient resource.FetchClient
		expectUserAgent   ua.UserAgent
		expectHeaders     map[string]string
	}{
		{
			name:              "empty",
			data:              &DomainSettings{},
			expectErr:         false,
			expectJSON:        `{}`,
			expectSitename:    "",
			expectFetchClient: resource.Unspecified,
			expectUserAgent:   ua.Zero,
			expectHeaders:     map[string]string{},
		},
		{
			name: "fully populated",
			data: &DomainSettings{
				Domain:      "example.com",
				Sitename:    "example",
				FetchClient: resource.HeadlessChromium,
				UserAgent:   ua.UserAgent("Mozilla/5.0"),
				Headers:     map[string]string{"x-special": "special"},
			},
			expectErr:         false,
			expectJSON:        `{"sitename":"example","fetch_client":"chromium-headless","user_agent":"Mozilla/5.0","headers":{"x-special":"special"}}`,
			expectSitename:    "example",
			expectFetchClient: resource.HeadlessChromium,
			expectUserAgent:   ua.UserAgent("Mozilla/5.0"),
			expectHeaders:     map[string]string{"x-special": "special"},
		},
	}
	for _, test := range tests {
		b, err := json.Marshal(test.data)
		if err != nil {
			if !test.expectErr {
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
			continue
		}
		if string(b) != test.expectJSON {
			t.Errorf("%s: JSON: got %q, want %q", test.name, string(b), test.expectJSON)
		}
		ds := &DomainSettings{}
		if err := json.Unmarshal(b, ds); err != nil {
			t.Errorf("%s: can't unmarshal json: %v", test.name, err)
			continue
		}
		if ds.Sitename != test.expectSitename {
			t.Errorf("%s: Sitename: got %q, want %q", test.name, ds.Sitename, test.expectSitename)
		}
		if ds.FetchClient != test.expectFetchClient {
			t.Errorf("%s: FetchClient: got %v, want %v", test.name, ds.FetchClient, test.expectFetchClient)
		}
		if ds.UserAgent != test.expectUserAgent {
			t.Errorf("%s: UserAgent: got %v, want %v", test.name, ds.UserAgent, test.expectUserAgent)
		}
		if len(ds.Headers) != len(test.expectHeaders) {
			t.Errorf("%s: Headers: got %v, want %v", test.name, ds.Headers, test.expectHeaders)
			continue
		}
		for k := range ds.Headers {
			if test.expectHeaders[k] != ds.Headers[k] {
				t.Errorf(
					"%s: Headers[%q]: got %q, want %q",
					test.name,
					k,
					ds.Headers[k],
					test.expectHeaders[k],
				)
			}
		}
	}
}

func TestStoreAndRetrieve(t *testing.T) {
	engine := sqlite.MustNew(sqlite.InMemoryDB())
	db := database.New(engine)
	if err := db.Open(context.Background()); err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	t.Cleanup(func() {
		t.Logf("Cleaning up test database %v", engine.DSNSource())
		db.Close()
	})
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
				Headers:     map[string]string{"x-special": "special"},
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
		for k := range ds.Headers {
			if test.settings.Headers[k] != ds.Headers[k] {
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

// validateDomain checks that the domain is a valid domain name.
func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"basic", "example.com", true},
		{"subdomain", "sub.example.com", true},
		{"long", "this.is.a.very.long.domain.name.that.is.valid.dev", true},
		{"long, invalid tld", "this.is.a.very.long.domain.name.that.is.inva1id", false},
		{"no tld", "example", false},
		{"has valid dashes", "example-pie.com", true},
		{"invalid dash and end of element", "example-.com", false},
		{"invalid dash at start of element", "www.-example.com", false},
		{"double dash", "example--pie.com", false},
		{"empty element", "example..com", false},
		{"empty domain", "", false},
		{"invalid char", "example!.com", false},
		{"dot at end", "example.com.", false},
		{"numerals", "www3.example.com.", false},
	}
	for _, test := range tests {
		err := validateDomain(test.domain)
		if test.valid && err != nil {
			t.Errorf("[%s]: domain %q should be valid: %v", test.name, test.domain, err)
		} else if !test.valid && err == nil {
			t.Errorf("[%s]: domain %q should be invalid", test.name, test.domain)
		}
	}
}

// We only use the random domain generator for testing but we can still
// just make sure that it's returning valid domains.
func TestRandomDomainGenerator(t *testing.T) {
	for i := 0; i < 10; i++ {
		d := randomDomain()
		if err := validateDomain(d); err != nil {
			t.Errorf("Error validating domain: %v", err)
		}
	}
}

var tlds = []string{
	"com", "net", "org", "io", "gov", "edu", "co", "us", "co", "dev",
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(l int) string {
	b := make([]rune, l)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomDomain() string {
	domLen := rand.Intn(32) + 3
	subLen := rand.Intn(16) + 3
	tld := tlds[rand.Intn(len(tlds))]
	return fmt.Sprintf("%s.%s.%s", randomString(subLen), randomString(domLen), tld)
}
