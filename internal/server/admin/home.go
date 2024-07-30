package admin

import (
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/efixler/scrape/internal/auth"
)

// mustHomeTemplate creates a template for the home page.
// To enable usage of the home page without a token when auth is enabled,
// for API endpoint, set openHome to true.
func (a *adminServer) mustHomeTemplate(ss AuthzProvider, openHome bool) *template.Template {
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
	tmpl := a.mustTemplate("index.html", funcMap)
	return tmpl
}

func (a *adminServer) homeHandler(ss AuthzProvider, openHome bool) http.HandlerFunc {
	tmpl := a.mustHomeTemplate(ss, openHome)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, baseTemplateName, a.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
