package server

import (
	"net/http"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/server/admin"
	"github.com/efixler/scrape/internal/server/healthchecks"
)

// Mux Initialization Arguments:
//   - ss: scrapeServer: used for setting up API routes
//   - db: database.DBHandle: used for healthchecks
//   - openHome: bool: if true, the the page will always be open, even
//     if auth is enabled
//   - enableProfiling: bool: if true, pprof routes will be added to the mux
func InitMux(
	ss *scrapeServer,
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
	h := ss.singleHandler()
	mux.HandleFunc("GET /extract", h)
	mux.HandleFunc("POST /extract", h)
	h = ss.singleHeadlessHandler()
	mux.HandleFunc("GET /extract/headless", h)
	mux.HandleFunc("POST /extract/headless", h)
	mux.HandleFunc("POST /batch", ss.batchHandler())
	mux.HandleFunc("DELETE /extract", ss.deleteHandler())
	h = ss.feedHandler()
	mux.HandleFunc("GET /feed", h)
	mux.HandleFunc("POST /feed", h)
	// settings
	if ss.settingsStorage != nil {
		mux.HandleFunc("GET /settings/domain/{DOMAIN}", ss.getSingleDomainSettingsHandler())
		mux.HandleFunc("PUT /settings/domain/{DOMAIN}", ss.putDomainSettingsHandler())
		mux.HandleFunc("GET /settings/domain", ss.getBatchDomainSettingsHandler())
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
