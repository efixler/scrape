package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/efixler/scrape/internal/settings"
	"github.com/efixler/scrape/store"
)

const MaxDomainSettingsBatchSize = 1000

type singleDomainRequest struct {
	Domain      string `json:"domain"`
	PrettyPrint bool   `json:"pp,omitempty"`
}

type batchDomainSettingsRequest struct {
	Query  string `json:"q,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type batchDomainSettingsResponse struct {
	Request  batchDomainSettingsRequest `json:"request"`
	Settings []*settings.DomainSettings `json:"settings"`
}

type dsKey struct{}

func (ss *scrapeServer) getSingleDomainSettingsHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(4096), extractDomainFromPath(dsKey{}))
	return Chain(ss.getSingleDomainSettings, ms...)
}

func (ss *scrapeServer) getSingleDomainSettings(w http.ResponseWriter, r *http.Request) {
	// if this value isn't here the request will have already been rejected
	// by middleware.
	req, _ := r.Context().Value(dsKey{}).(*singleDomainRequest)
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
	writeJSONOutput(w, ds, req.PrettyPrint, http.StatusOK)
}

func (ss *scrapeServer) putDomainSettingsHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(
		MaxBytes(4096),
		extractDomainFromPath(dsKey{}),
		DecodeJSONBody[settings.DomainSettings](),
	)
	return Chain(ss.putDomainSettings, ms...)
}

func (ss *scrapeServer) putDomainSettings(w http.ResponseWriter, r *http.Request) {
	// for put we get the domain value from here
	req, _ := r.Context().Value(dsKey{}).(*singleDomainRequest)
	ds, _ := r.Context().Value(payloadKey{}).(*settings.DomainSettings)
	ds.Domain = req.Domain
	if err := ss.settingsStorage.Save(ds); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	writeJSONOutput(w, ds, req.PrettyPrint, http.StatusOK)
}

func extractDomainFromPath(key ...any) middleware {
	var pkey any
	pkey = payloadKey{}
	if len(key) > 0 {
		pkey = key[0]
	}
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
			r = r.WithContext(context.WithValue(r.Context(), pkey, v))
			next(w, r)
		}
	}
}

func (ss *scrapeServer) getBatchDomainSettingsHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(
		MaxBytes(4096),
		extractBatchDomainSettingsQuery(),
	)
	return Chain(ss.getBatchDomainSettings, ms...)
}

func (ss *scrapeServer) getBatchDomainSettings(w http.ResponseWriter, r *http.Request) {
	req, _ := r.Context().Value(payloadKey{}).(*batchDomainSettingsRequest)
	dss, err := ss.settingsStorage.FetchRange(req.Offset, req.Limit, req.Query)
	if err != nil {
		switch err {
		case settings.ErrInvalidQuery:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
		return
	}
	writeJSONOutput(w, &batchDomainSettingsResponse{
		Request:  *req,
		Settings: dss,
	}, false, http.StatusOK)
}

func extractBatchDomainSettingsQuery(key ...any) middleware {
	var pkey any
	pkey = payloadKey{}
	if len(key) > 0 {
		pkey = key[0]
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			v := new(batchDomainSettingsRequest)
			v.Query = strings.ToLower(r.FormValue("q"))
			var (
				err    error
				offset int = 0
			)
			switch r.FormValue("offset") {
			case "":
			// no offset specified, use the default
			default:
				offset, err = strconv.Atoi(r.FormValue("offset"))
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("Invalid offset: %s", err)))
					return
				}
				if offset < 0 {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("Invalid offset %d: must be >= 0", offset)))
					return
				}
			}
			v.Offset = offset
			limit := MaxDomainSettingsBatchSize
			switch r.FormValue("limit") {
			case "":
			// no limit specified, use the default
			default:
				limit, err = strconv.Atoi(r.FormValue("limit"))
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("Invalid limit: %s", err)))
					return
				}
				if limit > MaxDomainSettingsBatchSize {
					limit = MaxDomainSettingsBatchSize
				}
			}
			v.Limit = limit
			r = r.WithContext(context.WithValue(r.Context(), pkey, v))
			next(w, r)
		}
	}
}

func (ss *scrapeServer) deleteDomainSettingsHandler() http.HandlerFunc {
	ms := ss.withAuthIfEnabled(MaxBytes(4096), extractDomainFromPath(dsKey{}))
	return Chain(ss.deleteDomainSettings, ms...)
}

func (ss *scrapeServer) deleteDomainSettings(w http.ResponseWriter, r *http.Request) {
	req, _ := r.Context().Value(dsKey{}).(*singleDomainRequest)
	if _, err := ss.settingsStorage.Delete(req.Domain); err != nil {
		if errors.Is(err, settings.ErrInvalidDomain) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
