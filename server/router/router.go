// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

package router

import (
	"database/sql"
	"net/http"

	"github.com/danielhkuo/quickly-pick/cliparse"
	"github.com/danielhkuo/quickly-pick/handlers"
	"github.com/danielhkuo/quickly-pick/middleware"
)

func NewRouter(db *sql.DB, cfg cliparse.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// Initialize handlers
	pollHandler := handlers.NewPollHandler(db, cfg)
	votingHandler := handlers.NewVotingHandler(db, cfg)
	resultsHandler := handlers.NewResultsHandler(db, cfg)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Poll management (admin operations)
	mux.HandleFunc("POST /polls", middleware.WithLogging(pollHandler.CreatePoll))
	mux.HandleFunc("GET /polls/{id}/admin", middleware.WithLogging(pollHandler.GetPollAdmin))
	mux.HandleFunc("POST /polls/{id}/options", middleware.WithLogging(pollHandler.AddOption))
	mux.HandleFunc("POST /polls/{id}/publish", middleware.WithLogging(pollHandler.PublishPoll))
	mux.HandleFunc("POST /polls/{id}/close", middleware.WithLogging(pollHandler.ClosePoll))

	// Voting operations (public)
	mux.HandleFunc("POST /polls/{slug}/claim-username", middleware.WithLogging(votingHandler.ClaimUsername))
	mux.HandleFunc("POST /polls/{slug}/ballots", middleware.WithLogging(votingHandler.SubmitBallot))

	// Results retrieval (public, with sealed results)
	mux.HandleFunc("GET /polls/{slug}", middleware.WithLogging(resultsHandler.GetPoll))
	mux.HandleFunc("GET /polls/{slug}/results", middleware.WithLogging(resultsHandler.GetResults))
	mux.HandleFunc("GET /polls/{slug}/ballot-count", middleware.WithLogging(resultsHandler.GetBallotCount))

	// Root endpoint
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("quickly-pick API v1"))
	})

	return mux
}
