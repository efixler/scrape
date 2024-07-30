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
	// mux.HandleFunc("GET /settings", as.settingsHandler())

	admin.MustServer(
		mux,
		admin.WithAuthz(ss),
		admin.WithOpenHome(openHome),
		admin.WithProfiling(enableProfiling),
	)
	// as := newAdminServer()
	// mux.HandleFunc("GET /{$}", as.homeHandler(ss, openHome))
	// mux.Handle("/assets/", assetsHandler())

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

	// healthchecks
	mux.Handle("GET /.well-known/", healthchecks.Handler("/.well-known", db))
	return mux, nil
}
