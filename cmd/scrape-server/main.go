package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	flags flag.FlagSet
	port  int
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	slog.Info("Starting scrape-server", "port", port)
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("scrape-server shutting down", "error", err)
		}
	}()
	slog.Info("scrape-server started")
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel() // releases resources.
	if err := s.Shutdown(ctx); err != nil {
		slog.Error("scrape-server shutdown failed", "error", err)
	}
	slog.Info("scrape-server stopped")
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, world!"))
}

func init() {
	flags.Init("", flag.ExitOnError)
	flags.Usage = usage
	flags.IntVar(&port, "port", 8080, "The port to run the server on")
	flags.Parse(os.Args[1:])
}

func usage() {
	fmt.Println(`Usage: 
	scrape-server [-port nnnn] [-h]
 
  -h	
  	Show this help message`)

	flags.PrintDefaults()
}
