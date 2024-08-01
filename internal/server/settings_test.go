package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/database/sqlite"
	"github.com/efixler/scrape/internal/settings"
	"github.com/pressly/goose/v3"
)

func TestExtractDomainFromPath(t *testing.T) {
	tests := []struct {
		name         string
		domain       string
		expectStatus int
	}{
		{
			name:         "empty",
			domain:       "/foo/bar/",
			expectStatus: 400,
		},
		{
			name:         "invalid domain",
			domain:       "INVALID",
			expectStatus: 400,
		},
		{
			name:         "valid domain",
			domain:       "example.com",
			expectStatus: 200,
		},
	}

	okHandler := func(testname string, domain string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			ds, _ := r.Context().Value(dsKey{}).(*singleDomainRequest)
			if ds.Domain != domain {
				t.Errorf("[%s]: got domain %q, want %q", testname, domain, ds.Domain)
			}
		}
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/foo/bar/{DOMAIN}", nil)
		r.SetPathValue("DOMAIN", tt.domain)
		w := httptest.NewRecorder()
		chain := Chain(okHandler(tt.name, tt.domain), extractDomainFromPath(dsKey{}))
		chain(w, r)
		if w.Code != tt.expectStatus {
			t.Errorf("[%s]: got status %d, want %d", tt.name, w.Code, tt.expectStatus)
		}
	}
}

func TestGetDomainSettings(t *testing.T) {
	tests := []struct {
		name         string
		settings     *settings.DomainSettings
		expectStatus int
	}{
		{
			name:         "no settings for domain",
			expectStatus: 404,
		},
		{
			name:         "settings exists",
			settings:     &settings.DomainSettings{Domain: "example.com", Sitename: "example"},
			expectStatus: 200,
		},
	}

	domainExtractor := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			v := new(singleDomainRequest)
			v.Domain = "example.com"
			r = r.WithContext(context.WithValue(r.Context(), dsKey{}, v))
			next(w, r)
		}
	}

	for _, tt := range tests {
		db := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
		if err := db.Open(context.Background()); err != nil {
			t.Fatalf("[%s] error opening db %v", tt.name, err)
		}
		t.Cleanup(func() {
			db.Close()
		})

		ss := &scrapeServer{
			ctx:             context.Background(),
			settingsStorage: settings.NewDomainSettingsStorage(db),
		}

		if tt.settings != nil {
			tt.settings.Domain = "example.com"
			if err := ss.settingsStorage.Save(tt.settings); err != nil {
				t.Fatalf("[%s] error saving domain settings %v", tt.name, err)
			}
		}

		r := httptest.NewRequest("GET", "/foo/bar/{DOMAIN}", nil)
		w := httptest.NewRecorder()
		chain := Chain(
			ss.getSingleDomainSettings,
			domainExtractor,
		)
		chain(w, r)
		if w.Code != tt.expectStatus {
			t.Errorf("[%s]: got status %d, want %d", tt.name, w.Code, tt.expectStatus)
		}
		if tt.expectStatus != 200 {
			continue
		}
		body := w.Result().Body
		result := new(settings.DomainSettings)
		if err := json.NewDecoder(body).Decode(result); err != nil {
			t.Errorf("[%s]: error decoding response %v", tt.name, err)
			continue
		}
		saved, err := ss.settingsStorage.Fetch("example.com")
		if err != nil {
			t.Errorf("[%s]: error fetching domain settings %v", tt.name, err)
			continue
		}
		if saved.Sitename != result.Sitename {
			t.Errorf("[%s]: got sitename %q, want %q", tt.name, result.Sitename, saved.Sitename)
		}
		if saved.UserAgent != result.UserAgent {
			t.Errorf("[%s]: got user agent %q, want %q", tt.name, result.UserAgent, saved.UserAgent)
		}
		if saved.FetchClient != result.FetchClient {
			t.Errorf("[%s]: got fetch client %q, want %q", tt.name, result.FetchClient, saved.FetchClient)
		}
		if len(saved.Headers) != len(result.Headers) {
			t.Errorf("[%s]: got headers %v, want %v", tt.name, result.Headers, saved.Headers)
		}
		for k, v := range saved.Headers {
			if result.Headers[k] != v {
				t.Errorf("[%s]: got header %q=%q, want %q=%q", tt.name, k, v, k, result.Headers[k])
			}
		}
	}
}

func TestPutDomainSettings(t *testing.T) {
	tests := []struct {
		name         string
		expectStatus int
		payload      string
	}{
		{
			name:         "empty",
			expectStatus: 400,
			payload:      "",
		},
		{
			name:         "invalid json keys",
			expectStatus: 400,
			payload:      `{"foo":"bar"}`,
		},
		{
			name:         "valid json",
			expectStatus: 200,
			payload:      `{"sitename":"example.com","fetch_client":"direct","user_agent":"bar","headers":{}}`,
		},
		{
			name:         "invalid json value",
			expectStatus: 400,
			payload:      `{"sitename":"example.com","fetch_client":"noop","user_agent":"bar","headers":{}}`,
		},
	}

	domainExtractor := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			v := new(singleDomainRequest)
			v.Domain = "example.com"
			r = r.WithContext(context.WithValue(r.Context(), dsKey{}, v))
			next(w, r)
		}
	}

	for _, tt := range tests {
		db := database.New(sqlite.MustNew(sqlite.InMemoryDB()))
		if err := db.Open(context.Background()); err != nil {
			t.Fatalf("[%s] error opening db %v", tt.name, err)
		}
		t.Cleanup(func() {
			db.Close()
		})

		ss := &scrapeServer{
			ctx:             context.Background(),
			settingsStorage: settings.NewDomainSettingsStorage(db),
		}

		r := httptest.NewRequest("PUT", "/foo/bar/{DOMAIN}", strings.NewReader(tt.payload))
		w := httptest.NewRecorder()
		chain := Chain(
			ss.putDomainSettings,
			domainExtractor,
			DecodeJSONBody[settings.DomainSettings](),
		)
		chain(w, r)
		if w.Code != tt.expectStatus {
			t.Errorf("[%s]: got status %d, want %d", tt.name, w.Code, tt.expectStatus)
			continue
		}
		if tt.expectStatus != 200 {
			continue
		}
		_, err := ss.settingsStorage.Fetch("example.com")
		if err != nil {
			t.Errorf("[%s]: error fetching domain settings %v", tt.name, err)
			continue
		}
		body := w.Result().Body
		saved := new(settings.DomainSettings)
		if err := json.NewDecoder(body).Decode(saved); err != nil {
			t.Errorf("[%s]: error decoding response %v", tt.name, err)
			continue
		}
	}
}

func TestExtractBatchDomainSettings(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		expectStatus int
		expect       batchDomainSettingsRequest
	}{
		{
			name:         "empty",
			queryString:  "",
			expectStatus: 200,
			expect:       batchDomainSettingsRequest{Limit: MaxDomainSettingsBatchSize},
		},
		{
			name:         "fully populated",
			queryString:  "q=foo&offset=1&limit=2",
			expectStatus: 200,
			expect:       batchDomainSettingsRequest{Query: "foo", Offset: 1, Limit: 2},
		},
		{
			name:         "limit too high",
			queryString:  "limit=1001",
			expectStatus: 200,
			expect:       batchDomainSettingsRequest{Limit: MaxDomainSettingsBatchSize},
		},
		{
			name:         "negative offset",
			queryString:  "offset=-1",
			expectStatus: 400,
			expect:       batchDomainSettingsRequest{},
		},
		{
			name:         "case folding query",
			queryString:  "q=FOO",
			expectStatus: 200,
			expect:       batchDomainSettingsRequest{Query: "foo", Limit: MaxDomainSettingsBatchSize},
		},
	}

	okHandler := func(testname string, expect batchDomainSettingsRequest) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			ds, _ := r.Context().Value(dsKey{}).(*batchDomainSettingsRequest)
			if ds.Query != expect.Query {
				t.Errorf("[%s]: got query %q, want %q", testname, ds.Query, expect.Query)
			}
			if ds.Offset != expect.Offset {
				t.Errorf("[%s]: got offset %d, want %d", testname, ds.Offset, expect.Offset)
			}
			if ds.Limit != expect.Limit {
				t.Errorf("[%s]: got limit %d, want %d", testname, ds.Limit, expect.Limit)
			}
		}
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/foo/bar?"+tt.queryString, nil)
		w := httptest.NewRecorder()
		chain := Chain(okHandler(tt.name, tt.expect), extractBatchDomainSettingsQuery(dsKey{}))
		chain(w, r)
		if w.Code != tt.expectStatus {
			t.Errorf("[%s]: got status %d, want %d", tt.name, w.Code, tt.expectStatus)
		}
	}
}

func TestGetBatchDomainSettings(t *testing.T) {
	tests := []struct {
		name         string
		error        error
		settings     []settings.DomainSettings
		expectStatus int
	}{
		{
			name:         "empty",
			expectStatus: 200,
		},
		{
			name: "single",
			settings: []settings.DomainSettings{
				{Domain: "example.com", Sitename: "example"},
			},
			expectStatus: 200,
		},
		{
			name: "multiple",
			settings: []settings.DomainSettings{
				{Domain: "example.com", Sitename: "example"},
				{Domain: "example2.org", Sitename: "example2"},
			},
			expectStatus: 200,
		},
		{
			name:         "invalid query error",
			error:        settings.ErrInvalidQuery,
			expectStatus: 400,
		},
		{
			name:         "some other error",
			error:        errors.New("some random error"),
			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		dss := &mockDomainSettingsStorage{
			settings: tt.settings,
			error:    tt.error,
		}
		ss := &scrapeServer{
			ctx:             context.Background(),
			settingsStorage: dss,
		}

		r := httptest.NewRequest("GET", "/foo/bar", nil)
		w := httptest.NewRecorder()
		chain := Chain(
			ss.getBatchDomainSettings,
			extractBatchDomainSettingsQuery(),
		)
		chain(w, r)
		if w.Code != tt.expectStatus {
			t.Errorf("[%s]: got status %d, want %d", tt.name, w.Code, tt.expectStatus)
			continue
		}
		if tt.expectStatus != 200 {
			continue
		}
		body := w.Result().Body
		result := new(batchDomainSettingsResponse)
		if err := json.NewDecoder(body).Decode(&result); err != nil {
			t.Errorf("[%s]: error decoding response %v", tt.name, err)
			continue
		}
		if len(result.Settings) != len(tt.settings) {
			t.Errorf("[%s]: got %d settings, want %d", tt.name, len(result.Settings), len(tt.settings))
			continue
		}
	}
}

type mockDomainSettingsStorage struct {
	settings []settings.DomainSettings
	error    error
}

func (m *mockDomainSettingsStorage) Fetch(domain string) (settings.DomainSettings, error) {
	if m.error != nil {
		return settings.DomainSettings{}, m.error
	}
	return m.settings[0], nil
}

func (m mockDomainSettingsStorage) FetchRange(offset, limit int, query string) ([]settings.DomainSettings, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.settings, nil
}

func (m *mockDomainSettingsStorage) Save(ds *settings.DomainSettings) error {
	if m.error != nil {
		return m.error
	}
	return nil
}

func (m *mockDomainSettingsStorage) Delete(domain string) (bool, error) {
	if m.error != nil {
		return false, m.error
	}
	return true, nil
}

func init() {
	goose.SetLogger(goose.NopLogger())
}
