package settings

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/ua"
)

func TestJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name              string
		data              string
		expectErr         bool
		expectSitename    string
		expectFetchClient resource.ClientIdentifier
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
		// NB: Unmarshaling doesn't change the case of the keys
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
		expectFetchClient resource.ClientIdentifier
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
			expectJSON:        `{"domain":"example.com","sitename":"example","fetch_client":"chromium-headless","user_agent":"Mozilla/5.0","headers":{"X-Special":"special"}}`,
			expectSitename:    "example",
			expectFetchClient: resource.HeadlessChromium,
			expectUserAgent:   ua.UserAgent("Mozilla/5.0"),
			expectHeaders:     map[string]string{"X-Special": "special"},
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
		for k := range test.expectHeaders {
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
		err := ValidateDomain(test.domain)
		if test.valid && err != nil {
			t.Errorf("[%s]: domain %q should be valid: %v", test.name, test.domain, err)
		} else if !test.valid && err == nil {
			t.Errorf("[%s]: domain %q should be invalid", test.name, test.domain)
		}
	}
}

func TestParseDomainQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		expect    string
		expectErr bool
	}{
		{"empty", "", "%%", false}, // expected, harmless
		{"basic", "example", "%example%", false},
		{"valid chars", "sub.exa-mple", "%sub.exa-mple%", false},
		{"invalid chars", "x:com", "", true},
		{"leading wildcard", "*wee", "%wee", false},
		{"trailing wildcard", "wee*", "wee%", false},
		{"both wildcards", "*wee*", "%wee%", false},
		{"no wildcards", "wee", "%wee%", false},
	}
	for _, test := range tests {
		q, err := parseDomainQuery(test.query)
		if err != nil {
			if !test.expectErr {
				t.Errorf("[%s]: unexpected error: %v", test.name, err)
			}
			continue
		}
		if q != test.expect {
			t.Errorf("[%s]: expected %q, got %q", test.name, test.expect, q)
		}
	}
}

// We only use the random domain generator for testing but we can still
// just make sure that it's returning valid domains.
func TestRandomDomainGenerator(t *testing.T) {
	for i := 0; i < 10; i++ {
		d := randomDomain()
		if err := ValidateDomain(d); err != nil {
			t.Errorf("Error validating domain: %v", err)
		}
	}
}

var tlds = []string{
	"com", "net", "org", "io", "gov", "edu", "co", "us", "co", "dev",
}

var chars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randomString(l int) string {
	b := make([]rune, l)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func randomDomain() string {
	domLen := rand.Intn(32) + 3
	subLen := rand.Intn(16) + 3
	tld := tlds[rand.Intn(len(tlds))]
	return fmt.Sprintf("%s.%s.%s", randomString(subLen), randomString(domLen), tld)
}
