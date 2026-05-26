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

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type HouseStore interface {
	EnsureAnonymousSession(ctx context.Context, token string, expiresAt time.Time) error
	LoadCurrentHouse(ctx context.Context, sessionToken string) (json.RawMessage, error)
	SaveCurrentHouse(ctx context.Context, sessionToken string, state json.RawMessage) error
	DeleteCurrentHouse(ctx context.Context, sessionToken string) error
	LoadUserHouse(ctx context.Context, userID string) (json.RawMessage, error)
	SaveUserHouse(ctx context.Context, userID string, state json.RawMessage) error
	DeleteUserHouse(ctx context.Context, userID string) error
	LoadDevUserHouse(ctx context.Context) (json.RawMessage, error)
	SaveDevUserHouse(ctx context.Context, state json.RawMessage) error
	ResetDevUserHouse(ctx context.Context) error
	UpsertAuthIdentity(ctx context.Context, identity AuthIdentity) (AuthUser, AppEntitlement, error)
	CreateUserSession(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	LoadUserSession(ctx context.Context, tokenHash string) (SignedInSession, error)
	RevokeUserSession(ctx context.Context, tokenHash string) error
}

func NewRouter(store HouseStore, cookieName string, anonymousTTL time.Duration, devMode bool) http.Handler {
	return NewRouterWithAuth(store, cookieName, anonymousTTL, devMode, AuthConfig{}, nil)
}

func NewRouterWithAuth(store HouseStore, cookieName string, anonymousTTL time.Duration, devMode bool, authConfig AuthConfig, authProvider AuthProvider) http.Handler {
	mux := http.NewServeMux()
	if authConfig.SessionCookieName == "" {
		authConfig.SessionCookieName = "homeplan_auth"
	}
	if authConfig.SessionTTL == 0 {
		authConfig.SessionTTL = 30 * 24 * time.Hour
	}
	if authProvider == nil && authConfig.GoogleClientID != "" && authConfig.GoogleClientSecret != "" && authConfig.BaseURL != "" {
		authProvider = newGoogleAuthProvider(authConfig)
	}

	api := apiServer{store: store, cookieName: cookieName, anonymousTTL: anonymousTTL, devMode: devMode, authConfig: authConfig, authProvider: authProvider}
	docsConfig := huma.DefaultConfig("HomePlan API", "0.1.0")
	docsConfig.DocsPath = "/api/docs"
	docsConfig.OpenAPIPath = "/api/openapi"
	api.registerHumaRoutes(humago.New(mux, docsConfig))

	mux.HandleFunc("GET /api/me", api.getMe)
	mux.HandleFunc("GET /api/auth/google/start", api.startGoogleAuth)
	mux.HandleFunc("GET /api/auth/google/callback", api.finishGoogleAuth)
	mux.HandleFunc("POST /api/auth/logout", api.logout)
	mux.HandleFunc("GET /api/house/current", api.getCurrentHouse)
	mux.HandleFunc("PUT /api/house/current", api.putCurrentHouse)
	mux.HandleFunc("DELETE /api/house/current", api.deleteCurrentHouse)
	mux.HandleFunc("POST /api/dev/users/user-1/house/current", api.seedDevUserHouse)
	mux.HandleFunc("DELETE /api/dev/users/user-1/house/current", api.resetDevUserHouse)
	return mux
}

type apiServer struct {
	store        HouseStore
	cookieName   string
	anonymousTTL time.Duration
	devMode      bool
	authConfig   AuthConfig
	authProvider AuthProvider
}

type healthOutput struct {
	Body map[string]string
}

func (api apiServer) registerHumaRoutes(humaAPI huma.API) {
	huma.Register(humaAPI, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Health check",
		Tags:        []string{"system"},
	}, func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		return &healthOutput{Body: map[string]string{"status": "ok"}}, nil
	})
	api.documentNetHTTPRoutes(humaAPI.OpenAPI())
}

func (api apiServer) documentNetHTTPRoutes(openAPI *huma.OpenAPI) {
	if openAPI.Paths == nil {
		openAPI.Paths = map[string]*huma.PathItem{}
	}
	openAPI.Paths["/api/me"] = &huma.PathItem{
		Get: &huma.Operation{
			OperationID: "get-me",
			Tags:        []string{"auth"},
			Summary:     "Get current account and HomePlan entitlement",
			Responses:   okResponses("Account state returned"),
		},
	}
	openAPI.Paths["/api/auth/google/start"] = &huma.PathItem{
		Get: &huma.Operation{
			OperationID: "start-google-auth",
			Tags:        []string{"auth"},
			Summary:     "Start Google sign-in",
			Responses: map[string]*huma.Response{
				"302": {Description: "Redirects to Google"},
				"503": {Description: "Google sign-in is not configured"},
			},
		},
	}
	openAPI.Paths["/api/auth/google/callback"] = &huma.PathItem{
		Get: &huma.Operation{
			OperationID: "finish-google-auth",
			Tags:        []string{"auth"},
			Summary:     "Finish Google sign-in callback",
			Responses: map[string]*huma.Response{
				"302": {Description: "Creates a signed-in session and redirects home"},
				"400": {Description: "Invalid callback state or code"},
				"502": {Description: "Provider profile could not be loaded"},
			},
		},
	}
	openAPI.Paths["/api/auth/logout"] = &huma.PathItem{
		Post: &huma.Operation{
			OperationID: "logout",
			Tags:        []string{"auth"},
			Summary:     "Revoke current signed-in session",
			Responses: map[string]*huma.Response{
				"204": {Description: "Session cleared"},
			},
		},
	}
	openAPI.Paths["/api/house/current"] = &huma.PathItem{
		Get: &huma.Operation{
			OperationID: "get-current-house",
			Tags:        []string{"house"},
			Summary:     "Load the current anonymous or signed-in house state",
			Responses: map[string]*huma.Response{
				"200": {Description: "House JSON state returned"},
				"403": {Description: "Signed-in account is not entitled to HomePlan"},
				"404": {Description: "No house has been saved"},
			},
		},
		Put: &huma.Operation{
			OperationID: "save-current-house",
			Tags:        []string{"house"},
			Summary:     "Save the current anonymous or signed-in house state",
			Responses: map[string]*huma.Response{
				"204": {Description: "House saved"},
				"400": {Description: "Invalid house JSON state"},
				"403": {Description: "Signed-in account is not entitled to HomePlan"},
			},
		},
		Delete: &huma.Operation{
			OperationID: "delete-current-house",
			Tags:        []string{"house"},
			Summary:     "Delete the current anonymous or signed-in house state",
			Responses: map[string]*huma.Response{
				"204": {Description: "House deleted"},
				"403": {Description: "Signed-in account is not entitled to HomePlan"},
			},
		},
	}
	if api.devMode {
		openAPI.Paths["/api/dev/users/user-1/house/current"] = &huma.PathItem{
			Post: &huma.Operation{
				OperationID: "seed-dev-user-house",
				Tags:        []string{"dev"},
				Summary:     "Seed deterministic dev user house state",
				Responses:   okResponses("House seeded"),
			},
			Delete: &huma.Operation{
				OperationID: "reset-dev-user-house",
				Tags:        []string{"dev"},
				Summary:     "Reset deterministic dev user house state",
				Responses:   okResponses("House reset"),
			},
		}
	}
}

func okResponses(description string) map[string]*huma.Response {
	return map[string]*huma.Response{
		"200": {Description: description},
	}
}

func (api apiServer) getCurrentHouse(w http.ResponseWriter, r *http.Request) {
	if api.devMode {
		state, err := api.store.LoadDevUserHouse(r.Context())
		if errors.Is(err, house.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no house saved for dev user 1")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load dev house")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(state)
		return
	}

	if session, ok := api.currentUserSession(r); ok {
		if !session.Entitlement.CanAccess {
			writeError(w, http.StatusForbidden, "homeplan access is not enabled for this account")
			return
		}
		state, err := api.store.LoadUserHouse(r.Context(), session.User.ID)
		if errors.Is(err, house.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no house saved for this user")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load user house")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(state)
		return
	}

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

	if api.devMode {
		if err := api.store.SaveDevUserHouse(r.Context(), raw); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save dev house")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if session, ok := api.currentUserSession(r); ok {
		if !session.Entitlement.CanAccess {
			writeError(w, http.StatusForbidden, "homeplan access is not enabled for this account")
			return
		}
		if err := api.store.SaveUserHouse(r.Context(), session.User.ID, raw); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save user house")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	sessionToken, err := api.ensureSession(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create anonymous session")
		return
	}
	if err := api.store.SaveCurrentHouse(r.Context(), sessionToken, raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save house")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api apiServer) deleteCurrentHouse(w http.ResponseWriter, r *http.Request) {
	if api.devMode {
		if err := api.store.ResetDevUserHouse(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "could not reset dev house")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if session, ok := api.currentUserSession(r); ok {
		if !session.Entitlement.CanAccess {
			writeError(w, http.StatusForbidden, "homeplan access is not enabled for this account")
			return
		}
		if err := api.store.DeleteUserHouse(r.Context(), session.User.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete user house")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	sessionToken, err := api.ensureSession(w, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create anonymous session")
		return
	}
	if err := api.store.DeleteCurrentHouse(r.Context(), sessionToken); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete house")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api apiServer) getMe(w http.ResponseWriter, r *http.Request) {
	if session, ok := api.currentUserSession(r); ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": true,
			"user":          session.User,
			"apps": map[string]AppEntitlement{
				homeplanAppKey: session.Entitlement,
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": false,
		"user":          nil,
		"apps": map[string]AppEntitlement{
			homeplanAppKey: {},
		},
	})
}

func (api apiServer) startGoogleAuth(w http.ResponseWriter, r *http.Request) {
	if api.authProvider == nil {
		writeError(w, http.StatusServiceUnavailable, "google sign-in is not configured")
		return
	}
	state, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start sign-in")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.oauthStateCookieName(),
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		Expires:  time.Now().Add(5 * time.Minute),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})

	providerSession, err := api.authProvider.BeginAuth(state)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not start google sign-in")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.oauthSessionCookieName(),
		Value:    encodeOAuthSessionCookie(providerSession.Session),
		Path:     "/",
		MaxAge:   300,
		Expires:  time.Now().Add(5 * time.Minute),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	http.Redirect(w, r, providerSession.AuthURL, http.StatusFound)
}

func (api apiServer) finishGoogleAuth(w http.ResponseWriter, r *http.Request) {
	if api.authProvider == nil {
		writeError(w, http.StatusServiceUnavailable, "google sign-in is not configured")
		return
	}
	stateCookie, err := r.Cookie(api.oauthStateCookieName())
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		writeError(w, http.StatusBadRequest, "invalid sign-in state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing sign-in code")
		return
	}

	sessionCookie, err := r.Cookie(api.oauthSessionCookieName())
	if err != nil || sessionCookie.Value == "" {
		writeError(w, http.StatusBadRequest, "missing sign-in session")
		return
	}
	providerSession, err := decodeOAuthSessionCookie(sessionCookie.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sign-in session")
		return
	}
	identity, err := api.authProvider.CompleteAuth(providerSession, r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not load google profile")
		return
	}
	user, _, err := api.store.UpsertAuthIdentity(r.Context(), identity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save signed-in user")
		return
	}

	sessionToken, err := randomToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	expiresAt := time.Now().Add(api.authConfig.SessionTTL)
	if err := api.store.CreateUserSession(r.Context(), user.ID, tokenHash(sessionToken), expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.authConfig.SessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(api.authConfig.SessionTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     api.oauthStateCookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     api.oauthSessionCookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (api apiServer) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(api.authConfig.SessionCookieName); err == nil && cookie.Value != "" {
		_ = api.store.RevokeUserSession(r.Context(), tokenHash(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.authConfig.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (api apiServer) seedDevUserHouse(w http.ResponseWriter, r *http.Request) {
	if !api.devMode {
		writeError(w, http.StatusNotFound, "not found")
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
	if err := api.store.SaveDevUserHouse(r.Context(), raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not seed dev house")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "seeded"})
}

func (api apiServer) resetDevUserHouse(w http.ResponseWriter, r *http.Request) {
	if !api.devMode {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err := api.store.ResetDevUserHouse(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reset dev house")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
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

func (api apiServer) currentUserSession(r *http.Request) (SignedInSession, bool) {
	cookie, err := r.Cookie(api.authConfig.SessionCookieName)
	if err != nil || cookie.Value == "" {
		return SignedInSession{}, false
	}
	session, err := api.store.LoadUserSession(r.Context(), tokenHash(cookie.Value))
	return session, err == nil
}

func (api apiServer) oauthStateCookieName() string {
	return api.authConfig.SessionCookieName + "_oauth_state"
}

func (api apiServer) oauthSessionCookieName() string {
	providerName := "google"
	if api.authProvider != nil && api.authProvider.Name() != "" {
		providerName = api.authProvider.Name()
	}
	return api.authConfig.SessionCookieName + "_oauth_" + providerName + "_session"
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
