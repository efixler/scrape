package server

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/version"
)

const (
	baseTemplateName = "base.html"
)

//go:embed htdocs/*.html
var htdocs embed.FS

//go:embed htdocs/includes/*.html
var includes embed.FS

var baseTemplate *template.Template

var tmu sync.Mutex

func mustBaseTemplate() *template.Template {
	if baseTemplate != nil {
		goto CloneAndReturn
	}
	tmu.Lock()
	defer tmu.Unlock()
	if baseTemplate == nil {
		d, err := fs.Sub(includes, "htdocs/includes")
		if err != nil {
			panic(err)
		}
		baseTemplate = template.Must(template.New("base").ParseFS(d, "*.html"))
		baseTemplate = baseTemplate.Option("missingkey=zero")
	}
CloneAndReturn:
	clone, err := baseTemplate.Clone()
	if err != nil {
		panic(err)
	}
	return clone
}

func mustTemplate(name string, funcs template.FuncMap) *template.Template {
	tmpl := mustBaseTemplate()
	if funcs != nil {
		tmpl = tmpl.Funcs(funcs)
	}
	tmpl = template.Must(tmpl.ParseFS(htdocs, "htdocs/"+name))
	return tmpl
}

// mustHomeTemplate creates a template for the home page.
// To enable usage of the home page without a token when auth is enabled,
// for API endpoint, set openHome to true.
func mustHomeTemplate(ss *scrapeServer, openHome bool) *template.Template {
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
	tmpl := mustTemplate("index_block.html", funcMap)
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
		if err := tmpl.ExecuteTemplate(w, baseTemplateName, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	// return func(w http.ResponseWriter, r *http.Request) {
	// 	var buf bytes.Buffer
	// 	if err := tmpl.Execute(&buf, data); err != nil {
	// 		http.Error(w, "Error rendering home page", http.StatusInternalServerError)
	// 		return
	// 	}
	// 	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 	w.WriteHeader(http.StatusOK)
	// 	w.Write(buf.Bytes())
	// }
}

func settingsHandler() http.HandlerFunc {
	//tmpl := mustBaseTemplate()
	//tmpl = template.Must(tmpl.ParseFS(htdocs, "htdocs/settings.html"))
	tmpl := mustTemplate("settings.html", nil)
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
		if err := tmpl.ExecuteTemplate(w, baseTemplateName, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
