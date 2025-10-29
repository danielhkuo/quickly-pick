package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/danielhkuo/quickly-pick/cliparse"
	"github.com/danielhkuo/quickly-pick/router"
	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"
)

func main() {

	_ = godotenv.Load(".env")

	// Parse flags
	cfg, err := cliparse.ParseFlags(os.Args[1:])
	if err != nil {
		slog.Error("Error parsing flags", "error", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open(cfg.DatabaseType, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	// Create router with dependencies
	mux := router.NewRouter(db, cfg)

	// Create server
	server := http.Server{
		Handler: mux,
		Addr:    ":" + strconv.Itoa(cfg.Port),
	}

	// signal.Notify requires the channel to be buffered
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for Ctrl-C signal
		<-ctrlc
		server.Close()
	}()

	// Start server
	slog.Info("Listening", "port", cfg.Port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("Server closed", "error", err)
	} else {
		slog.Info("Server closed", "error", err)
	}
}
