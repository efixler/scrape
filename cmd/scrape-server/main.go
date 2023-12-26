package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/efixler/scrape/server"
	"github.com/efixler/scrape/store/sqlite"
)

var (
	flags  flag.FlagSet
	port   int
	dbPath string
)

// TODO: Create the db on startup if it doesn't exist
func main() {
	slog.Info("scrape-server starting up", "port", port)
	// Create the db if it doesn't exist
	assertDBExists()
	// use this context to handle resources hanging off mux handlers
	ctx, cancel := context.WithCancel(context.Background())
	mux, err := server.InitMux(ctx)
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
	slog.Info("scape-server started", "addr", s.Addr)
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-kill
	wchan := shutdownServer(s, cancel)
	<-wchan
	slog.Info("scrape-server bye!")
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
		// if main() exits before the cancelfunc is done, logs don't get flushed
		// so we wait a bit here. There isn't really a good way to find out when
		// cancel _finishes_ so we wait instead. This would logically be a little
		// better as an AfterFunc to the mux context, but that doesn't work any
		// better than this.
		time.Sleep(2 * time.Second)
		close(wchan)
	})
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slog.Error("scrape-server shutdown failed", "error", err)
	}
	slog.Info("scrape-server stopped")
	return wchan
}

func assertDBExists() {
	err := sqlite.CreateDB(context.Background(), dbPath)
	switch {
	case err == nil:
		slog.Info("scrape-server created database", "path", dbPath)
	case errors.Is(err, sqlite.ErrDatabaseExists):
		slog.Info("scrape-server found db", "path", dbPath)
	default:
		slog.Error("scrape-server error creating database", "error", err)
		panic(err)
	}
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	flags.StringVar(&dbPath,
		"database",
		sqlite.DEFAULT_DB_FILENAME,
		"Database path. If the database doesn't exist, it will be created. \nUse ':memory:' for an in-memory database",
	)
	flags.IntVar(&port, "port", 8080, "The port to run the server on")
	var logLevel slog.Level
	flags.Func(
		"log-level",
		"Set the log level [debug|error|info|warn] (default info)",
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
	flags.Parse(os.Args[1:])
	logger := slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: logLevel,
		},
	))
	slog.SetDefault(logger)
}

func usage() {
	fmt.Println(`Usage: 
	scrape-server [-port nnnn] [-h]
 
  -h	
  	Show this help message`)

	flags.PrintDefaults()
}
