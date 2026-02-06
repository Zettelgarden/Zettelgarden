// Package chat provides SSE streaming functionality for the chat service.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"io"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// SSE event types
const (
	SSEEventTypeContent    = "content"
	SSEEventTypeToolCall   = "tool_call"
	SSEEventTypeToolResult = "tool_result"
	SSEEventTypeProgress   = "progress"
	SSEEventTypeError      = "error"
	SSEEventTypeDone       = "done"
)

// StreamAssistantResponse handles streaming the assistant response via Server-Sent Events.
func (s *Service) StreamAssistantResponse(
	ctx context.Context,
	w http.ResponseWriter,
	userID int,
	conversation *models.ChatConversation,
	assistantMessageID string,
	modelOverride *string,
	getMessages func(conversationID string) ([]models.ChatMessage, error),
	updateMessage func(messageID string, content *string, toolCallsJSON *string, status string) error,
) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.logError("Streaming not supported")
		return fmt.Errorf("streaming unsupported")
	}

	// Ensure message status is cleaned up on any exit path
	updateStatus := true
	defer func() {
		if ctx.Err() != nil {
			s.logError("Client disconnected during streaming, marking message as failed: %s", assistantMessageID)
			updateMessage(assistantMessageID, nil, nil, "failed")
			return
		}
		if updateStatus {
			// Status is already set by updateMessage to "completed"
		}
	}()

	// Helper to send SSE events
	sendEvent := func(eventType string, data interface{}) error {
		return s.sendSSEEvent(w, flusher, eventType, data)
	}

	// Check for client disconnect early
	select {
	case <-ctx.Done():
		s.logError("Client disconnected before streaming started: %s", assistantMessageID)
		return ctx.Err()
	default:
	}

	// Generate title if needed and send event
	if err := s.GenerateTitleIfNeeded(ctx, userID, conversation, getMessages); err != nil {
		s.logError("Error generating title: %v", err)
	}

	// Get conversation history
	messages, err := getMessages(conversation.ID)
	if err != nil {
		s.logError("Error getting conversation history: %v", err)
		updateMessage(assistantMessageID, nil, nil, "failed")
		sendEvent(SSEEventTypeError, map[string]string{"error": "Failed to get conversation history"})
		return err
	}

	// Determine which model to use
	modelToUse := s.determineModel(conversation, modelOverride)

	// Generate response with streaming
	if err := s.StreamChatResponse(ctx, w, userID, conversation, messages, modelToUse, assistantMessageID, sendEvent, updateMessage); err != nil {
		s.logError("Error generating chat response: %v", err)
		updateMessage(assistantMessageID, nil, nil, "failed")
		sendEvent(SSEEventTypeError, map[string]string{"error": err.Error()})
		return err
	}

	sendEvent(SSEEventTypeDone, map[string]string{"message_id": assistantMessageID})
	return nil
}

// sendSSEEvent sends a Server-Sent Event to the client.
func (s *Service) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonData)
	flusher.Flush()
	return nil
}

// sendSSEProgressEvent sends a progress update during long-running operations.
func (s *Service) sendSSEProgressEvent(sendEvent func(string, interface{}) error, step int, message string) error {
	return sendEvent(SSEEventTypeProgress, map[string]interface{}{
		"step":       step,
		"message":    message,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// sendSSEContentEvent sends a content delta event for streaming text.
func (s *Service) sendSSEContentEvent(sendEvent func(string, interface{}) error, delta string) error {
	return sendEvent(SSEEventTypeContent, map[string]string{
		"delta": delta,
	})
}

// sendSSEToolCallEvent sends a tool call invocation event.
func (s *Service) sendSSEToolCallEvent(sendEvent func(string, interface{}) error, toolCall openai.ToolCall) error {
	var args map[string]interface{}
	json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
	return sendEvent(SSEEventTypeToolCall, map[string]interface{}{
		"id":        toolCall.ID,
		"name":      toolCall.Function.Name,
		"arguments": args,
	})
}

// sendSSEErrorEvent sends an error event to the client.
func (s *Service) sendSSEErrorEvent(sendEvent func(string, interface{}) error, errMsg string) error {
	return sendEvent(SSEEventTypeError, map[string]string{
		"error": errMsg,
	})
}

// processStreamResponse processes the streaming LLM response and collects content and tool calls.
func (s *Service) processStreamResponse(ctx context.Context, stream *openai.ChatCompletionStream, sendEvent func(string, interface{}) error) (string, []openai.ToolCall, error) {
	var currentContent string
	var currentToolCalls []openai.ToolCall

	for {
		select {
		case <-ctx.Done():
			s.logError("Client disconnected during stream processing")
			return "", nil, ctx.Err()
		default:
		}

		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			s.logError("Stream error: %v", err)
			return "", nil, err
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta

		if delta.Content != "" {
			currentContent += delta.Content
			sendEvent(SSEEventTypeContent, map[string]string{"delta": delta.Content})
		}

		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.Index != nil {
					idx := *tc.Index
					for len(currentToolCalls) <= idx {
						currentToolCalls = append(currentToolCalls, openai.ToolCall{})
					}
					if tc.ID != "" {
						currentToolCalls[idx].ID = tc.ID
						currentToolCalls[idx].Type = tc.Type
					}
					if tc.Function.Name != "" {
						currentToolCalls[idx].Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						currentToolCalls[idx].Function.Arguments += tc.Function.Arguments
					}
				}
			}
		}
	}

	return currentContent, currentToolCalls, nil
}

// isStreamingSupported checks if the response writer supports SSE streaming.
func (s *Service) isStreamingSupported(w http.ResponseWriter) bool {
	_, ok := w.(http.Flusher)
	return ok
}
