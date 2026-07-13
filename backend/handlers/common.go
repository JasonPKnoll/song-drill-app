package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"song-drill-backend/db"
)

// Env holds shared dependencies for all handlers.
type Env struct {
	DB *sql.DB
}

func NewEnv(database *sql.DB) *Env {
	return &Env{DB: database}
}

type contextKey int

const userIDContextKey contextKey = iota

// activeProfileCookie names the profile a request should act as. This is a
// plain, unsigned preference cookie, not a credential — the app is
// Tailscale-only, so network access is already gated; see the Profiles
// section of song_drill_schema.md.
const activeProfileCookie = "song_drill_user"

// WithActiveUser resolves which profile a request acts as, falling back to
// the earliest-created profile if the cookie is missing or names a profile
// that no longer exists (e.g. deleted from another tab).
func (e *Env) WithActiveUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID int64
		var resolved bool

		if cookie, err := r.Cookie(activeProfileCookie); err == nil {
			if id, err := strconv.ParseInt(cookie.Value, 10, 64); err == nil {
				if u, err := db.GetUser(e.DB, id); err == nil && u != nil {
					userID, resolved = id, true
				}
			}
		}

		if !resolved {
			id, err := db.FirstUserID(e.DB)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "no profile available: "+err.Error())
				return
			}
			userID = id
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDContextKey).(int64)
	return id
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
