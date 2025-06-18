package models

import "time"

type Feedback struct {
	FeedbackID string    `json:"feedback_id,omitempty"`
	UserID     string    `json:"user_id" validate:"required"`
	BookID     string    `json:"book_id" validate:"required"`
	Text       string    `json:"text" validate:"required"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	Name       string    `json:"name"`
	Lastname   string    `json:"lastname"`
}
type SortOrder string
