package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"homeplan/api/internal/house"
)

type HouseStore interface {
	EnsureAnonymousSession(ctx context.Context, token string, expiresAt time.Time) error
	LoadCurrentHouse(ctx context.Context, sessionToken string) (json.RawMessage, error)
	SaveCurrentHouse(ctx context.Context, sessionToken string, state json.RawMessage) error
}

func NewRouter(store HouseStore, cookieName string, anonymousTTL time.Duration) http.Handler {
	mux := http.NewServeMux()
	api := apiServer{store: store, cookieName: cookieName, anonymousTTL: anonymousTTL}
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /api/house/current", api.getCurrentHouse)
	mux.HandleFunc("PUT /api/house/current", api.putCurrentHouse)
	return mux
}

type apiServer struct {
	store        HouseStore
	cookieName   string
	anonymousTTL time.Duration
}

func (api apiServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (api apiServer) getCurrentHouse(w http.ResponseWriter, r *http.Request) {
	sessionToken, err := api.ensureSession(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create anonymous session")
		return
	}

	state, err := api.store.LoadCurrentHouse(r.Context(), sessionToken)
	if errors.Is(err, house.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no house saved for this anonymous session")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load house")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(state)
}

func (api apiServer) putCurrentHouse(w http.ResponseWriter, r *http.Request) {
	sessionToken, err := api.ensureSession(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create anonymous session")
		return
	}
	defer r.Body.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := house.ValidateState(raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid house state")
		return
	}
	if err := api.store.SaveCurrentHouse(r.Context(), sessionToken, raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save house")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api apiServer) ensureSession(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(api.cookieName); err == nil && cookie.Value != "" {
		expiresAt := time.Now().Add(api.anonymousTTL)
		return cookie.Value, api.store.EnsureAnonymousSession(r.Context(), cookie.Value, expiresAt)
	}

	token, err := randomToken()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(api.anonymousTTL)
	if err := api.store.EnsureAnonymousSession(r.Context(), token, expiresAt); err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(api.anonymousTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	return token, nil
}

func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
