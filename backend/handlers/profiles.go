package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

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
func (e *Env) ListProfiles(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers(e.DB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// GET /api/song-drill/profiles/active
func (e *Env) GetActiveProfile(w http.ResponseWriter, r *http.Request) {
	user, err := db.GetUser(e.DB, userIDFromContext(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(w, http.StatusInternalServerError, "active profile no longer exists")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type setActiveProfileRequest struct {
	ID int64 `json:"id"`
}

// POST /api/song-drill/profiles/active
func (e *Env) SetActiveProfile(w http.ResponseWriter, r *http.Request) {
	var req setActiveProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	user, err := db.GetUser(e.DB, req.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     activeProfileCookie,
		Value:    strconv.FormatInt(user.ID, 10),
		Path:     "/",
		MaxAge:   profileCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, user)
}

// POST /api/song-drill/profiles
func (e *Env) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}
	if req.Color == "" {
		req.Color = defaultProfileColor
	}
	user, err := db.CreateUser(e.DB, req.DisplayName, req.Color)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// PATCH /api/song-drill/profiles/{id}
func (e *Env) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	var req profileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}
	if req.Color == "" {
		req.Color = defaultProfileColor
	}
	updated, err := db.UpdateUser(e.DB, id, req.DisplayName, req.Color)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	user, err := db.GetUser(e.DB, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// DELETE /api/song-drill/profiles/{id}
func (e *Env) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	deleted, err := db.DeleteUser(e.DB, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
