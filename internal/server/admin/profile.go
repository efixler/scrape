package admin

import (
	"fmt"
	"net/http"
	"net/http/pprof"
)

func initPProf(mux *http.ServeMux, basePath string) {
	// pprof
	mux.HandleFunc(fmt.Sprintf("GET /%s/pprof/", basePath), http.HandlerFunc(pprof.Index))
	mux.HandleFunc(fmt.Sprintf("GET /%s/pprof/cmdline", basePath), http.HandlerFunc(pprof.Cmdline))
	mux.HandleFunc(fmt.Sprintf("GET /%s/pprof/profile", basePath), http.HandlerFunc(pprof.Profile))
	mux.HandleFunc(fmt.Sprintf("GET /%s/pprof/symbol", basePath), http.HandlerFunc(pprof.Symbol))
	mux.HandleFunc(fmt.Sprintf("GET /%s/pprof/trace", basePath), http.HandlerFunc(pprof.Trace))
}
