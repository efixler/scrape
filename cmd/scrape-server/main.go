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
	"syscall"
	"time"

	"github.com/efixler/envflags"
	"github.com/efixler/scrape/fetch"
	"github.com/efixler/scrape/internal/cmd"
	"github.com/efixler/scrape/internal/server"
	"github.com/efixler/scrape/resource"
)

const (
	DefaultPort = 8080
)

var (
	flags     flag.FlagSet
	port      *envflags.Value[int]
	ttl       *envflags.Value[time.Duration]
	userAgent *envflags.Value[string]
	dbFlags   *cmd.DatabaseFlags
	profile   *envflags.Value[bool]
	logWriter io.Writer
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
	mux, err := server.InitMux(ctx, dbFactory, profile.Get())
	if err != nil {
		slog.Error("scrape-server error initializing the server's mux", "error", err)
		os.Exit(1)
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
	envflags.EnvPrefix = "SCRAPE_"
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	dbFlags = cmd.AddDatabaseFlags("DB", &flags, false)
	_ = cmd.AddProxyConfigFlags("headless", false, &flags)
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
