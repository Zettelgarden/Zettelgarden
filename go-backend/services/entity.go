package services

import (
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func FindEntities(c *models.LLMClient, title, body string) ([]models.Entity, error) {
	systemPrompt := `You are an AI specialized in analyzing zettelkasten cards and extracting entities.
Follow these rules strictly:

1. Entity Types must be one of: person, concept, theory, book, software, place, organization, event, method

2. Entity Names should be:
   - Concise (1-5 words)
   - Properly capitalized
   - Specific enough to be unique
   - Consistent with academic/professional terminology

3. Descriptions should be:
   - 10-20 words maximum
   - Focus on relevance to the card's context
   - Include key relationships or significance
   - Objective and factual

4. Extract entities that are:
   - Explicitly mentioned in the text
   - Significant to the card's main ideas
   - Could be useful for connecting to other cards
   - Worth tracking as separate concepts

Return only valid JSON matching the specified structure.`
	prompt := `Please analyze this zettelkasten card and extract all meaningful entities:
    Title: %s
    Body: %s
    
    Return only a JSON array of entities matching this structure:
[
    
        {
            "name": "entity name",
            "description": "brief description",
            "type": "entity type"
        }
    ]`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf(prompt, title, body),
		},
	}
	var entities []models.Entity
	var jsonErr error
	for range 3 {

		resp, err := ExecuteLLMRequest(c, messages)
		if err != nil {
			log.Printf("error getting completion: %v", err)
			return []models.Entity{}, err
		}
		if len(resp.Choices) == 0 {
			continue
		}
		content := resp.Choices[0].Message.Content
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)

		jsonErr = json.Unmarshal([]byte(content), &entities)
		if jsonErr == nil {
			break
		} else {
			time.Sleep(1000 * time.Millisecond)

		}
	}

	return entities, nil

}

func CheckExistingEntities(c *models.LLMClient, similar []models.Entity, entity models.Entity) (models.Entity, error) {
	if len(similar) == 0 {
		return entity, nil
	}

	// System prompt to explain the task
	systemPrompt := `You are an AI specialized in determining if entities refer to the same thing.
Consider names, descriptions, and entity types carefully.
Return a JSON response indicating if they are the same and explaining why.
Be strict - only indicate they are the same if you are highly confident they refer to the exact same entity.`

	// Check each similar entity
	for _, sim := range similar {
		prompt := fmt.Sprintf(`Compare these two entities and determine if they refer to the same thing:

Entity 1:
Name: %s
Type: %s
Description: %s

Entity 2:
Name: %s
Type: %s
Description: %s

Return JSON in this format:
{
    "areSame": boolean,
    "explanation": "brief explanation of decision",
    "preferredEntity": "1" or "2"
}`,
			entity.Name, entity.Type, entity.Description,
			sim.Name, sim.Type, sim.Description)

		messages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		}

		type Response struct {
			AreSame         bool   `json:"areSame"`
			Explanation     string `json:"explanation"`
			PreferredEntity string `json:"preferredEntity"`
		}

		// Make the API call

		resp, err := ExecuteLLMRequest(c, messages)
		if err != nil {
			log.Printf("error getting completion: %v", err)
			continue
		}

		var result Response

		content := resp.Choices[0].Message.Content
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
		err = json.Unmarshal([]byte(content), &result)
		if err != nil {
			log.Printf("error parsing response: %v", err)
			continue
		}

		// If the LLM thinks they're the same
		if result.AreSame {
			// log.Printf("Found matching entity: %s and %s. Explanation: %s",
			// 	entity.Name, sim.Name, result.Explanation)
			return sim, nil
		}
	}

	// If no matches found or new entity is always preferred
	return entity, nil
}

// FindEntitiesBatch processes multiple cards (title, body) together and returns entities per card.
func FindEntitiesBatch(c *models.LLMClient, facts []models.Fact) ([][]models.Entity, error) {
	systemPrompt := `You are an AI specialized in analyzing zettelkasten cards and extracting entities.
Follow these rules strictly:

1. Entity Types must be one of: person, concept, theory, book, software, place, organization, event, method

2. Entity Names should be:
   - Concise (1-5 words)
   - Properly capitalized
   - Specific enough to be unique
   - Consistent with academic/professional terminology

3. Descriptions should be:
   - 10-20 words maximum
   - Focus on relevance to the card's context
   - Include key relationships or significance
   - Objective and factual

4. Extract entities that are:
   - Explicitly mentioned in the text
   - Significant to the card's main ideas
   - Could be useful for connecting to other cards
   - Worth tracking as separate concepts

Return only valid JSON matching the specified structure.`

	// Build prompt for multiple cards
	var sb strings.Builder
	sb.WriteString("Please analyze the following zettelkasten cards and extract meaningful entities for each.\n")
	sb.WriteString("Return only a JSON array where each element corresponds to a card in this format:\n")
	sb.WriteString(`{
  "cardIndex": i,
  "entities": [
    {
      "name": "entity name",
      "description": "brief description",
      "type": "entity type"
    }
  ]
}
`)
	for i, fact := range facts {
		sb.WriteString(fmt.Sprintf("\nCard %d:\nBody: %s\n", i, fact.Fact))
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: sb.String(),
		},
	}

	type batchEntityResponse struct {
		CardIndex int             `json:"cardIndex"`
		Entities  []models.Entity `json:"entities"`
	}

	var parsed []batchEntityResponse
	var jsonErr error
	for range 3 {
		resp, err := ExecuteLLMRequest(c, messages)
		if err != nil {
			log.Printf("error getting completion: %v", err)
			return nil, err
		}
		if len(resp.Choices) == 0 {
			continue
		}
		content := resp.Choices[0].Message.Content
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)

		jsonErr = json.Unmarshal([]byte(content), &parsed)
		if jsonErr == nil {
			break
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
	}
	if jsonErr != nil {
		return nil, jsonErr
	}
	log.Printf("found %v", parsed)

	// Prepare results with embeddings
	results := make([][]models.Entity, len(facts))
	for _, resp := range parsed {
		var entitiesWithEmb []models.Entity
		for _, entity := range resp.Entities {
			entitiesWithEmb = append(entitiesWithEmb, entity)
		}
		if resp.CardIndex >= 0 && resp.CardIndex < len(facts) {
			results[resp.CardIndex] = entitiesWithEmb
		}
	}

	return results, nil
}

func GenerateNewEntityDescription(c *models.LLMClient, e1, e2 models.Entity, newName string) (string, error) {
	systemPrompt := `You are an AI assistant tasked with synthesizing new entities.
Given two existing entities, propose a new entity with the provided name.
Rules:
- Write a description of 10–20 words.
- Choose the most appropriate type from: person, concept, theory, book, software, place, organization, event, method
- Return a JSON object: { "description": "..." }`

	userPrompt := fmt.Sprintf(`Entity 1:
Name: %s
Type: %s
Description: %s

Entity 2:
Name: %s
Type: %s
Description: %s

New Entity Name: %s`, e1.Name, e1.Type, e1.Description, e2.Name, e2.Type, e2.Description, newName)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		},
	}

	resp, err := ExecuteLLMRequest(c, messages)
	if err != nil {
		return "", fmt.Errorf("error generating new entity: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var newEntity models.Entity
	if err := json.Unmarshal([]byte(content), &newEntity); err != nil {
		return "", fmt.Errorf("error parsing entity JSON: %w", err)
	}

	return newEntity.Description, nil
}
