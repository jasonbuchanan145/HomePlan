package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
