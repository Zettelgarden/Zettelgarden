package models

import "time"

type File struct {
	ID            int         `json:"id"`
	UserID        int         `json:"user_id"`
	Name          string      `json:"name"`
	Filetype      string      `json:"filetype"`
	Path          string      `json:"path"`
	Filename      string      `json:"filename"`
	Size          int         `json:"size"`
	CreatedBy     int         `json:"created_by"`
	UpdatedBy     int         `json:"updated_by"`
	CardPK        *int        `json:"card_pk,omitempty"`
	IsDeleted     bool        `json:"is_deleted"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	ThumbnailPath *string     `json:"thumbnail_path"`
	Card          PartialCard `json:"card"`
}

type EditFileMetadataParams struct {
	Name   string `json:"name"`
	CardPK *int   `json:"card_pk,omitempty"`
}

type UploadFileResponse struct {
	Message string `json:"message"`
	File    File   `json:"file"`
}

type UploadFileParams struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}
