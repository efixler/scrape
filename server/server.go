package server

// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"net/http"
// 	"os"
// 	"time"
// )

// func InitServer(ctx context.Context) (chan <-bool, error) {
// 	mctx, mcancel := context.WithCancel(ctx)
// 	mux, err := InitMux(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	rchan := make(chan bool)

// 	s := &http.Server{
// 		Addr:    fmt.Sprintf(":%d", port),
// 		Handler: mux,
// 	}
// 	slog.Info("scrape-server starting", "port", port)
// 	go func() {
// 		if err := s.ListenAndServe(); err != nil {
// 			slog.Error("scrape-server failed", "error", err)
// 		}
// 	}()
// 	slog.Info("scrape-server started", "port", port)
// 	// ctx_srv, cancel_srv := context.WithTimeout(context.Background(), 15*time.Second)
// 	// context.AfterFunc(ctx_srv, func() {
// 	// 	slog.Info("scrape-server context is done, shutting down the mux")
// 	// 	cancel_mux()
// 	// })
// 	sctx := shutdownServer(s)
// 	context.AfterFunc(ctx, func() {
// 		slog.Info("fuuuuck")
// 	})
// 	<-sctx.Done()
// 	cancel()
// 	//<-ctx_mux.Done()
// 	slog.Info("scrape-server bye!")
// 	os.Stderr.Sync()
// 	time.Sleep(3 * time.Second)
// }

// func ShutdownServer(s *http.Server) context.Context {
// 	shutdownServer(s)

// }
// func shutdownServer(s *http.Server) context.Context {
// 	slog.Info("scrape-server shutting down")
// 	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
// 	context.AfterFunc(ctx, func() {
// 		slog.Info("sServer context afterfunc")
// 	})
// 	defer cancel()
// 	if err := s.Shutdown(ctx); err != nil {
// 		slog.Error("scrape-server shutdown failed", "error", err)
// 	}
// 	return ctx
// }
