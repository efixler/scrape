package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/efixler/scrape/internal/settings"
	"github.com/efixler/scrape/store"
)

type singleDomainRequest struct {
	Domain      string `json:"domain"`
	PrettyPrint bool   `json:"pp,omitempty"`
}

func (ss *scrapeServer) singleDomainSettingsHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(4096), parseSingleDomain())
	return Chain(ss.singleDomainSettings, ms...)
}

func (ss *scrapeServer) singleDomainSettings(w http.ResponseWriter, r *http.Request) {
	req, ok := r.Context().Value(payloadKey{}).(*singleDomainRequest)
	if !ok {
		http.Error(w, "Can't process request, no data", http.StatusInternalServerError)
		return
	}

	ds, err := ss.settingsStorage.Fetch(req.Domain)
	if err != nil {
		switch err {
		case store.ErrResourceNotFound:
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if req.PrettyPrint {
		encoder.SetIndent("", "  ")
	}
	encoder.Encode(ds)

	// GET /settings/domain/{domain}
	// or
	// GET /settings/domain?domain={domain}
	// multi =
	// GET /settings/domain/*
	// or
	// GET /settings/domain/*/10/0
	// or
	// GET /settings/domain/?q=foo&limit=10&offset=0
	// q, limit, offset := r.FormValue("q"), r.FormValue
	// ("limit"), r.FormValue("offset")
	//dsr := new(domainSettingsRequest)
}

func parseSingleDomain() middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			pp := r.FormValue("pp") == "1"
			v := new(singleDomainRequest)
			if pp {
				v.PrettyPrint = true
			}
			targetDomain := r.PathValue("DOMAIN")
			if targetDomain == "" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("No domain provided"))
				return
			}
			targetDomain = strings.ToLower(targetDomain)
			if err := settings.ValidateDomain(targetDomain); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				msg := strings.Replace(err.Error(), "\n", ": ", 1)
				w.Write([]byte(msg))
				return
			}
			v.Domain = targetDomain
			//slog.Debug("ParseSingleDomain", "domain", v.Domain, "pp", v.PrettyPrint, "encoding", r.Header.Get("Content-Type"))
			r = r.WithContext(context.WithValue(r.Context(), payloadKey{}, v))
			next(w, r)
		}
	}
}
