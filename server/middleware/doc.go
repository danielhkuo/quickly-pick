// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package middleware provides HTTP middleware and helper functions.

# Request Logging

Wrap handlers with request logging:

	mux.HandleFunc("GET /health", middleware.WithLogging(handler))

Logs request start (method, path, remote) and completion (duration_ms).

# CORS Middleware

Enable cross-origin requests for frontend access:

	server := http.Server{
		Handler: middleware.CORS(mux),
	}

Allows methods GET, POST, PUT, DELETE, OPTIONS with headers
Content-Type, Authorization, X-Admin-Key, X-Voter-Token.

# JSON Helpers

Write JSON responses:

	middleware.JSONResponse(w, http.StatusOK, data)
	middleware.ErrorResponse(w, http.StatusBadRequest, "message")

Parse JSON request bodies:

	var req models.CreatePollRequest
	if err := middleware.ParseJSONBody(r, &req); err != nil {
		middleware.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

# Client IP Extraction

Get the original client IP (handles X-Forwarded-For, X-Real-IP):

	ip := middleware.GetClientIP(r)

Used for IP hashing in fraud detection.
*/
package middleware
