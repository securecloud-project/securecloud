package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
