package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	_ "github.com/lib/pq"

	"github.com/danielhkuo/quickly-pick/cliparse"
	"github.com/danielhkuo/quickly-pick/db"
	"github.com/danielhkuo/quickly-pick/router"
)

func main() {
	var err error

	// Parse configuration
	cfg, err := cliparse.ParseFlags(os.Args[1:])
	if err != nil {
		slog.Error("Error parsing flags", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	dbConn, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// Verify connection
	if err := dbConn.Ping(); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	// Create schema (tables)
	if err := db.CreateSchema(dbConn); err != nil {
		slog.Error("schema creation failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database schema ready")

	// Create router
	mux := router.NewRouter(dbConn, cfg)

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
