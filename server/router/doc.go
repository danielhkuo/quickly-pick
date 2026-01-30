// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package router defines HTTP routes for the Quickly Pick API.

# Route Registration

NewRouter creates a configured http.ServeMux with all endpoints:

	mux := router.NewRouter(db, cfg)

# Endpoints

Health:

	GET /health

Poll management (admin, requires X-Admin-Key):

	POST /polls              - Create poll
	GET  /polls/{id}/admin   - Get poll details
	POST /polls/{id}/options - Add option
	POST /polls/{id}/publish - Open for voting
	POST /polls/{id}/close   - Seal results

Voting (public, uses share slug):

	POST /polls/{slug}/claim-username - Claim voter identity
	POST /polls/{slug}/ballots        - Submit/update ballot

Results (public):

	GET /polls/{slug}              - Poll info and options
	GET /polls/{slug}/results      - Final results (closed only)
	GET /polls/{slug}/ballot-count - Vote count
	GET /polls/{slug}/preview      - Compact preview data

Device management:

	POST /devices/register - Register device
	GET  /devices/me       - Get device info
	GET  /devices/my-polls - List device's polls

# Handler Initialization

The router creates handler instances with dependency injection:

	pollHandler := handlers.NewPollHandler(db, cfg)
	votingHandler := handlers.NewVotingHandler(db, cfg)
	resultsHandler := handlers.NewResultsHandler(db, cfg)
	deviceHandler := handlers.NewDeviceHandler(db, cfg)

All handlers receive the database connection and configuration.
*/
package router
