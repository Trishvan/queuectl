package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store defines the interface for job persistence.
type Store interface {
	Init() error
	Enqueue(job *Job) error
	FindAndLockJob() (*Job, error)
	UpdateJob(job *Job) error
	GetJob(id string) (*Job, error)
	ListJobsByState(state JobState) ([]*Job, error)
	GetStatusSummary() (map[JobState]int, error)
	Close() error
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.Init(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Init() error {
	query := `
    CREATE TABLE IF NOT EXISTS jobs (
        id TEXT PRIMARY KEY,
        command TEXT NOT NULL,
        state TEXT NOT NULL,
        attempts INTEGER NOT NULL,
        max_retries INTEGER NOT NULL,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        next_run_at DATETIME NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_jobs_state_next_run ON jobs(state, next_run_at);
    `
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) Enqueue(job *Job) error {
	query := `INSERT INTO jobs (id, command, state, attempts, max_retries, created_at, updated_at, next_run_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, job.ID, job.Command, job.State, job.Attempts, job.MaxRetries, job.CreatedAt, job.UpdatedAt, job.NextRunAt)
	return err
}

// FindAndLockJob finds a pending job, locks it by changing its state to 'processing', and returns it.
// This is the critical section for concurrency.
func (s *SQLiteStore) FindAndLockJob() (*Job, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Rollback on any error

	// Find a pending job that is ready to run.
	// The "FOR UPDATE" clause is implicit in SQLite's transaction model.
	// We select the oldest, ready-to-run job.
	query := `SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_run_at
              FROM jobs
              WHERE state = ? AND next_run_at <= ?
              ORDER BY created_at ASC
              LIMIT 1`

	row := tx.QueryRow(query, StatePending, time.Now().UTC())

	job := &Job{}
	err = row.Scan(&job.ID, &job.Command, &job.State, &job.Attempts, &job.MaxRetries, &job.CreatedAt, &job.UpdatedAt, &job.NextRunAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No job available
		}
		return nil, err
	}

	// Lock the job by updating its state
	job.State = StateProcessing
	job.UpdatedAt = time.Now().UTC()
	job.Attempts++

	updateQuery := `UPDATE jobs SET state = ?, updated_at = ?, attempts = ? WHERE id = ?`
	_, err = tx.Exec(updateQuery, job.State, job.UpdatedAt, job.Attempts, job.ID)
	if err != nil {
		return nil, err
	}

	return job, tx.Commit()
}

func (s *SQLiteStore) UpdateJob(job *Job) error {
	job.UpdatedAt = time.Now().UTC()
	query := `UPDATE jobs SET state = ?, attempts = ?, updated_at = ?, next_run_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, job.State, job.Attempts, job.UpdatedAt, job.NextRunAt, job.ID)
	return err
}

func (s *SQLiteStore) GetJob(id string) (*Job, error) {
	query := `SELECT id, command, state, attempts, max_retries, created_at, updated_at, next_run_at FROM jobs WHERE id = ?`
	row := s.db.QueryRow(query, id)

	job := &Job{}
	err := row.Scan(&job.ID, &job.Command, &job.State, &job.Attempts, &job.MaxRetries, &job.CreatedAt, &job.UpdatedAt, &job.NextRunAt)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *SQLiteStore) ListJobsByState(state JobState) ([]*Job, error) {
	query := `SELECT id, command, state, attempts, max_retries, created_at, updated_at FROM jobs WHERE state = ? ORDER BY created_at ASC`
	rows, err := s.db.Query(query, state)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		// Note: next_run_at is not scanned here as it's less relevant for listing
		err := rows.Scan(&job.ID, &job.Command, &job.State, &job.Attempts, &job.MaxRetries, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *SQLiteStore) GetStatusSummary() (map[JobState]int, error) {
	query := `SELECT state, COUNT(*) FROM jobs GROUP BY state`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make(map[JobState]int)
	for rows.Next() {
		var state JobState
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return nil, err
		}
		summary[state] = count
	}
	return summary, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
