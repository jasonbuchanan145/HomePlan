package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"homeplan/api/internal/httpapi"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const integrationState = `{"schemaVersion":1,"id":"test-house","name":"Test House","floors":{"top":{"label":"Top","defaultRoom":"room-1","grid":{"columns":100,"rows":100},"rooms":[]}},"taskGroups":{},"roomTaskSets":{}}`

func TestPostgresStoreAnonymousSaveLoad(t *testing.T) {
	db := openMigratedTestDB(t)
	store := NewPostgresStore(db)
	ctx := context.Background()

	if err := store.EnsureAnonymousSession(ctx, "test-session", time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("ensure session: %v", err)
	}
	if err := store.SaveCurrentHouse(ctx, "test-session", []byte(integrationState)); err != nil {
		t.Fatalf("save house: %v", err)
	}
	state, err := store.LoadCurrentHouse(ctx, "test-session")
	if err != nil {
		t.Fatalf("load house: %v", err)
	}
	assertJSONEqual(t, integrationState, state)
}

func TestPostgresStoreDevSeedAndReset(t *testing.T) {
	db := openMigratedTestDB(t)
	store := NewPostgresStore(db)
	ctx := context.Background()

	if err := store.SaveDevUserHouse(ctx, []byte(integrationState)); err != nil {
		t.Fatalf("seed dev house: %v", err)
	}
	state, err := store.LoadDevUserHouse(ctx)
	if err != nil {
		t.Fatalf("load dev house: %v", err)
	}
	assertJSONEqual(t, integrationState, state)

	assertCount(t, db, "users", 1)
	assertCount(t, db, "houses", 1)
	assertCount(t, db, "house_members", 1)
	assertCount(t, db, "house_state", 1)
	assertCount(t, db, "house_events", 1)

	if err := store.ResetDevUserHouse(ctx); err != nil {
		t.Fatalf("reset dev house: %v", err)
	}
	assertCount(t, db, "users", 1)
	assertCount(t, db, "houses", 0)
	assertCount(t, db, "house_state", 0)
}

func TestPostgresStoreSharedIdentitySessionAndEntitlement(t *testing.T) {
	db := openMigratedTestDB(t)
	store := NewPostgresStore(db)
	ctx := context.Background()

	user, entitlement, err := store.UpsertAuthIdentity(ctx, httpapi.AuthIdentity{
		Provider:        "google",
		ProviderSubject: "google-subject-1",
		Email:           "Person@Example.com",
		EmailVerified:   true,
		DisplayName:     "Person Example",
		AvatarURL:       "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("upsert identity: %v", err)
	}
	if user.Email != "person@example.com" {
		t.Fatalf("expected lower-case email, got %s", user.Email)
	}
	if !entitlement.CanAccess || entitlement.CanUseAI {
		t.Fatalf("unexpected entitlement: %+v", entitlement)
	}

	if err := store.CreateUserSession(ctx, user.ID, "test-token-hash", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}
	session, err := store.LoadUserSession(ctx, "test-token-hash")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if session.User.ID != user.ID || !session.Entitlement.CanAccess {
		t.Fatalf("unexpected session: %+v", session)
	}

	if err := store.RevokeUserSession(ctx, "test-token-hash"); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, err := store.LoadUserSession(ctx, "test-token-hash"); err == nil {
		t.Fatal("expected revoked session to fail")
	}
}

func openMigratedTestDB(t *testing.T) *sql.DB {
	t.Helper()

	baseURL := os.Getenv("HOMEPLAN_TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("HOMEPLAN_TEST_DATABASE_URL is not set")
	}

	adminDB, err := sql.Open("pgx", baseURL)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })

	schema := fmt.Sprintf("homeplan_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`create schema ` + schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _, _ = adminDB.Exec(`drop schema if exists ` + schema + ` cascade`) })

	testURL := withSearchPath(t, baseURL, schema)
	db, err := sql.Open("pgx", testURL)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrationPaths, err := filepath.Glob(filepath.Join(repoRoot(t), "db", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	sort.Strings(migrationPaths)
	for _, migrationPath := range migrationPaths {
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", migrationPath, err)
		}
		if _, err := db.Exec(string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", migrationPath, err)
		}
	}

	return db
}

func withSearchPath(t *testing.T, rawURL string, schema string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func assertCount(t *testing.T, db *sql.DB, table string, expected int) {
	t.Helper()
	if strings.ContainsAny(table, `"; `) {
		t.Fatalf("invalid table name: %s", table)
	}
	var actual int
	if err := db.QueryRow(`select count(*) from ` + table).Scan(&actual); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if actual != expected {
		t.Fatalf("expected %s count %d, got %d", table, expected, actual)
	}
}

func assertJSONEqual(t *testing.T, expected string, actual []byte) {
	t.Helper()

	var expectedValue any
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatalf("unmarshal expected json: %v", err)
	}

	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("unmarshal actual json: %v", err)
	}

	expectedCanonical, err := json.Marshal(expectedValue)
	if err != nil {
		t.Fatalf("marshal expected json: %v", err)
	}
	actualCanonical, err := json.Marshal(actualValue)
	if err != nil {
		t.Fatalf("marshal actual json: %v", err)
	}

	if string(expectedCanonical) != string(actualCanonical) {
		t.Fatalf("json mismatch\nexpected: %s\nactual:   %s", expectedCanonical, actualCanonical)
	}
}
