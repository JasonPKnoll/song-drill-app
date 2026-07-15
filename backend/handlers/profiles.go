package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"song-drill-backend/db"
)

// profileCookieMaxAge is long-lived since this is just a UI preference, not
// a credential — there's nothing sensitive to expire.
const profileCookieMaxAge = 60 * 60 * 24 * 365 // 1 year

const defaultProfileColor = "#a78bfa"

type profileRequest struct {
	DisplayName string `json:"display_name"`
	Color       string `json:"color"`
}

// GET /api/song-drill/profiles
func (e *Env) ListProfiles(c *gin.Context) {
	users, err := db.ListUsers(e.DB)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, users)
}

// GET /api/song-drill/profiles/active
func (e *Env) GetActiveProfile(c *gin.Context) {
	user, err := db.GetUser(e.DB, userIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(c, http.StatusInternalServerError, "active profile no longer exists")
		return
	}
	writeJSON(c, http.StatusOK, user)
}

type setActiveProfileRequest struct {
	ID int64 `json:"id"`
}

// POST /api/song-drill/profiles/active
func (e *Env) SetActiveProfile(c *gin.Context) {
	var req setActiveProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	user, err := db.GetUser(e.DB, req.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(c, http.StatusNotFound, "profile not found")
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(activeProfileCookie, strconv.FormatInt(user.ID, 10), profileCookieMaxAge, "/", "", false, true)
	writeJSON(c, http.StatusOK, user)
}

// POST /api/song-drill/profiles
func (e *Env) CreateProfile(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.DisplayName == "" {
		writeError(c, http.StatusBadRequest, "display_name is required")
		return
	}
	if req.Color == "" {
		req.Color = defaultProfileColor
	}
	user, err := db.CreateUser(e.DB, req.DisplayName, req.Color)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusCreated, user)
}

// PATCH /api/song-drill/profiles/:id
func (e *Env) UpdateProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid profile id")
		return
	}
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.DisplayName == "" {
		writeError(c, http.StatusBadRequest, "display_name is required")
		return
	}
	if req.Color == "" {
		req.Color = defaultProfileColor
	}
	updated, err := db.UpdateUser(e.DB, id, req.DisplayName, req.Color)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !updated {
		writeError(c, http.StatusNotFound, "profile not found")
		return
	}
	user, err := db.GetUser(e.DB, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, user)
}

// DELETE /api/song-drill/profiles/:id
func (e *Env) DeleteProfile(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid profile id")
		return
	}
	deleted, err := db.DeleteUser(e.DB, id)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !deleted {
		writeError(c, http.StatusNotFound, "profile not found")
		return
	}
	writeJSON(c, http.StatusOK, gin.H{"ok": true})
}
