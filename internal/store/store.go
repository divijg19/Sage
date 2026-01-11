package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/divijg19/sage/internal/event"
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

	// Reduce "database is locked" errors under concurrent access (e.g. hooks + reads).
	// This is a connection-local setting.
	_, _ = db.Exec(`PRAGMA busy_timeout = 5000;`)

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	// v2 schema adds a numeric, user-facing id (seq).
	// We migrate existing v1 DBs by copying events in stable chronological order.
	if ok, err := hasColumn(db, "events", "seq"); err != nil {
		return err
	} else if ok {
		return ensureIndexes(db)
	}

	// If this is a v1 DB, migration must be transactional and fail-loud.
	// We should never drop/rename the existing table unless the copy succeeds.
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create v2 table and copy data.
	createV2 := `
	CREATE TABLE IF NOT EXISTS events_v2 (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		id TEXT NOT NULL UNIQUE,
		timestamp TEXT NOT NULL,
		type TEXT NOT NULL,
		project TEXT NOT NULL,
		data TEXT NOT NULL
	);
	`
	if _, err := tx.Exec(createV2); err != nil {
		return err
	}

	// Copy from v1 if it exists. Order by timestamp then id for deterministic seq assignment.
	if ok, err := tableExists(db, "events"); err != nil {
		return err
	} else if ok {
		copySQL := `
		INSERT INTO events_v2 (id, timestamp, type, project, data)
		SELECT id, timestamp, type, project, data
		FROM events
		ORDER BY timestamp ASC, id ASC;
		`
		if _, err := tx.Exec(copySQL); err != nil {
			return err
		}
	}

	// Swap tables.
	// If v1 didn't exist, events_v2 is our empty table.
	if _, err := tx.Exec(`DROP TABLE IF EXISTS events;`); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE events_v2 RENAME TO events;`); err != nil {
		return err
	}

	schema := `
	CREATE INDEX IF NOT EXISTS idx_events_time
	ON events(timestamp);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_events_id
	ON events(id);
	CREATE INDEX IF NOT EXISTS idx_events_project
	ON events(project);
	CREATE INDEX IF NOT EXISTS idx_events_project_seq
	ON events(project, seq);
	`
	if _, err := tx.Exec(schema); err != nil {
		return err
	}

	return tx.Commit()
}

func tableExists(db *sql.DB, table string) (bool, error) {
	row := db.QueryRow(`SELECT 1 FROM sqlite_master WHERE type='table' AND name = ? LIMIT 1;`, table)
	var one int
	if err := row.Scan(&one); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func hasColumn(db *sql.DB, table string, column string) (bool, error) {
	q := fmt.Sprintf("PRAGMA table_info(%s);", table)
	rows, err := db.Query(q)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func ensureIndexes(db *sql.DB) error {
	schema := `
	CREATE INDEX IF NOT EXISTS idx_events_time
	ON events(timestamp);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_events_id
	ON events(id);
	CREATE INDEX IF NOT EXISTS idx_events_project
	ON events(project);
	CREATE INDEX IF NOT EXISTS idx_events_project_seq
	ON events(project, seq);
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
		e.Kind,
		e.Project,
		string(data),
	)

	return err
}

func (s *Store) List() ([]event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	ORDER BY seq ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var seq int64
		var raw string
		if err := rows.Scan(&seq, &raw); err != nil {
			return nil, err
		}

		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		e.Seq = seq

		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) ListByProject(project string) ([]event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	WHERE project = ?
	ORDER BY seq ASC
	`

	rows, err := s.db.Query(query, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var seq int64
		var raw string
		if err := rows.Scan(&seq, &raw); err != nil {
			return nil, err
		}

		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		e.Seq = seq
		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) ListUntil(t time.Time) ([]event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	WHERE timestamp <= ?
	ORDER BY seq ASC
	`

	rows, err := s.db.Query(query, t.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var seq int64
		var raw string
		if err := rows.Scan(&seq, &raw); err != nil {
			return nil, err
		}

		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		e.Seq = seq

		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) ListUntilByProject(t time.Time, project string) ([]event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	WHERE timestamp <= ? AND project = ?
	ORDER BY seq ASC
	`

	rows, err := s.db.Query(query, t.Format(time.RFC3339), project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event

	for rows.Next() {
		var seq int64
		var raw string
		if err := rows.Scan(&seq, &raw); err != nil {
			return nil, err
		}

		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		e.Seq = seq
		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) Latest() (*event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	ORDER BY seq DESC
	LIMIT 1
	`

	var seq int64
	var raw string
	err := s.db.QueryRow(query).Scan(&seq, &raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var e event.Event
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return nil, err
	}
	e.Seq = seq

	return &e, nil
}

func (s *Store) LatestByProject(project string) (*event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	WHERE project = ?
	ORDER BY seq DESC
	LIMIT 1
	`

	var seq int64
	var raw string
	err := s.db.QueryRow(query, project).Scan(&seq, &raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var e event.Event
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return nil, err
	}
	e.Seq = seq
	return &e, nil
}

func (s *Store) ListProjects() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT project FROM events WHERE project IS NOT NULL AND project != '' ORDER BY project ASC;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Count() (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM events;`)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) GetBySeq(seq int64) (*event.Event, error) {
	query := `
	SELECT seq, data
	FROM events
	WHERE seq = ?
	LIMIT 1
	`

	var gotSeq int64
	var raw string
	err := s.db.QueryRow(query, seq).Scan(&gotSeq, &raw)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var e event.Event
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		return nil, err
	}
	e.Seq = gotSeq
	return &e, nil
}

func (s *Store) UpdateTagsBySeq(seq int64, tags []string) error {
	e, err := s.GetBySeq(seq)
	if err != nil {
		return err
	}
	if e == nil {
		return fmt.Errorf("no entry with id %d", seq)
	}

	e.Tags = tags
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`UPDATE events SET data = ? WHERE seq = ?;`, string(b), seq)
	return err
}

// ReadEventsFromDB reads events from a DB file without migrating it.
// This is used for importing legacy per-directory stores.
func ReadEventsFromDB(path string) ([]event.Event, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT data FROM events ORDER BY timestamp ASC, id ASC;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []event.Event
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "constraint failed")
}

// ImportEvents appends events into this store in deterministic order.
// Duplicate IDs are skipped.
func (s *Store) ImportEvents(events []event.Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	// Sort for deterministic seq assignment.
	sort.Slice(events, func(i, j int) bool {
		if events[i].Timestamp.Equal(events[j].Timestamp) {
			return events[i].ID < events[j].ID
		}
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	inserted := 0
	for _, e := range events {
		if err := s.Append(e); err != nil {
			// If already present, skip.
			if isUniqueConstraintErr(err) {
				continue
			}
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}
