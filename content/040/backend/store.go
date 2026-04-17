package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store owns the SQLite connection and provides all persistence operations.
type Store struct {
	db *sql.DB
	// writeMu serializes write transactions because modernc.org/sqlite still
	// uses a single writer at a time and we want to batch cleanly.
	writeMu sync.Mutex
}

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id    TEXT NOT NULL,
    host          TEXT NOT NULL,
    session_user  TEXT NOT NULL,
    session_uid   INTEGER NOT NULL,
    os            TEXT,
    kernel        TEXT,
    started_at    INTEGER NOT NULL,
    last_seen_at  INTEGER NOT NULL,
    closed_at     INTEGER,
    UNIQUE(machine_id, session_user, started_at)
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(session_user);
CREATE INDEX IF NOT EXISTS idx_sessions_machine ON sessions(machine_id);
CREATE INDEX IF NOT EXISTS idx_sessions_last_seen ON sessions(last_seen_at);

CREATE TABLE IF NOT EXISTS samples (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id     INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    window_start   INTEGER NOT NULL,
    window_end     INTEGER NOT NULL,
    interval_sec   REAL,
    keys_pressed   INTEGER,
    keys_released  INTEGER,
    clicks         INTEGER,
    mouse_moves    INTEGER,
    scrolls        INTEGER,
    rx_bytes       INTEGER,
    tx_bytes       INTEGER,
    net_calls      INTEGER,
    cpu_used_pct   REAL,
    mem_used_pct   REAL,
    load1          REAL,
    load5          REAL,
    load15         REAL,
    num_procs      INTEGER,
    per_user_net   TEXT   -- JSON
);
CREATE INDEX IF NOT EXISTS idx_samples_session ON samples(session_id, window_end);
CREATE INDEX IF NOT EXISTS idx_samples_window ON samples(window_end);
`

// IdleGap is how long a session can go without a sample before we close it
// and start a new one on next ingest.
const IdleGap = 5 * time.Minute

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // modernc is fine with more, but simplifies writes
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// --- types ---

type PerUserNet struct {
	UID      int    `json:"uid"`
	User     string `json:"user"`
	RxBytes  int64  `json:"rx_bytes"`
	TxBytes  int64  `json:"tx_bytes"`
	RxCalls  int64  `json:"rx_calls"`
	TxCalls  int64  `json:"tx_calls"`
}

type Sample struct {
	AgentVersion string       `json:"agent_version"`
	Host         string       `json:"host"`
	MachineID    string       `json:"machine_id"`
	Kernel       string       `json:"kernel"`
	OS           string       `json:"os"`
	SessionUser  string       `json:"session_user"`
	SessionUID   int          `json:"session_uid"`
	WindowStart  float64      `json:"window_start"`
	WindowEnd    float64      `json:"window_end"`
	IntervalSec  float64      `json:"interval_sec"`
	KeysPressed  int64        `json:"keys_pressed"`
	KeysReleased int64        `json:"keys_released"`
	Clicks       int64        `json:"clicks"`
	MouseMoves   int64        `json:"mouse_moves"`
	Scrolls      int64        `json:"scrolls"`
	RxBytes      int64        `json:"rx_bytes"`
	TxBytes      int64        `json:"tx_bytes"`
	NetCalls     int64        `json:"net_calls"`
	PerUserNet   []PerUserNet `json:"per_user_net"`
	Load1        float64      `json:"load1"`
	Load5        float64      `json:"load5"`
	Load15       float64      `json:"load15"`
	MemUsedPct   float64      `json:"mem_used_pct"`
	CPUUsedPct   float64      `json:"cpu_used_pct"`
	NumProcs     int          `json:"num_procs"`
}

type Session struct {
	ID          int64  `json:"id"`
	MachineID   string `json:"machine_id"`
	Host        string `json:"host"`
	SessionUser string `json:"session_user"`
	SessionUID  int    `json:"session_uid"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	StartedAt   int64  `json:"started_at"`
	LastSeenAt  int64  `json:"last_seen_at"`
	ClosedAt    *int64 `json:"closed_at,omitempty"`
}

// --- writes ---

// Ingest either finds or opens a session for (machine_id, session_user) and
// appends a sample. If no user is logged in (session_user=="") samples are
// attributed to a synthetic "_system" user so the machine stays visible.
func (s *Store) Ingest(sample *Sample) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	user := sample.SessionUser
	if user == "" {
		user = "_system"
	}

	now := time.Now().Unix()
	winStart := int64(sample.WindowStart)
	winEnd := int64(sample.WindowEnd)
	if winStart == 0 {
		winStart = now - int64(sample.IntervalSec)
	}
	if winEnd == 0 {
		winEnd = now
	}
	// Session-gap check anchors on the incoming sample's window, not wallclock,
	// so backfills and historical ingests collapse to a small number of
	// sessions instead of one per sample.
	anchor := winEnd

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck

	// Find most recent open session for this (machine, user) that hasn't
	// gone idle. Otherwise open a new one.
	var sessID int64
	var lastSeen int64
	var closedAt sql.NullInt64
	row := tx.QueryRow(`SELECT id, last_seen_at, closed_at FROM sessions
	                     WHERE machine_id=? AND session_user=?
	                  ORDER BY id DESC LIMIT 1`,
		sample.MachineID, user)
	err = row.Scan(&sessID, &lastSeen, &closedAt)
	needNew := false
	switch err {
	case sql.ErrNoRows:
		needNew = true
	case nil:
		if closedAt.Valid {
			needNew = true
		} else if anchor-lastSeen > int64(IdleGap.Seconds()) {
			// Close stale session and start a new one.
			if _, err := tx.Exec(`UPDATE sessions SET closed_at=? WHERE id=?`,
				lastSeen, sessID); err != nil {
				return 0, err
			}
			needNew = true
		}
	default:
		return 0, err
	}

	if needNew {
		res, err := tx.Exec(`INSERT INTO sessions
		       (machine_id, host, session_user, session_uid, os, kernel,
		        started_at, last_seen_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
			sample.MachineID, sample.Host, user, sample.SessionUID,
			sample.OS, sample.Kernel, winStart, winEnd)
		if err != nil {
			return 0, err
		}
		sessID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
		if _, err := tx.Exec(`UPDATE sessions SET last_seen_at=? WHERE id=?`,
			winEnd, sessID); err != nil {
			return 0, err
		}
	}

	perUserJSON, _ := json.Marshal(sample.PerUserNet)
	if _, err := tx.Exec(`INSERT INTO samples
	       (session_id, window_start, window_end, interval_sec,
	        keys_pressed, keys_released, clicks, mouse_moves, scrolls,
	        rx_bytes, tx_bytes, net_calls,
	        cpu_used_pct, mem_used_pct, load1, load5, load15, num_procs,
	        per_user_net)
	 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sessID, winStart, winEnd, sample.IntervalSec,
		sample.KeysPressed, sample.KeysReleased, sample.Clicks,
		sample.MouseMoves, sample.Scrolls,
		sample.RxBytes, sample.TxBytes, sample.NetCalls,
		sample.CPUUsedPct, sample.MemUsedPct,
		sample.Load1, sample.Load5, sample.Load15, sample.NumProcs,
		string(perUserJSON),
	); err != nil {
		return 0, err
	}

	return sessID, tx.Commit()
}

func (s *Store) CloseIdleSessions(now time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	cutoff := now.Add(-IdleGap).Unix()
	_, err := s.db.Exec(`UPDATE sessions SET closed_at = last_seen_at
	                      WHERE closed_at IS NULL AND last_seen_at < ?`, cutoff)
	return err
}

// --- reads ---

func (s *Store) ListUsers() ([]map[string]any, error) {
	rows, err := s.db.Query(`
	    SELECT session_user,
	           COUNT(DISTINCT id) AS session_count,
	           MIN(started_at)    AS first_seen,
	           MAX(last_seen_at)  AS last_seen,
	           COUNT(DISTINCT machine_id) AS machine_count
	      FROM sessions
	  GROUP BY session_user
	  ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var user string
		var count, first, last int64
		var machines int
		if err := rows.Scan(&user, &count, &first, &last, &machines); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"user":          user,
			"session_count": count,
			"first_seen":    first,
			"last_seen":     last,
			"machine_count": machines,
		})
	}
	return out, rows.Err()
}

func (s *Store) ListSessions(user string, since int64) ([]Session, error) {
	var rows *sql.Rows
	var err error
	if user != "" {
		rows, err = s.db.Query(`
		  SELECT id, machine_id, host, session_user, session_uid,
		         COALESCE(os,''), COALESCE(kernel,''),
		         started_at, last_seen_at, closed_at
		    FROM sessions
		   WHERE session_user = ? AND last_seen_at >= ?
		ORDER BY started_at DESC`, user, since)
	} else {
		rows, err = s.db.Query(`
		  SELECT id, machine_id, host, session_user, session_uid,
		         COALESCE(os,''), COALESCE(kernel,''),
		         started_at, last_seen_at, closed_at
		    FROM sessions
		   WHERE last_seen_at >= ?
		ORDER BY started_at DESC`, since)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var sess Session
		var closed sql.NullInt64
		if err := rows.Scan(&sess.ID, &sess.MachineID, &sess.Host,
			&sess.SessionUser, &sess.SessionUID, &sess.OS, &sess.Kernel,
			&sess.StartedAt, &sess.LastSeenAt, &closed); err != nil {
			return nil, err
		}
		if closed.Valid {
			v := closed.Int64
			sess.ClosedAt = &v
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// GetSession returns a single session by id.
func (s *Store) GetSession(id int64) (*Session, error) {
	row := s.db.QueryRow(`
	  SELECT id, machine_id, host, session_user, session_uid,
	         COALESCE(os,''), COALESCE(kernel,''),
	         started_at, last_seen_at, closed_at
	    FROM sessions WHERE id = ?`, id)
	var sess Session
	var closed sql.NullInt64
	if err := row.Scan(&sess.ID, &sess.MachineID, &sess.Host,
		&sess.SessionUser, &sess.SessionUID, &sess.OS, &sess.Kernel,
		&sess.StartedAt, &sess.LastSeenAt, &closed); err != nil {
		return nil, err
	}
	if closed.Valid {
		v := closed.Int64
		sess.ClosedAt = &v
	}
	return &sess, nil
}

// Samples returns samples for the given user within [from,to], grouped in
// order of window_end ascending. If sessionID is non-zero, scope to that session.
func (s *Store) Samples(user string, sessionID int64, from, to int64) ([]Sample, error) {
	q := `SELECT s.window_start, s.window_end, s.interval_sec,
	             s.keys_pressed, s.keys_released, s.clicks, s.mouse_moves, s.scrolls,
	             s.rx_bytes, s.tx_bytes, s.net_calls,
	             s.cpu_used_pct, s.mem_used_pct,
	             s.load1, s.load5, s.load15, s.num_procs,
	             s.per_user_net,
	             se.machine_id, se.host, se.session_user, se.session_uid,
	             COALESCE(se.os,''), COALESCE(se.kernel,'')
	        FROM samples s
	        JOIN sessions se ON se.id = s.session_id
	       WHERE s.window_end >= ? AND s.window_end <= ?`
	args := []any{from, to}
	if user != "" {
		q += " AND se.session_user = ?"
		args = append(args, user)
	}
	if sessionID > 0 {
		q += " AND se.id = ?"
		args = append(args, sessionID)
	}
	q += " ORDER BY s.window_end ASC"
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Sample
	for rows.Next() {
		var sa Sample
		var perUserJSON string
		var winStart, winEnd int64
		if err := rows.Scan(&winStart, &winEnd, &sa.IntervalSec,
			&sa.KeysPressed, &sa.KeysReleased, &sa.Clicks, &sa.MouseMoves, &sa.Scrolls,
			&sa.RxBytes, &sa.TxBytes, &sa.NetCalls,
			&sa.CPUUsedPct, &sa.MemUsedPct,
			&sa.Load1, &sa.Load5, &sa.Load15, &sa.NumProcs,
			&perUserJSON,
			&sa.MachineID, &sa.Host, &sa.SessionUser, &sa.SessionUID,
			&sa.OS, &sa.Kernel); err != nil {
			return nil, err
		}
		sa.WindowStart = float64(winStart)
		sa.WindowEnd = float64(winEnd)
		if perUserJSON != "" {
			_ = json.Unmarshal([]byte(perUserJSON), &sa.PerUserNet)
		}
		out = append(out, sa)
	}
	return out, rows.Err()
}
