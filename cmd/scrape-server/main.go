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
	"github.com/efixler/scrape/internal/cmd"
	"github.com/efixler/scrape/internal/headless"
	"github.com/efixler/scrape/internal/server"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/webutil/graceful"
)

const (
	DefaultPort = 8080
)

var (
	flags           flag.FlagSet
	port            *envflags.Value[int]
	ttl             *envflags.Value[time.Duration]
	userAgent       *envflags.Value[string]
	dbFlags         *cmd.DatabaseFlags
	headlessEnabled bool
	profile         *envflags.Value[bool]
	logWriter       io.Writer
)

func main() {
	slog.Info("scrape-server starting up", "port", port.Get())
	// use this context to handle resources hanging off mux handlers
	ctx, cancel := context.WithCancel(context.Background())
	dbFactory, err := dbFlags.Database()
	if err != nil {
		slog.Error("scrape-server error creating database factory", "error", err, "dbSpec", dbFlags)
		os.Exit(1)
	}
	dbFlags = nil

	var headlessClient fetch.Client = nil
	if headlessEnabled {
		headlessClient = headless.MustChromeClient(ctx, 6)
	}
	// headlessClient = fetch.MustClient(fetch.WithUserAgent(userAgent.Get()))

	mux, err := server.InitMux(ctx, dbFactory, headlessClient)
	if err != nil {
		slog.Error("scrape-server error initializing the server's mux", "error", err)
		os.Exit(1)
	}
	if profile.Get() {
		server.EnableProfiling(mux)
	}
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port.Get()),
		Handler:        mux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
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
	logFile, ok := (logWriter).(*os.File)
	if ok {
		logFile.Sync()
	}
}

func init() {
	logWriter = os.Stderr
	envflags.EnvPrefix = "SCRAPE_"
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	dbFlags = cmd.AddDatabaseFlags("DB", &flags, false)

	flags.BoolVar(&headlessEnabled, "headless", false, "Use headless browser")

	port = envflags.NewInt("PORT", DefaultPort)
	port.AddTo(&flags, "port", "Port to run the server on")

	ttl = envflags.NewDuration("TTL", resource.DefaultTTL)
	ttl.AddTo(&flags, "ttl", "TTL for fetched resources")

	userAgent = envflags.NewString("USER_AGENT", fetch.DefaultUserAgent)
	userAgent.AddTo(&flags, "user-agent", "User agent to use for fetching")

	profile = envflags.NewBool("PROFILE", false)
	profile.AddTo(&flags, "profile", "Enable profiling at /debug/pprof")

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
