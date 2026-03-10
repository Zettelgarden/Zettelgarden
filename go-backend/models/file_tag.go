package models

import "time"

type FileTag struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type FileTagWithCount struct {
	FileTag
	FileCount int `json:"file_count"`
}
