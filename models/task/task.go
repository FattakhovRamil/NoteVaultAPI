package task

import (
	"errors"
)

// @name Task
type Task struct {
	ID          int    `json:"id"`
	Text       string `json:"text"`
	UserID      int    `json:"userId"`
}

// Validate checks if the task data is valid.
func (t *Task) Validate() error {
	// Check if the title is empty.
	if t.Text == "" {
		return errors.New("title is required")
	}
	return nil
}
