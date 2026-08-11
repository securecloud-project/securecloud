package store

import "time"

type ScanStatus string

const (
	StatusQueued   ScanStatus = "queued"
	StatusRunning  ScanStatus = "running"
	StatusComplete ScanStatus = "complete"
	StatusFailed   ScanStatus = "failed"
)

type Finding struct {
	Check    string `json:"check"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type Scan struct {
	ID        string     `json:"id"`
	Target    string     `json:"target"`
	Status    ScanStatus `json:"status"`
	Score     int        `json:"score"`
	Findings  []Finding  `json:"findings"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
