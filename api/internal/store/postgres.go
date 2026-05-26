package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"homeplan/api/internal/house"
	"homeplan/api/internal/identity"
)

type PostgresStore struct {
	db *sql.DB
}

const devUserEmail = "dev-user-1@homeplan.local"

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) EnsureAnonymousSession(ctx context.Context, token string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		insert into anonymous_sessions (token, expires_at)
		values ($1, $2)
		on conflict (token) do update set expires_at = excluded.expires_at
	`, token, expiresAt)
	return err
}

func (s *PostgresStore) LoadCurrentHouse(ctx context.Context, sessionToken string) (json.RawMessage, error) {
	var state []byte
	err := s.db.QueryRowContext(ctx, `
		select hs.state
		from house_state hs
		join houses h on h.id = hs.house_id
		join anonymous_sessions s on s.id = h.anonymous_session_id
		where s.token = $1
		order by hs.updated_at desc
		limit 1
	`, sessionToken).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, house.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(state), nil
}

func (s *PostgresStore) SaveCurrentHouse(ctx context.Context, sessionToken string, state json.RawMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sessionID string
	if err := tx.QueryRowContext(ctx, `select id from anonymous_sessions where token = $1`, sessionToken).Scan(&sessionID); err != nil {
		return err
	}

	var houseID string
	err = tx.QueryRowContext(ctx, `
		select id from houses
		where anonymous_session_id = $1
		order by created_at
		limit 1
	`, sessionID).Scan(&houseID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, `
			insert into houses (anonymous_session_id, name, expires_at)
			values ($1, 'Home repair plan', now() + interval '14 days')
			returning id
		`, sessionID).Scan(&houseID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_state (house_id, state)
		values ($1, $2)
		on conflict (house_id) do update set state = excluded.state, updated_at = now()
	`, houseID, []byte(state)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_events (house_id, actor_type, event_type, payload)
		values ($1, 'anonymous_session', 'house_state_saved', jsonb_build_object('source', 'web'))
	`, houseID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) DeleteCurrentHouse(ctx context.Context, sessionToken string) error {
	_, err := s.db.ExecContext(ctx, `
		delete from houses
		where anonymous_session_id = (
			select id from anonymous_sessions where token = $1
		)
	`, sessionToken)
	return err
}

func (s *PostgresStore) LoadUserHouse(ctx context.Context, userID string) (json.RawMessage, error) {
	var state []byte
	err := s.db.QueryRowContext(ctx, `
		select hs.state
		from house_state hs
		join houses h on h.id = hs.house_id
		where h.owner_user_id = $1
		order by hs.updated_at desc
		limit 1
	`, userID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, house.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(state), nil
}

func (s *PostgresStore) SaveUserHouse(ctx context.Context, userID string, state json.RawMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	houseID, err := ensureUserHouse(ctx, tx, userID)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_state (house_id, state)
		values ($1, $2)
		on conflict (house_id) do update set state = excluded.state, updated_at = now()
	`, houseID, []byte(state)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_events (house_id, actor_type, actor_user_id, event_type, payload)
		values ($1, 'user', $2, 'house_state_saved', jsonb_build_object('source', 'web'))
	`, houseID, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) DeleteUserHouse(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `delete from houses where owner_user_id = $1`, userID)
	return err
}

func (s *PostgresStore) UpsertAuthIdentity(ctx context.Context, authIdentity identity.AuthIdentity) (identity.AuthUser, identity.AppEntitlement, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return identity.AuthUser{}, identity.AppEntitlement{}, err
	}
	defer tx.Rollback()

	email := strings.ToLower(authIdentity.Email)
	displayName := authIdentity.DisplayName
	if displayName == "" {
		displayName = email
	}

	var userID string
	if err := tx.QueryRowContext(ctx, `
		insert into users (email, primary_email, display_name, avatar_url, last_login_at)
		values ($1, $1, $2, $3, now())
		on conflict (email) do update set
			primary_email = coalesce(users.primary_email, excluded.primary_email),
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			last_login_at = now(),
			updated_at = now()
		returning id
	`, email, displayName, authIdentity.AvatarURL).Scan(&userID); err != nil {
		return identity.AuthUser{}, identity.AppEntitlement{}, err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into user_identities (user_id, provider, provider_subject, email, email_verified, display_name, avatar_url)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (provider, provider_subject) do update set
			user_id = excluded.user_id,
			email = excluded.email,
			email_verified = excluded.email_verified,
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			updated_at = now()
	`, userID, authIdentity.Provider, authIdentity.ProviderSubject, email, authIdentity.EmailVerified, displayName, authIdentity.AvatarURL); err != nil {
		return identity.AuthUser{}, identity.AppEntitlement{}, err
	}

	var entitlement identity.AppEntitlement
	if err := tx.QueryRowContext(ctx, `
		insert into app_entitlements (user_id, app_key, can_access, can_use_ai)
		values ($1, $2, true, false)
		on conflict (user_id, app_key) do update set updated_at = now()
		returning can_access, can_use_ai
	`, userID, homeplanAppKey()).Scan(&entitlement.CanAccess, &entitlement.CanUseAI); err != nil {
		return identity.AuthUser{}, identity.AppEntitlement{}, err
	}

	if err := tx.Commit(); err != nil {
		return identity.AuthUser{}, identity.AppEntitlement{}, err
	}
	return identity.AuthUser{ID: userID, Email: email, DisplayName: displayName, AvatarURL: authIdentity.AvatarURL}, entitlement, nil
}

func (s *PostgresStore) CreateUserSession(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		insert into user_sessions (user_id, token_hash, expires_at)
		values ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func (s *PostgresStore) LoadUserSession(ctx context.Context, tokenHash string) (identity.SignedInSession, error) {
	var session identity.SignedInSession
	err := s.db.QueryRowContext(ctx, `
		select u.id, coalesce(u.primary_email, u.email, ''), coalesce(u.display_name, ''), coalesce(u.avatar_url, ''),
			coalesce(e.can_access, false), coalesce(e.can_use_ai, false)
		from user_sessions s
		join users u on u.id = s.user_id
		left join app_entitlements e on e.user_id = u.id and e.app_key = $2
		where s.token_hash = $1
			and s.revoked_at is null
			and s.expires_at > now()
	`, tokenHash, homeplanAppKey()).Scan(
		&session.User.ID,
		&session.User.Email,
		&session.User.DisplayName,
		&session.User.AvatarURL,
		&session.Entitlement.CanAccess,
		&session.Entitlement.CanUseAI,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.SignedInSession{}, house.ErrNotFound
	}
	if err != nil {
		return identity.SignedInSession{}, err
	}
	_, _ = s.db.ExecContext(ctx, `update user_sessions set last_seen_at = now() where token_hash = $1`, tokenHash)
	return session, nil
}

func (s *PostgresStore) RevokeUserSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `update user_sessions set revoked_at = now() where token_hash = $1`, tokenHash)
	return err
}

func (s *PostgresStore) LoadDevUserHouse(ctx context.Context) (json.RawMessage, error) {
	var state []byte
	err := s.db.QueryRowContext(ctx, `
		select hs.state
		from house_state hs
		join houses h on h.id = hs.house_id
		join users u on u.id = h.owner_user_id
		where u.email = $1
		order by hs.updated_at desc
		limit 1
	`, devUserEmail).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, house.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(state), nil
}

func (s *PostgresStore) SaveDevUserHouse(ctx context.Context, state json.RawMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	houseID, userID, err := ensureDevUserHouse(ctx, tx)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_state (house_id, state)
		values ($1, $2)
		on conflict (house_id) do update set state = excluded.state, updated_at = now()
	`, houseID, []byte(state)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_events (house_id, actor_type, actor_user_id, event_type, payload)
		values ($1, 'user', $2, 'dev_house_seeded', jsonb_build_object('source', 'seed-dev-house.ps1'))
	`, houseID, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStore) ResetDevUserHouse(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		delete from houses
		where owner_user_id = (select id from users where email = $1)
	`, devUserEmail)
	return err
}

func ensureDevUserHouse(ctx context.Context, tx *sql.Tx) (houseID string, userID string, err error) {
	if err := tx.QueryRowContext(ctx, `
		insert into users (email, display_name)
		values ($1, 'Dev User 1')
		on conflict (email) do update set display_name = excluded.display_name, updated_at = now()
		returning id
	`, devUserEmail).Scan(&userID); err != nil {
		return "", "", err
	}

	err = tx.QueryRowContext(ctx, `
		select id from houses
		where owner_user_id = $1
		order by created_at
		limit 1
	`, userID).Scan(&houseID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, `
			insert into houses (owner_user_id, name)
			values ($1, 'Home repair plan')
			returning id
		`, userID).Scan(&houseID); err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_members (house_id, user_id, role)
		values ($1, $2, 'owner')
		on conflict (house_id, user_id) do update set role = excluded.role
	`, houseID, userID); err != nil {
		return "", "", err
	}

	return houseID, userID, nil
}

func ensureUserHouse(ctx context.Context, tx *sql.Tx, userID string) (string, error) {
	var houseID string
	err := tx.QueryRowContext(ctx, `
		select id from houses
		where owner_user_id = $1
		order by created_at
		limit 1
	`, userID).Scan(&houseID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, `
			insert into houses (owner_user_id, name)
			values ($1, 'Home repair plan')
			returning id
		`, userID).Scan(&houseID); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		insert into house_members (house_id, user_id, role)
		values ($1, $2, 'owner')
		on conflict (house_id, user_id) do update set role = excluded.role
	`, houseID, userID); err != nil {
		return "", err
	}
	return houseID, nil
}

func homeplanAppKey() string {
	return "homeplan"
}
