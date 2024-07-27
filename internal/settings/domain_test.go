package settings

import (
	"context"
	"encoding/json"
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
