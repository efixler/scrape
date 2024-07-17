package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/efixler/scrape/database"
	"github.com/efixler/scrape/internal/server/healthchecks"
)

func InitMux(ss *scrapeServer, db *database.DBHandle, openHome bool) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", homeHandler(ss, openHome))
	mux.Handle("/assets/", assetsHandler())
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
