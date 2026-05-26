package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"homeplan/api/internal/store"
)

const validState = `{"schemaVersion":1,"id":"test-house","name":"Test House","floors":{"top":{"label":"Top","defaultRoom":"room-1","grid":{"columns":100,"rows":100},"rooms":[]}},"taskGroups":{},"roomTaskSets":{}}`

func TestHealth(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestOpenAPIDocsAvailable(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d: %s", spec.Code, spec.Body.String())
	}
	if !strings.Contains(spec.Body.String(), `"get-health"`) {
		t.Fatalf("expected health operation in openapi spec, got %s", spec.Body.String())
	}

	docs := httptest.NewRecorder()
	router.ServeHTTP(docs, httptest.NewRequest(http.MethodGet, "/api/docs", nil))
	if docs.Code != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", docs.Code)
	}
}

func TestSaveAndLoadCurrentHouse(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)

	save := httptest.NewRecorder()
	router.ServeHTTP(save, httptest.NewRequest(http.MethodPut, "/api/house/current", bytes.NewBufferString(validState)))
	if save.Code != http.StatusNoContent {
		t.Fatalf("expected save 204, got %d: %s", save.Code, save.Body.String())
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/house/current", nil)
	for _, cookie := range save.Result().Cookies() {
		loadReq.AddCookie(cookie)
	}
	load := httptest.NewRecorder()
	router.ServeHTTP(load, loadReq)

	if load.Code != http.StatusOK {
		t.Fatalf("expected load 200, got %d: %s", load.Code, load.Body.String())
	}
	if load.Body.String() != validState {
		t.Fatalf("unexpected body: %s", load.Body.String())
	}
}

func TestDeleteCurrentHouse(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)

	save := httptest.NewRecorder()
	router.ServeHTTP(save, httptest.NewRequest(http.MethodPut, "/api/house/current", bytes.NewBufferString(validState)))
	if save.Code != http.StatusNoContent {
		t.Fatalf("expected save 204, got %d: %s", save.Code, save.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/house/current", nil)
	for _, cookie := range save.Result().Cookies() {
		deleteReq.AddCookie(cookie)
	}
	reset := httptest.NewRecorder()
	router.ServeHTTP(reset, deleteReq)
	if reset.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", reset.Code, reset.Body.String())
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/house/current", nil)
	for _, cookie := range save.Result().Cookies() {
		loadReq.AddCookie(cookie)
	}
	load := httptest.NewRecorder()
	router.ServeHTTP(load, loadReq)
	if load.Code != http.StatusNotFound {
		t.Fatalf("expected load after delete 404, got %d: %s", load.Code, load.Body.String())
	}
}

func TestInvalidJSON(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/house/current", bytes.NewBufferString(`{"schemaVersion":1}`)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCurrentHouseNotFoundBeforeSave(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/house/current", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestDevEndpointsDisabled(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/dev/users/user-1/house/current", bytes.NewBufferString(validState)))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestDevSeedLoadAndReset(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, true)

	before := httptest.NewRecorder()
	router.ServeHTTP(before, httptest.NewRequest(http.MethodGet, "/api/house/current", nil))
	if before.Code != http.StatusNotFound {
		t.Fatalf("expected initial 404, got %d", before.Code)
	}

	seed := httptest.NewRecorder()
	router.ServeHTTP(seed, httptest.NewRequest(http.MethodPost, "/api/dev/users/user-1/house/current", bytes.NewBufferString(validState)))
	if seed.Code != http.StatusOK {
		t.Fatalf("expected seed 200, got %d: %s", seed.Code, seed.Body.String())
	}

	load := httptest.NewRecorder()
	router.ServeHTTP(load, httptest.NewRequest(http.MethodGet, "/api/house/current", nil))
	if load.Code != http.StatusOK {
		t.Fatalf("expected load 200, got %d: %s", load.Code, load.Body.String())
	}
	if load.Body.String() != validState {
		t.Fatalf("unexpected body: %s", load.Body.String())
	}

	reset := httptest.NewRecorder()
	router.ServeHTTP(reset, httptest.NewRequest(http.MethodDelete, "/api/dev/users/user-1/house/current", nil))
	if reset.Code != http.StatusOK {
		t.Fatalf("expected reset 200, got %d: %s", reset.Code, reset.Body.String())
	}

	after := httptest.NewRecorder()
	router.ServeHTTP(after, httptest.NewRequest(http.MethodGet, "/api/house/current", nil))
	if after.Code != http.StatusNotFound {
		t.Fatalf("expected final 404, got %d", after.Code)
	}
}

func TestGetMeAnonymous(t *testing.T) {
	router := NewRouter(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/me", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"authenticated":false`) {
		t.Fatalf("expected anonymous me response, got %s", recorder.Body.String())
	}
}

func TestGoogleCallbackCreatesSessionAndMe(t *testing.T) {
	router := NewRouterWithAuth(store.NewMemoryStore(), "homeplan_session", 14*24*time.Hour, false, AuthConfig{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		BaseURL:            "http://example.test",
		SessionCookieName:  "homeplan_auth",
		SessionTTL:         time.Hour,
	}, fakeOAuthClient{})

	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/api/auth/google/start", nil))
	if start.Code != http.StatusFound {
		t.Fatalf("expected start redirect, got %d", start.Code)
	}

	var stateCookie *http.Cookie
	var sessionCookie *http.Cookie
	for _, cookie := range start.Result().Cookies() {
		if cookie.Name == "homeplan_auth_oauth_state" {
			stateCookie = cookie
		}
		if cookie.Name == "homeplan_auth_oauth_google_session" {
			sessionCookie = cookie
		}
	}
	if stateCookie == nil {
		t.Fatal("expected oauth state cookie")
	}
	if sessionCookie == nil {
		t.Fatal("expected oauth session cookie")
	}

	callback := httptest.NewRecorder()
	callbackReq := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=test-code&state="+stateCookie.Value, nil)
	callbackReq.AddCookie(stateCookie)
	callbackReq.AddCookie(sessionCookie)
	router.ServeHTTP(callback, callbackReq)
	if callback.Code != http.StatusFound {
		t.Fatalf("expected callback redirect, got %d: %s", callback.Code, callback.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	for _, cookie := range callback.Result().Cookies() {
		meReq.AddCookie(cookie)
	}
	me := httptest.NewRecorder()
	router.ServeHTTP(me, meReq)
	if me.Code != http.StatusOK {
		t.Fatalf("expected me 200, got %d", me.Code)
	}
	body := me.Body.String()
	if !strings.Contains(body, `"authenticated":true`) || !strings.Contains(body, `"canUseAI":false`) {
		t.Fatalf("unexpected me response: %s", body)
	}

	logout := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	for _, cookie := range callback.Result().Cookies() {
		logoutReq.AddCookie(cookie)
	}
	router.ServeHTTP(logout, logoutReq)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("expected logout 204, got %d", logout.Code)
	}
}

type fakeOAuthClient struct{}

func (fakeOAuthClient) Name() string {
	return "google"
}

func (fakeOAuthClient) BeginAuth(state string) (AuthProviderSession, error) {
	if state == "" {
		return AuthProviderSession{}, context.Canceled
	}
	return AuthProviderSession{
		AuthURL: "https://accounts.example.test/oauth?state=" + state,
		Session: "fake-session",
	}, nil
}

func (fakeOAuthClient) CompleteAuth(session string, params url.Values) (AuthIdentity, error) {
	if session != "fake-session" || params.Get("code") != "test-code" {
		return AuthIdentity{}, context.Canceled
	}
	return AuthIdentity{
		Provider:        "google",
		ProviderSubject: "google-user-1",
		Email:           "person@example.com",
		EmailVerified:   true,
		DisplayName:     "Person Example",
		AvatarURL:       "https://example.com/avatar.png",
	}, nil
}
