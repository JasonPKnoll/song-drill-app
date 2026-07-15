package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
)

// Env holds shared dependencies for all handlers.
type Env struct {
	DB *sql.DB
}

func NewEnv(database *sql.DB) *Env {
	return &Env{DB: database}
}

const userIDContextKey = "userID"

// activeProfileCookie names the profile a request should act as. This is a
// plain, unsigned preference cookie, not a credential — the app is
// Tailscale-only, so network access is already gated; see the Profiles
// section of song_drill_schema.md.
const activeProfileCookie = "song_drill_user"

// WithActiveUser resolves which profile a request acts as, falling back to
// the earliest-created profile if the cookie is missing or names a profile
// that no longer exists (e.g. deleted from another tab).
func (e *Env) WithActiveUser(c *gin.Context) {
	var userID int64
	var resolved bool

	if cookieValue, err := c.Cookie(activeProfileCookie); err == nil {
		if id, err := strconv.ParseInt(cookieValue, 10, 64); err == nil {
			if u, err := db.GetUser(e.DB, id); err == nil && u != nil {
				userID, resolved = id, true
			}
		}
	}

	if !resolved {
		id, err := db.FirstUserID(e.DB)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "no profile available: "+err.Error())
			c.Abort()
			return
		}
		userID = id
	}

	c.Set(userIDContextKey, userID)
	c.Next()
}

func userIDFromContext(c *gin.Context) int64 {
	return c.GetInt64(userIDContextKey)
}

// writeJSON and writeError are thin wrappers around gin.Context.JSON, kept
// so every handler's error-response shape stays identical to before this
// was a plain net/http app: {"error": "message"} for failures, or the raw
// value for a success. gin.H is just map[string]any.
func writeJSON(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}

func writeError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}
