package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/divijg19/Sage/internal/event"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		timestamp TEXT NOT NULL,
		type TEXT NOT NULL,
		project TEXT NOT NULL,
		data TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_events_time
	ON events(timestamp);
	`

	_, err := db.Exec(schema)
	return err
}

func (s *Store) Append(e event.Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO events (id, timestamp, type, project, data)
	VALUES (?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(
		query,
		e.ID,
		e.Timestamp.Format(time.RFC3339),
		e.Type,
		e.Project,
		string(data),
	)

	return err
}

func (s *Store) List() ([]event.Event, error) {
	query := `
	SELECT data
	FROM events
	ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}

		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) ListUntil(t time.Time) ([]event.Event, error) {
	query := `
	SELECT data
	FROM events
	WHERE timestamp <= ?
	ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, t.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}

		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}

		events = append(events, e)
	}

	return events, rows.Err()
}
