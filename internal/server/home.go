package server

import (
	"bytes"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/version"
)

//go:embed htdocs/index.html
var home embed.FS

// mustHomeTemplate creates a template for the home page.
// To enable usage of the home page without a token when auth is enabled,
// for API endpoint, set openHome to true.
func mustHomeTemplate(ss *scrapeServer, openHome bool) *template.Template {
	tmpl := template.New("home")
	var authTokenF = func() string { return "" }
	var showTokenWidget = func() bool {
		// when openHome is true don't show the token entry widget
		if openHome {
			return false
		}
		return ss.AuthEnabled()
	}
	if ss.AuthEnabled() && openHome {
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
		"AuthToken":       authTokenF,
		"ShowTokenWidget": showTokenWidget,
	}
	tmpl = tmpl.Funcs(funcMap)
	homeSource, _ := home.ReadFile("htdocs/index.html")
	tmpl = template.Must(tmpl.Parse(string(homeSource)))
	return tmpl
}

func homeHandler(ss *scrapeServer, openHome bool) http.HandlerFunc {
	tmpl := mustHomeTemplate(ss, openHome)
	data := struct {
		Commit  string
		RepoURL string
		Tag     string
	}{
		Commit:  version.Commit,
		RepoURL: version.RepoURL,
		Tag:     version.Tag,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			http.Error(w, "Error rendering home page", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}
