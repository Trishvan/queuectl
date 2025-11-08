package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JobState string

const (
	StatePending    JobState = "pending"
	StateProcessing JobState = "processing"
	StateCompleted  JobState = "completed"
	StateFailed     JobState = "failed"
	StateDead       JobState = "dead"
)

type Job struct {
	ID         string    `json:"id"`
	Command    string    `json:"command"`
	State      JobState  `json:"state"`
	Attempts   int       `json:"attempts"`
	MaxRetries int       `json:"max_retries"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	NextRunAt  time.Time `json:"-"` // Not exposed in JSON, used for scheduling
}

// NewJobFromSpec creates a job from a JSON string specification.
func NewJobFromSpec(spec string, defaultMaxRetries int) (*Job, error) {
	var partialJob struct {
		ID      string `json:"id"`
		Command string `json:"command"`
	}

	if err := json.Unmarshal([]byte(spec), &partialJob); err != nil {
		return nil, err
	}

	jobID := partialJob.ID
	if jobID == "" {
		jobID = uuid.New().String()
	}

	now := time.Now().UTC()
	return &Job{
		ID:         jobID,
		Command:    partialJob.Command,
		State:      StatePending,
		Attempts:   0,
		MaxRetries: defaultMaxRetries,
		CreatedAt:  now,
		UpdatedAt:  now,
		NextRunAt:  now,
	}, nil
}
