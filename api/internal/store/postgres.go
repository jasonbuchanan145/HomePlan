package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"homeplan/api/internal/house"
)

type PostgresStore struct {
	db *sql.DB
}

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
