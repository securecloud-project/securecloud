package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if dbPath != ":memory:" {
		if err := os.Chmod(dbPath, 0o600); err != nil {
			db.Close()
			return nil, fmt.Errorf("secure database permissions: %w", err)
		}
	}

	const schema = `
	CREATE TABLE IF NOT EXISTS scans (
		id TEXT PRIMARY KEY,
		target TEXT NOT NULL,
		status TEXT NOT NULL,
		score INTEGER NOT NULL DEFAULT 0,
		findings TEXT NOT NULL DEFAULT '[]',
		error TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create scans table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) CreateScan(ctx context.Context, target string) (*Scan, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate scan ID: %w", err)
	}

	now := time.Now().UTC()

	scan := &Scan{
		ID:        id,
		Target:    target,
		Status:    StatusQueued,
		Score:     0,
		Findings:  []Finding{},
		Error:     "",
		CreatedAt: now,
		UpdatedAt: now,
	}

	findingsJSON, err := json.Marshal(scan.Findings)
	if err != nil {
		return nil, fmt.Errorf("encode findings: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`
		INSERT INTO scans (
			id,
			target,
			status,
			score,
			findings,
			error,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
		scan.ID,
		scan.Target,
		scan.Status,
		scan.Score,
		string(findingsJSON),
		scan.Error,
		scan.CreatedAt.Format(time.RFC3339Nano),
		scan.UpdatedAt.Format(time.RFC3339Nano),
	)

	if err != nil {
		return nil, fmt.Errorf("insert scan: %w", err)
	}

	return scan, nil
}

func (s *Store) MarkRunning(ctx context.Context, id string) error {
	return s.transition(ctx, id, StatusQueued, StatusRunning, 0, nil, "")
}

func (s *Store) CompleteScan(ctx context.Context, id string, score int, findings []Finding) error {
	if score < 0 || score > 100 {
		return fmt.Errorf("score %d is outside 0..100", score)
	}
	return s.transition(ctx, id, StatusRunning, StatusComplete, score, findings, "")
}

func (s *Store) FailScan(ctx context.Context, id, message string) error {
	if message == "" {
		message = "scan failed"
	}
	findings := []Finding{{Check: "scan", Severity: "error", Message: message}}
	encoded, err := json.Marshal(findings)
	if err != nil {
		return fmt.Errorf("encode findings: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE scans SET status = ?, score = 0, findings = ?, error = ?, updated_at = ?
		WHERE id = ? AND status IN (?, ?)
	`, StatusFailed, string(encoded), message, time.Now().UTC().Format(time.RFC3339Nano), id, StatusQueued, StatusRunning)
	if err != nil {
		return fmt.Errorf("fail scan: %w", err)
	}
	return requireUpdated(result, id)
}

func (s *Store) RequeueRunning(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE scans SET status = ?, error = '', updated_at = ? WHERE status = ?
	`, StatusQueued, time.Now().UTC().Format(time.RFC3339Nano), StatusRunning)
	if err != nil {
		return fmt.Errorf("requeue interrupted scans: %w", err)
	}
	return nil
}

func (s *Store) ListQueuedIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM scans WHERE status = ? ORDER BY created_at`, StatusQueued)
	if err != nil {
		return nil, fmt.Errorf("query queued scans: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("decode queued scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) transition(ctx context.Context, id string, from, to ScanStatus, score int, findings []Finding, message string) error {
	if findings == nil {
		findings = []Finding{}
	}
	encoded, err := json.Marshal(findings)
	if err != nil {
		return fmt.Errorf("encode findings: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE scans SET status = ?, score = ?, findings = ?, error = ?, updated_at = ?
		WHERE id = ? AND status = ?
	`, to, score, string(encoded), message, time.Now().UTC().Format(time.RFC3339Nano), id, from)
	if err != nil {
		return fmt.Errorf("transition scan: %w", err)
	}
	return requireUpdated(result, id)
}

func requireUpdated(result sql.Result, id string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("scan %s is missing or in an invalid state", id)
	}
	return nil
}

func (s *Store) GetScan(ctx context.Context, id string) (*Scan, error) {
	row := s.db.QueryRowContext(
		ctx,
		`
		SELECT
			id,
			target,
			status,
			score,
			findings,
			error,
			created_at,
			updated_at
		FROM scans
		WHERE id = ?
		`,
		id,
	)

	scan, err := decodeScan(row)
	if err != nil {
		return nil, err
	}

	return scan, nil
}

func (s *Store) ListScans(ctx context.Context) ([]Scan, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`
		SELECT
			id,
			target,
			status,
			score,
			findings,
			error,
			created_at,
			updated_at
		FROM scans
		ORDER BY created_at DESC
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("query scans: %w", err)
	}

	defer rows.Close()

	scans := make([]Scan, 0)

	for rows.Next() {
		scan, err := decodeScan(rows)
		if err != nil {
			return nil, err
		}

		scans = append(scans, *scan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scans: %w", err)
	}

	return scans, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func decodeScan(row scanner) (*Scan, error) {
	var scan Scan

	var findingsJSON string
	var createdAt string
	var updatedAt string

	err := row.Scan(
		&scan.ID,
		&scan.Target,
		&scan.Status,
		&scan.Score,
		&findingsJSON,
		&scan.Error,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(findingsJSON), &scan.Findings); err != nil {
		return nil, fmt.Errorf("decode findings: %w", err)
	}

	scan.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	scan.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	return &scan, nil
}

func generateID() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
