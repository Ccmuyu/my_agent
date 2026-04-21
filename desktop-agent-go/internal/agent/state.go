package agent

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusConfirming TaskStatus = "confirming"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusCancelled   TaskStatus = "cancelled"
)

type Task struct {
	ID            string      `json:"task_id"`
	Intent        string      `json:"intent"`
	Status        TaskStatus  `json:"status"`
	Actions       []Action    `json:"actions"`
	CurrentAction int         `json:"current_action"`
	Result        []ActionResult `json:"result"`
	Error         string      `json:"error,omitempty"`
	Confirmed     bool        `json:"confirmed"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type Action struct {
	Tool      string         `json:"tool"`
	Params    map[string]any `json:"params"`
	Retry     int           `json:"retry"`
	RiskScore int           `json:"risk_score"`
}

type ActionResult struct {
	Success    bool        `json:"success"`
	Output     any         `json:"output,omitempty"`
	Error      string      `json:"error,omitempty"`
	ActionIndex int       `json:"action_index"`
}

func NewTask(intent string) *Task {
	id := uuid.New().String()[:8]
	return &Task{
		ID:            id,
		Intent:        intent,
		Status:        TaskStatusPending,
		Actions:       []Action{},
		CurrentAction: 0,
		Result:        []ActionResult{},
		Confirmed:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func (t *Task) ToMap() map[string]any {
	return map[string]any{
		"task_id":         t.ID,
		"intent":          t.Intent,
		"status":          t.Status,
		"current_action":  t.CurrentAction,
		"total_actions":   len(t.Actions),
		"result":          t.Result,
		"error":           t.Error,
		"created_at":      t.CreatedAt.Format(time.RFC3339),
		"updated_at":      t.UpdatedAt.Format(time.RFC3339),
	}
}
