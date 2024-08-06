package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	nurl "net/url"

	"github.com/efixler/scrape/internal/server/middleware"
)

func parseSinglePayload() middleware.Step {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			pp := r.FormValue("pp") == "1"
			v := new(SingleURLRequest)
			if middleware.IsJSONRequest(r) {
				decoder := json.NewDecoder(r.Body)
				decoder.DisallowUnknownFields()
				err := decoder.Decode(v)
				if !middleware.AssertJSONDecode(err, w) {
					return
				}
			} else {
				url := r.FormValue("url")
				if url == "" {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("No URL provided"))
					return
				}
				netUrl, err := nurl.Parse(url)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("Invalid URL provided: %q, %s", url, err)))
					return
				}
				v.URL = netUrl
			}
			if pp {
				v.PrettyPrint = true
			}
			slog.Debug("ParseSingle", "url", v.URL, "pp", v.PrettyPrint, "encoding", r.Header.Get("Content-Type"))
			r = r.WithContext(context.WithValue(r.Context(), payloadKey{}, v))
			next(w, r)
		}
	}
}
