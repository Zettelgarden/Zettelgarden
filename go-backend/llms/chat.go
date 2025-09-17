package llms

import (
	"context"
	"go-backend/models"

	openai "github.com/sashabaranov/go-openai"
)

func CreateChatCompletion(c *models.LLMClient, ctx context.Context, messages []models.ChatMessage, context string) (string, error) {
	var openaiMessages []openai.ChatCompletionMessage

	if context != "" {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. Please use the following context to answer the user's question:\n\n" + context,
		})
	}

	for _, msg := range messages {
		var content string
		if msg.Content != nil {
			content = *msg.Content
		}
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	resp, err := ExecuteLLMRequest(c, openaiMessages)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
