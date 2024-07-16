package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/server/healthchecks"
)

func InitMux(scrapeServer *scrapeServer, db *database.DBHandle) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", scrapeServer.homeHandler())
	mux.Handle("/assets/", assetsHandler())
	h := scrapeServer.singleHandler()
	mux.HandleFunc("GET /extract", h)
	mux.HandleFunc("POST /extract", h)
	h = scrapeServer.singleHeadlessHandler()
	mux.HandleFunc("GET /extract/headless", h)
	mux.HandleFunc("POST /extract/headless", h)
	mux.HandleFunc("POST /batch", scrapeServer.batchHandler())
	mux.HandleFunc("DELETE /extract", scrapeServer.deleteHandler())
	h = scrapeServer.feedHandler()
	mux.HandleFunc("GET /feed", h)
	mux.HandleFunc("POST /feed", h)
	mux.Handle("GET /.well-known/", healthchecks.Handler("/.well-known", db))
	return mux, nil
}

func EnableProfiling(mux *http.ServeMux) {
	initPProf(mux)
}

func initPProf(mux *http.ServeMux) {
	// pprof
	mux.HandleFunc("GET /debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.HandleFunc("GET /debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc("GET /debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.HandleFunc("GET /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc("GET /debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}
