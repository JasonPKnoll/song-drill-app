package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// Env holds shared dependencies for all handlers.
type Env struct {
	DB *sql.DB
}

func NewEnv(database *sql.DB) *Env {
	return &Env{DB: database}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
