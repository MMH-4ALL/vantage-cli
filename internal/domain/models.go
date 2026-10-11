// Package domain defines data shared by services and presentation layers.
package domain

import "time"

type Project struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"created_at"`
	AccessedAt time.Time `json:"accessed_at"`
}

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ProjectID   string    `json:"project_id,omitempty"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
}

// Workflow commands are argument arrays. They are never implicitly passed to a shell.
// Env holds nonsecret overrides; inherited secrets are resolved from the live process.
type Workflow struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	ProjectID      string            `json:"project_id,omitempty"`
	Executable     string            `json:"executable"`
	Args           []string          `json:"args"`
	Directory      string            `json:"directory"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	Env            map[string]string `json:"env,omitempty"`
	InheritEnv     []string          `json:"inherit_env,omitempty"`
	Approved       bool              `json:"approved"`
	CreatedAt      time.Time         `json:"created_at"`
}

type State struct {
	ActiveProjectID string `json:"active_project_id"`
}
