package server

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/efixler/scrape/internal/auth"
)

//go:embed templates/index.html
var home embed.FS

func mustHomeTemplate(ss *scrapeServer) *template.Template {
	tmpl := template.New("home")
	var authTokenF = func() string { return "" }
	var authEnabledF = func() bool { return ss.AuthEnabled() }
	if authEnabledF() {
		authTokenF = func() string {
			c, err := auth.NewClaims(
				auth.WithSubject("home"),
				auth.ExpiresIn(60*time.Minute),
			)
			if err != nil {
				slog.Error("Error creating claims for home view", "error", err)
				return ""
			}
			s, err := c.Sign(ss.SigningKey())
			if err != nil {
				slog.Error("Error signing claims for home view", "error", err)
				return ""
			}
			return s
		}
	}
	funcMap := template.FuncMap{
		"AuthToken":   authTokenF,
		"AuthEnabled": authEnabledF,
	}
	tmpl = tmpl.Funcs(funcMap)
	homeSource, _ := home.ReadFile("templates/index.html")
	tmpl = template.Must(tmpl.Parse(string(homeSource)))
	return tmpl
}

func homeHandler(ss *scrapeServer) http.HandlerFunc {
	tmpl := mustHomeTemplate(ss)
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, nil); err != nil {
			http.Error(w, "Error rendering home page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}
