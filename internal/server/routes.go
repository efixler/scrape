// Router setup;  Subpackages implement the API server and admin servers,
// along with middleware, healthchecks, and utility functions.
package server

import (
	"net/http"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/server/admin"
	"github.com/efixler/scrape/internal/server/api"
	"github.com/efixler/scrape/internal/server/healthchecks"
)

// Mux Initialization Arguments:
//   - ss: api.Server: used for setting up API routes
//   - db: database.DBHandle: used for healthchecks
//   - openHome: bool: if true, the the page will always be open, even
//     if auth is enabled
//   - enableProfiling: bool: if true, pprof routes will be added to the mux
func InitMux(
	ss *api.Server,
	db *database.DBHandle,
	openHome bool,
	enableProfiling bool,
) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	// Admin routes (and home)
	admin.MustServer(
		mux,
		admin.WithAuthz(ss),
		admin.WithOpenHome(openHome),
		admin.WithProfiling(enableProfiling),
	)

	// API routes
	h := ss.ExtractHandler()
	mux.HandleFunc("GET /extract", h)
	mux.HandleFunc("POST /extract", h)
	h = ss.ExtractHeadlessHandler()
	mux.HandleFunc("GET /extract/headless", h)
	mux.HandleFunc("POST /extract/headless", h)
	mux.HandleFunc("POST /batch", ss.BatchHandler())
	mux.HandleFunc("DELETE /extract", ss.DeleteHandler())
	h = ss.FeedHandler()
	mux.HandleFunc("GET /feed", h)
	mux.HandleFunc("POST /feed", h)
	// settings
	// Until settings migrations for MySQL are in place
	if (db != nil) && db.Engine.Driver() == string(database.SQLite) {
		mux.HandleFunc("GET /settings/domain/{DOMAIN}", ss.GetDomainSettingsHandler())
		mux.HandleFunc("PUT /settings/domain/{DOMAIN}", ss.PutDomainSettingsHandler())
		mux.HandleFunc("GET /settings/domain", ss.SearchDomainSettingsHandler())
		mux.HandleFunc("DELETE /settings/domain/{DOMAIN}", ss.DeleteDomainSettingsHandler())
	} else {
		mux.HandleFunc("/settings/domain/", serviceUnavailable)
	}

	// healthchecks
	mux.Handle("GET /.well-known/", healthchecks.Handler("/.well-known", db))
	return mux, nil
}

func serviceUnavailable(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
}
