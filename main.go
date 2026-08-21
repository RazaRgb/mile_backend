package main

import (
	"backend/src/auth"
	"backend/src/db"
	"backend/src/handlers"
	"backend/src/utils"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Load backend/.env if present (real env vars always win).
	if err := utils.LoadDotEnv(".env"); err != nil {
		slog.Error("failed to load .env", "error", err)
		os.Exit(1)
	}

	logFile := setLogger()
	defer logFile.Close()

	// Panic at boot if JWT_SECRET is missing (auth requires it).
	auth.Init()

	// Connect to Postgres and ensure the schema exists (exits on failure).
	db.InitDB()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.HandleHealth)
	mux.HandleFunc("GET /{$}", handlers.HandleRoot)

	// Public auth routes.
	mux.HandleFunc("POST /api/v1/register", handlers.HandleRegister)
	mux.HandleFunc("POST /api/v1/login", handlers.HandleLogin)

	// Private (auth-protected) routes.
	mux.Handle("POST /api/v1/stream", handlers.WithAuth(http.HandlerFunc(handlers.HandleCreateStream)))
	mux.Handle("GET /api/v1/streams", handlers.WithAuth(http.HandlerFunc(handlers.HandleListStreams)))
	mux.Handle("GET /api/v1/stream/{stream_id}/tree", handlers.WithAuth(http.HandlerFunc(handlers.HandleGetStreamTree)))
	mux.Handle("DELETE /api/v1/stream/{stream_id}", handlers.WithAuth(http.HandlerFunc(handlers.HandleDeleteStream)))
	mux.Handle("GET /api/v1/debug/trees", handlers.WithAuth(http.HandlerFunc(handlers.HandleDumpTrees)))
	mux.Handle("GET /api/v1/next-articles", handlers.WithAuth(http.HandlerFunc(handlers.HandleGetNextArticles)))
	mux.Handle("GET /api/v1/feed", handlers.WithAuth(http.HandlerFunc(handlers.HandleFeed)))
	mux.Handle("POST /api/v1/article/{node_id}/status", handlers.WithAuth(http.HandlerFunc(handlers.HandleArticleStatus)))

	handler := handlers.WithRecovery(handlers.WithLogging(handlers.WithCORS(mux)))

	server := &http.Server{
		Addr:         ":" + utils.EnvOr("PORT", "8080"),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 300 * time.Second, // feed generation makes LLM calls; can take minutes
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "Addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	//graceful shudown
	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)
	sig := <-stopSig

	slog.Info("shutting down...", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
	}
	slog.Info("server stopped")
}

func setLogger() *os.File {
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open app.log: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(logFile, nil))
	slog.SetDefault(logger)
	return logFile
}
