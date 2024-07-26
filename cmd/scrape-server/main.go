// scrape-server launches the `scrape` url fetching and metadata service.
//
// When run without any arguments, the service will start on port 8080
// and use a local SQLite database.
//
// For full documentation on launch options invoke `scrape-server` with the -h flag.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/fetch/trafilatura"
	"github.com/efixler/scrape/internal"
	"github.com/efixler/scrape/internal/auth"
	"github.com/efixler/scrape/internal/cmd"
	"github.com/efixler/scrape/internal/headless"
	"github.com/efixler/scrape/internal/server"
	"github.com/efixler/scrape/internal/storage"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/ua"
	"github.com/efixler/webutil/graceful"
)

const (
	DefaultPort = 8080
)

var (
	flags           flag.FlagSet
	host            *envflags.Value[string]
	port            *envflags.Value[int]
	signingKey      *envflags.Value[*auth.HMACBase64Key]
	ttl             *envflags.Value[time.Duration]
	userAgent       *envflags.Value[*ua.UserAgent]
	dbFlags         *cmd.DatabaseFlags
	headlessEnabled *envflags.Value[bool]
	profile         *envflags.Value[bool]
	publicHome      *envflags.Value[bool]
	logWriter       io.Writer
)

func main() {
	slog.Info("scrape-server starting up", "addr", fmt.Sprintf("%s:%d", host.Get(), port.Get()))
	// use this context to handle resources hanging off mux handlers
	ctx, cancel := context.WithCancel(context.Background())
	dbh := dbFlags.MustDatabase()
	dbFlags = nil
	if err := dbh.Open(ctx); err != nil {
		slog.Error("scrape-server error opening database", "database", dbh, "error", err)
		os.Exit(1)
	}

	directClient := fetch.MustClient(fetch.WithUserAgent(userAgent.Get().String()))
	var headlessFetcher fetch.URLFetcher = nil
	if headlessEnabled.Get() {
		headlessClient := headless.MustChromeClient(ctx, userAgent.Get().String(), 6)
		headlessFetcher = trafilatura.MustNew(headlessClient)
	}

	sbf := internal.NewStorageBackedFetcher(
		trafilatura.MustNew(directClient),
		storage.NewURLDataStore(dbh),
	)

	ss := server.MustScrapeServer(
		ctx,
		server.WithURLFetcher(sbf),
		server.WithHeadlessIf(headlessFetcher),
		server.WithAuthorizationIf(*signingKey.Get()),
	)

	if ss.AuthEnabled() {
		slog.Info("scrape-server authorization via JWT is enabled")
	} else {
		slog.Info("scrape-server authorization is disabled, running in open access mode")
	}

	mux, err := server.InitMux(
		ss,
		dbh,
		publicHome.Get(),
		profile.Get(),
	)
	if err != nil {
		slog.Error("scrape-server error initializing the server's mux", "error", err)
		os.Exit(1)
	}
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", host.Get(), port.Get()),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second, // some feed/batch requests can be slow to complete
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("scrape-server shutting down", "error", err)
			os.Exit(1)
		}
	}()
	slog.Info("scrape-server started", "addr", s.Addr)
	graceful.WaitForShutdown(s, cancel)
	slog.Info("scrape-server bye!")
	if logFile, ok := (logWriter).(*os.File); ok {
		logFile.Sync()
	}
}

func init() {
	logWriter = os.Stderr
	envflags.EnvPrefix = "SCRAPE_"
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	dbFlags = cmd.AddDatabaseFlags("DB", &flags, false)

	headlessEnabled = envflags.NewBool("ENABLE_HEADLESS", false)
	headlessEnabled.AddTo(&flags, "enable-headless", "Enable headless browser extraction functionality")

	host = envflags.NewString("HOST", "")
	host.AddTo(&flags, "host", "TCP address to listen on (empty for all interfaces)")

	port = envflags.NewInt("PORT", DefaultPort)
	port.AddTo(&flags, "port", "Port to run the server on")

	signingKey = envflags.NewText("SIGNING_KEY", &auth.HMACBase64Key{})
	signingKey.AddTo(
		&flags,
		"signing-key",
		"Base64 encoded HS256 key to verify JWT tokens. Required for JWT auth, and enables JWT auth if set.",
	)

	ttl = envflags.NewDuration("TTL", resource.DefaultTTL)
	ttl.AddTo(&flags, "ttl", "TTL for fetched resources")

	defaultUA := ua.UserAgent(fetch.DefaultUserAgent)
	userAgent = envflags.NewText("USER_AGENT", &defaultUA)
	userAgent.AddTo(&flags, "user-agent", "User agent for fetching")

	profile = envflags.NewBool("PROFILE", false)
	profile.AddTo(&flags, "profile", "Enable profiling at /debug/pprof")

	publicHome = envflags.NewBool("PUBLIC_HOME", false)
	publicHome.AddTo(&flags, "public-home", "Enable the homepage without requiring a token (when auth is enabled)")

	logLevel := envflags.NewLogLevel("LOG_LEVEL", slog.LevelInfo)
	logLevel.AddTo(&flags, "log-level", "Set the log level [debug|error|info|warn]")
	flags.Parse(os.Args[1:])
	logger := slog.New(slog.NewTextHandler(
		logWriter,
		&slog.HandlerOptions{
			Level: logLevel.Get(),
		},
	))
	slog.SetDefault(logger)
}

func usage() {
	fmt.Println(`
Usage:
-----
scrape-server [-port nnnn] [-h]

Some options have environment variable equivalents. Invalid environment settings
are ignored. Command line options override environment variables.
	
If environment variables are set, they'll override the defaults displayed in this 
help message.
 
Command line options:
--------------------

  -h	
  	Show this help message`)

	flags.PrintDefaults()
}
