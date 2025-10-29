package router

import (
	"database/sql"
	"net/http"

	"github.com/danielhkuo/quickly-pick/cliparse"
)

func NewRouter(db *sql.DB, cfg cliparse.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Hello world (replace with your endpoints)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("quickly-pick API"))
	})

	return mux
}
