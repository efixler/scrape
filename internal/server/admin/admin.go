package admin

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"

	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/server/version"
)

const (
	baseTemplateName = "base.html"
	DefaultBasePath  = "/admin"
)

type AuthzProvider interface {
	AuthEnabled() bool
	SigningKey() auth.HMACBase64Key
}

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

type config struct {
	basePath string
	authz    AuthzProvider
	openHome bool
	profile  bool
}

type option func(*config) error

func WithBasePath(basePath string) option {
	return func(c *config) error {
		c.basePath = basePath
		return nil
	}
}

func WithAuthz(authz AuthzProvider) option {
	return func(c *config) error {
		c.authz = authz
		return nil
	}
}

func WithOpenHome(openHome bool) option {
	return func(c *config) error {
		c.openHome = openHome
		return nil
	}
}

func WithProfiling(profile bool) option {
	return func(c *config) error {
		c.profile = profile
		return nil
	}
}

func MustServer(mux *http.ServeMux, options ...option) *adminServer {
	s, err := NewServer(mux, options...)
	if err != nil {
		panic(err)
	}
	return s
}

func NewServer(mux *http.ServeMux, options ...option) (*adminServer, error) {
	c := &config{
		basePath: DefaultBasePath,
	}

	for _, o := range options {
		if err := o(c); err != nil {
			slog.Error("AdminServer: Error applying option", "error", err)
			return nil, err
		}
	}
	as := &adminServer{
		data: codeData{
			Commit:  version.Commit,
			RepoURL: version.RepoURL,
			Tag:     version.Tag,
		},
	}
	// nil mux provided for tests
	if mux != nil {
		// home handler is always at root
		mux.HandleFunc("/{$}", as.homeHandler(c.authz, c.openHome))
		mux.Handle("/assets/", assetsHandler())
		if c.profile {
			initPProf(mux, c.basePath)
		}
		mux.HandleFunc(c.basePath+"/settings", as.settingsHandler())
	}
	return as, nil
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

func (a *adminServer) settingsHandler() http.HandlerFunc {
	tmpl := a.mustTemplate("settings.html", nil)
	return func(w http.ResponseWriter, r *http.Request) {
		if err := tmpl.ExecuteTemplate(w, baseTemplateName, a.data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
