package router

import (
	"database/sql"
	"net/http"

	"github.com/danielhkuo/quickly-pick/cliparse"
)

func NewRouter(db *sql.DB, cfg cliparse.Config) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	return mux
}
