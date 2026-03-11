package services

import (
	"testing"

	"go-backend/models"
)

func TestFileTextExtractionJob_ProcessJob(t *testing.T) {
	// This is an integration test that would need real file and database
	t.Skip("Integration test - requires real file and database")
}

func TestFileTextExtractionJob_MissingPayload(t *testing.T) {
	// Test that file_text_extraction job type is recognized
	// (actual processing logic will be tested in integration tests)
	job := &models.LLMJob{
		JobType: models.JobTypeFileTextExtraction,
		Payload: map[string]interface{}{}, // Empty payload
	}

	// Just verify the job type exists
	if job.JobType != "file_text_extraction" {
		t.Errorf("Expected job type 'file_text_extraction', got %s", job.JobType)
	}
}
