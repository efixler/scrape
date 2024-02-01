package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/resource"
	"github.com/efixler/scrape/server"
	"github.com/efixler/scrape/store/sqlite"
)

const (
	DefaultPort = 8080
)

var (
	flags     flag.FlagSet
	port      int = DefaultPort
	profile   bool
	logLevel  slog.Level
	logWriter io.Writer
)

// TODO: Create the db on startup if it doesn't exist
func main() {
	slog.Info("scrape-server starting up", "port", port)
	// use this context to handle resources hanging off mux handlers
	ctx, cancel := context.WithCancel(context.Background())
	mux, err := server.InitMux(ctx, profile)
	if err != nil {
		slog.Error("scrape-server error initializing the server's mux", "error", err)
		os.Exit(1)
	}
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("scrape-server shutting down", "error", err)
		}
	}()
	slog.Info("scrape-server started", "addr", s.Addr)
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-kill
	wchan := shutdownServer(s, cancel)
	<-wchan
	slog.Info("scrape-server bye!")
	logFile, ok := (logWriter).(*os.File)
	if ok {
		logFile.Sync()
	}
}

// Shutdown the server and then progate the shutdown to the mux
// This will let the requests finish before shutting down the db
// cf is the cancel function for the mux context, or, generically
// speaking, a cancel function to queue up after the server is done
// Caller should block on the returned channel.
func shutdownServer(s *http.Server, cf context.CancelFunc) chan bool {
	slog.Info("scrape-server shutting down")
	wchan := make(chan bool)
	// a large request set can take a while to finish,
	// so we give the server a couple minutes to finish if it needs to
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	context.AfterFunc(ctx, func() {
		cf()
		// without a little bit of sleep here sometimes final log messages
		// don't get flushed, even with the file sync above
		time.Sleep(100 * time.Millisecond)
		close(wchan)
	})
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slog.Error("scrape-server shutdown failed", "error", err)
	}
	slog.Info("scrape-server stopped")
	return wchan
}

func init() {
	logWriter = os.Stderr
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage

	if os.Getenv("SCRAPE_DB") != "" {
		sqlite.DefaultDatabase = os.Getenv("SCRAPE_DB")
	}
	flags.StringVar(&sqlite.DefaultDatabase,
		"database",
		sqlite.DefaultDatabase,
		`Database path. If the database doesn't exist, it will be created.
Use ':memory:' for an in-memory database
Environment variable equivalent: SCRAPE_DB
`,
	)
	if envPort, err := strconv.Atoi(os.Getenv("SCRAPE_PORT")); err == nil {
		port = envPort
	}
	flags.IntVar(&port, "port", port, "Port to run the server on\nEnvironment variable equivalent: SCRAPE_PORT\n")

	if ttl, err := time.ParseDuration(os.Getenv("SCRAPE_TTL")); err == nil {
		resource.DefaultTTL = ttl
	} else if os.Getenv("SCRAPE_TTL") != "" {
		slog.Error("scrape-server error parsing ttl from environment, ignoring",
			"SCRAPE_TTL", os.Getenv("SCRAPE_TTL"),
			"error", err,
			"default", resource.DefaultTTL,
		)
	}
	flags.DurationVar(
		&resource.DefaultTTL,
		"ttl",
		resource.DefaultTTL,
		"TTL for fetched resources\nEnvironment variable equivalent: SCRAPE_TTL\n",
	)
	var userAgent string = os.Getenv("SCRAPE_USER_AGENT")
	flags.StringVar(
		&userAgent,
		"user-agent",
		fetch.DefaultUserAgent,
		"The user agent to use for fetching\nEnvironment variable equivalent: SCRAPE_USER_AGENT\n",
	)

	flags.BoolVar(&profile, "profile", false, "Enable profiling at /debug/pprof\n (default off)")
	flags.Func(
		"log-level",
		"Set the log level [debug|error|info|warn]\n (default info)",
		func(s string) error {
			switch strings.ToLower(s) {
			case "debug":
				logLevel = slog.LevelDebug
			case "warn":
				logLevel = slog.LevelWarn
			case "error":
				logLevel = slog.LevelError
			}
			return nil
		})
	var showSettings bool
	flags.BoolVar(&showSettings, "s", false, "Show current settings and exit\n (default false)")
	flags.Parse(os.Args[1:])
	logger := slog.New(slog.NewTextHandler(
		logWriter,
		&slog.HandlerOptions{
			Level: logLevel,
		},
	))
	slog.SetDefault(logger)
	if showSettings {
		showOptions()
		os.Exit(0)
	}
}

func showOptions() {
	fmt.Print(`scrape-server options 
(considering environment variables and command line options)`, "\n\n")
	flags.VisitAll(func(f *flag.Flag) {
		var value string
		switch f.Name {
		case "s":
			return
		case "log-level":
			value = logLevel.String()
		default:
			value = f.Value.String()
		}
		fmt.Println("  ", f.Name, ":", value)
	})
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
