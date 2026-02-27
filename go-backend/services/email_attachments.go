package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	"go-backend/models"
)

// EmailAttachmentService handles email attachment storage operations
type EmailAttachmentService struct {
	db     models.Database
	s3     S3StorageService
	userID int
}

// NewEmailAttachmentService creates a new EmailAttachmentService
func NewEmailAttachmentService(db models.Database, s3 S3StorageService, userID int) *EmailAttachmentService {
	return &EmailAttachmentService{
		db:     db,
		s3:     s3,
		userID: userID,
	}
}

// S3StorageService interface for S3 operations (to be implemented by handlers)
type S3StorageService interface {
	UploadAttachment(key string, data []byte, contentType string) (string, error)
	GenerateThumbnail(data []byte, contentType string) ([]byte, error)
}

// CreateAttachment creates an attachment record and uploads to S3
func (s *EmailAttachmentService) CreateAttachment(ctx context.Context, emailID int, attachment models.EmailAttachment) (*models.EmailAttachment, error) {
	// Note: This creates a database record only. Use CreateAttachmentWithData for full S3 upload.
	// The models.EmailAttachment doesn't have Data field - it's only in services.EmailAttachment
	// The data will need to be passed separately or we need to modify the approach

	// Create attachment record in database
	query := `
		INSERT INTO email_attachments (
			user_id, email_id, filename, content_type, size, s3_key, thumbnail_path, content_id, is_inline
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, email_id, file_id, filename, content_type, size, s3_key, thumbnail_path, content_id, is_inline, created_at, updated_at
	`

	var result models.EmailAttachment
	var fileID sql.NullInt32
	var s3KeyNull sql.NullString
	var thumbnailPathNull sql.NullString
	var contentID sql.NullString
	var contentTypeNull sql.NullString

	// Handle nullable fields
	var contentTypeVal interface{}
	if attachment.ContentType != nil {
		contentTypeVal = *attachment.ContentType
	}
	var contentIDVal interface{}
	if attachment.ContentID != nil {
		contentIDVal = *attachment.ContentID
	}

	err := s.db.QueryRowContext(ctx, query,
		s.userID, emailID, attachment.Filename,
		contentTypeVal, attachment.Size, nil,
		nil, contentIDVal, attachment.IsInline,
	).Scan(
		&result.ID,
		&result.UserID,
		&result.EmailID,
		&fileID,
		&result.Filename,
		&contentTypeNull,
		&result.Size,
		&s3KeyNull,
		&thumbnailPathNull,
		&contentID,
		&result.IsInline,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	// Convert nullable fields
	if fileID.Valid {
		id := int(fileID.Int32)
		result.FileID = &id
	}
	if s3KeyNull.Valid {
		result.S3Key = &s3KeyNull.String
	}
	if thumbnailPathNull.Valid {
		result.ThumbnailPath = &thumbnailPathNull.String
	}
	if contentID.Valid {
		result.ContentID = &contentID.String
	}
	if contentTypeNull.Valid {
		result.ContentType = &contentTypeNull.String
	}

	log.Printf("[email-attachment] created attachment %d for email %d: %s", result.ID, emailID, result.Filename)

	return &result, nil
}

// CreateAttachmentWithData creates an attachment record with data and uploads to S3
func (s *EmailAttachmentService) CreateAttachmentWithData(ctx context.Context, emailID int, filename string, contentType string, contentID string, isInline bool, data []byte) (*models.EmailAttachment, error) {
	// Generate S3 key
	s3Key := fmt.Sprintf("email-attachments/%d/%s", s.userID, uuid.New().String())

	// Upload attachment to S3
	s3KeyPtr := &s3Key
	_, err := s.s3.UploadAttachment(s3Key, data, contentType)
	if err != nil {
		log.Printf("[email-attachment] failed to upload attachment to S3: %v", err)
		return nil, fmt.Errorf("failed to upload attachment: %w", err)
	}

	// Generate thumbnail for images
	var thumbnailPath *string
	if isImageContentTypeString(contentType) {
		thumbnailData, err := s.s3.GenerateThumbnail(data, contentType)
		if err != nil {
			log.Printf("[email-attachment] failed to generate thumbnail: %v", err)
		} else {
			thumbnailS3Key := s3Key + "_thumb.jpg"
			if _, err := s.s3.UploadAttachment(thumbnailS3Key, thumbnailData, "image/jpeg"); err == nil {
				thumbnailPath = &thumbnailS3Key
			}
		}
	}

	// Create attachment record
	attachment := models.EmailAttachment{
		Filename:      filename,
		ContentType:   &contentType,
		ContentID:     &contentID,
		IsInline:      isInline,
		S3Key:         s3KeyPtr,
		ThumbnailPath: thumbnailPath,
	}
	size := int64(len(data))
	attachment.Size = &size

	return s.CreateAttachment(ctx, emailID, attachment)
}

// GetAttachmentsByEmailID retrieves all attachments for an email
func (s *EmailAttachmentService) GetAttachmentsByEmailID(ctx context.Context, userID, emailID int) ([]models.EmailAttachment, error) {
	query := `
		SELECT id, user_id, email_id, file_id, filename, content_type, size, s3_key, thumbnail_path, content_id, is_inline, created_at, updated_at
		FROM email_attachments
		WHERE user_id = $1 AND email_id = $2
		ORDER BY is_inline ASC, filename ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []models.EmailAttachment
	for rows.Next() {
		var a models.EmailAttachment
		var fileID sql.NullInt32
		var s3Key sql.NullString
		var thumbnailPath sql.NullString
		var contentID sql.NullString
		var contentType sql.NullString

		err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.EmailID,
			&fileID,
			&a.Filename,
			&contentType,
			&a.Size,
			&s3Key,
			&thumbnailPath,
			&contentID,
			&a.IsInline,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}

		// Convert nullable fields
		if fileID.Valid {
			id := int(fileID.Int32)
			a.FileID = &id
		}
		if s3Key.Valid {
			a.S3Key = &s3Key.String
		}
		if thumbnailPath.Valid {
			a.ThumbnailPath = &thumbnailPath.String
		}
		if contentID.Valid {
			a.ContentID = &contentID.String
		}
		if contentType.Valid {
			a.ContentType = &contentType.String
		}

		attachments = append(attachments, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attachments: %w", err)
	}

	return attachments, nil
}

// GetAttachmentByID retrieves a single attachment by ID
func (s *EmailAttachmentService) GetAttachmentByID(ctx context.Context, userID, attachmentID int) (*models.EmailAttachment, error) {
	query := `
		SELECT id, user_id, email_id, file_id, filename, content_type, size, s3_key, thumbnail_path, content_id, is_inline, created_at, updated_at
		FROM email_attachments
		WHERE id = $1 AND user_id = $2
	`

	var a models.EmailAttachment
	var fileID sql.NullInt32
	var s3Key sql.NullString
	var thumbnailPath sql.NullString
	var contentID sql.NullString
	var contentType sql.NullString

	err := s.db.QueryRowContext(ctx, query, attachmentID, userID).Scan(
		&a.ID,
		&a.UserID,
		&a.EmailID,
		&fileID,
		&a.Filename,
		&contentType,
		&a.Size,
		&s3Key,
		&thumbnailPath,
		&contentID,
		&a.IsInline,
		&a.CreatedAt,
		&a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("attachment not found")
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	// Convert nullable fields
	if fileID.Valid {
		id := int(fileID.Int32)
		a.FileID = &id
	}
	if s3Key.Valid {
		a.S3Key = &s3Key.String
	}
	if thumbnailPath.Valid {
		a.ThumbnailPath = &thumbnailPath.String
	}
	if contentID.Valid {
		a.ContentID = &contentID.String
	}
	if contentType.Valid {
		a.ContentType = &contentType.String
	}

	return &a, nil
}

// SaveToFileVault links an attachment to a file in the file vault
func (s *EmailAttachmentService) SaveToFileVault(ctx context.Context, userID, attachmentID int, cardPK *int) (*models.EmailAttachment, error) {
	// First get the attachment
	attachment, err := s.GetAttachmentByID(ctx, userID, attachmentID)
	if err != nil {
		return nil, err
	}

	if attachment.S3Key == nil {
		return nil, fmt.Errorf("attachment has no S3 key")
	}

	// Create a file record linked to this attachment
	var contentType string
	if attachment.ContentType != nil {
		contentType = *attachment.ContentType
	} else {
		contentType = "application/octet-stream"
	}

	var fileID sql.NullInt32
	query := `
		INSERT INTO files (name, user_id, type, path, filename, size, card_pk, created_by, updated_by, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id
	`

	var cardPKValue sql.NullInt32
	if cardPK != nil {
		cardPKValue.Int32 = int32(*cardPK)
		cardPKValue.Valid = true
	}

	var size int64
	if attachment.Size != nil {
		size = *attachment.Size
	}

	err = s.db.QueryRowContext(ctx, query,
		attachment.Filename,
		userID,
		contentType,
		attachment.S3Key,
		attachment.S3Key,
		size,
		cardPKValue,
		userID,
		userID,
	).Scan(&fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// Update attachment with file_id
	updateQuery := `
		UPDATE email_attachments
		SET file_id = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, email_id, file_id, filename, content_type, size, s3_key, thumbnail_path, content_id, is_inline, created_at, updated_at
	`

	var result models.EmailAttachment
	var thumbnailPath sql.NullString
	var contentID sql.NullString
	var contentTypeNull sql.NullString
	var s3KeyNull sql.NullString

	err = s.db.QueryRowContext(ctx, updateQuery, fileID, attachmentID, userID).Scan(
		&result.ID,
		&result.UserID,
		&result.EmailID,
		&fileID,
		&result.Filename,
		&contentTypeNull,
		&result.Size,
		&s3KeyNull,
		&thumbnailPath,
		&contentID,
		&result.IsInline,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update attachment: %w", err)
	}

	// Convert nullable fields
	if fileID.Valid {
		id := int(fileID.Int32)
		result.FileID = &id
	}
	if thumbnailPath.Valid {
		result.ThumbnailPath = &thumbnailPath.String
	}
	if contentID.Valid {
		result.ContentID = &contentID.String
	}
	if contentTypeNull.Valid {
		result.ContentType = &contentTypeNull.String
	}
	if s3KeyNull.Valid {
		result.S3Key = &s3KeyNull.String
	}

	log.Printf("[email-attachment] saved attachment %d to file vault (file_id: %d)", attachmentID, fileID.Int32)

	return &result, nil
}

// DeleteAttachment deletes an attachment
func (s *EmailAttachmentService) DeleteAttachment(ctx context.Context, userID, attachmentID int) error {
	query := `DELETE FROM email_attachments WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, attachmentID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("attachment not found")
	}

	log.Printf("[email-attachment] deleted attachment %d", attachmentID)
	return nil
}

// isImageContentTypeString checks if the content type is an image (string version)
func isImageContentTypeString(contentType string) bool {
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/bmp", "image/svg+xml"}
	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

// isImageContentType checks if the content type is an image (pointer version)
func isImageContentType(contentType *string) bool {
	if contentType == nil {
		return false
	}
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp", "image/bmp", "image/svg+xml"}
	for _, t := range imageTypes {
		if *contentType == t {
			return true
		}
	}
	return false
}
