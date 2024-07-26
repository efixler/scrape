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

type codeData struct {
	Commit  string
	RepoURL string
	Tag     string
}
type adminServer struct {
	mutex        sync.Mutex
	baseTemplate *template.Template
	data         codeData
}

func newAdminServer() *adminServer {
	return &adminServer{
		data: codeData{
			Commit:  version.Commit,
			RepoURL: version.RepoURL,
			Tag:     version.Tag,
		},
	}
}

// mustBaseTemplate returns a template for the base template. The returned template
// is cloned so that the base template's namespace is not modified by the material
// pages calling it.
// Any .html files dropped into the includes folder will be included in the base template.
// It is possible that it's important that base.html is the first file included in the template.
// This sorts relatively high right now because it starts with a b.
func (a *adminServer) mustBaseTemplate() *template.Template {
	if a.baseTemplate != nil {
		goto CloneAndReturn
	}
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.baseTemplate == nil {
		d, err := fs.Sub(includes, "htdocs/includes")
		if err != nil {
			panic(err)
		}
		a.baseTemplate = template.Must(template.New("base").ParseFS(d, "*.html"))
		a.baseTemplate = a.baseTemplate.Option("missingkey=zero")
	}
CloneAndReturn:
	clone, err := a.baseTemplate.Clone()
	if err != nil {
		panic(err)
	}
	return clone
}

// Return a template for the given (base) name, from the htdocs directory. Funcs must be provided
// before parsing the template; if no funcs are needed, pass nil.
func (a *adminServer) mustTemplate(name string, funcs template.FuncMap) *template.Template {
	tmpl := a.mustBaseTemplate()
	if funcs != nil {
		tmpl = tmpl.Funcs(funcs)
	}
	tmpl = template.Must(tmpl.ParseFS(htdocs, "htdocs/"+name))
	return tmpl
}

// mustHomeTemplate creates a template for the home page.
// To enable usage of the home page without a token when auth is enabled,
// for API endpoint, set openHome to true.
func (a *adminServer) mustHomeTemplate(ss *scrapeServer, openHome bool) *template.Template {
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

func (a *adminServer) homeHandler(ss *scrapeServer, openHome bool) http.HandlerFunc {
	tmpl := a.mustHomeTemplate(ss, openHome)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, baseTemplateName, a.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (a *adminServer) settingsHandler() http.HandlerFunc {
	tmpl := a.mustTemplate("settings.html", nil)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, baseTemplateName, a.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
