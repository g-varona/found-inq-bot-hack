package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/storage"
	"github.com/sirupsen/logrus"
)

// LLMService handles AI-powered response generation
type LLMService struct {
	client *http.Client
	config *config.Config
}

// LiteLLMRequest represents a request to LiteLLM API
type LiteLLMRequest struct {
	Model       string           `json:"model"`
	Messages    []LiteLLMMessage `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
}

// LiteLLMMessage represents a message in the conversation
type LiteLLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LiteLLMResponse represents a response from LiteLLM API
type LiteLLMResponse struct {
	Choices []LiteLLMChoice `json:"choices"`
}

// LiteLLMChoice represents a choice in the response
type LiteLLMChoice struct {
	Message LiteLLMMessage `json:"message"`
}

// NewLLMService creates a new LLM service instance
func NewLLMService(cfg *config.Config) *LLMService {
	return &LLMService{
		client: &http.Client{
			Timeout: 30 * time.Second, // 30 second timeout for LLM API calls
		},
		config: cfg,
	}
}

// GenerateResponse generates an AI response based on the inquiry and search results
func (s *LLMService) GenerateResponse(ctx context.Context, inquiry *storage.Inquiry, searchResults []storage.SearchResult) (string, error) {
	if s.config.LiteLLMAPIKey == "" || s.config.LiteLLMBaseURL == "" {
		return "", fmt.Errorf("LiteLLM not configured")
	}

	// Build the context from search results
	contextStr := s.buildContext(inquiry, searchResults)

	// Create the prompt
	prompt := s.buildPrompt(inquiry.MessageText, contextStr)

	// Prepare the request payload
	request := LiteLLMRequest{
		Model:       s.config.LLMModel,
		Temperature: s.config.LLMTemperature,
		MaxTokens:   s.config.LLMMaxTokens,
		Messages: []LiteLLMMessage{
			{
				Role:    "system",
				Content: s.getSystemPrompt(),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", s.config.LiteLLMBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-litellm-api-key", s.config.LiteLLMAPIKey)

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Failed to call LiteLLM API")
		return "", fmt.Errorf("failed to call LiteLLM API: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		// Read error response body for more context
		var body map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&body)
		if err != nil {
			logrus.WithError(err).Error("Failed to call LiteLLM API")
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return "", fmt.Errorf("LiteLLM API authentication failed (401): check API key")
		case http.StatusForbidden:
			return "", fmt.Errorf("LiteLLM API access forbidden (403): insufficient permissions")
		case http.StatusTooManyRequests:
			return "", fmt.Errorf("LiteLLM API rate limit exceeded (429): try again later")
		case http.StatusInternalServerError:
			return "", fmt.Errorf("LiteLLM API internal error (500): service unavailable")
		case http.StatusBadRequest:
			return "", fmt.Errorf("LiteLLM API bad request (400): invalid request format")
		default:
			// Log only status code to avoid exposing sensitive information in response body
			logrus.WithFields(logrus.Fields{
				"status_code": resp.StatusCode,
			}).Error("LiteLLM API returned non-200 status")
			return "", fmt.Errorf("LiteLLM API returned status %d", resp.StatusCode)
		}
	}

	// Parse response
	var response LiteLLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	return response.Choices[0].Message.Content, nil
}

// buildContext creates a context string from search results
func (s *LLMService) buildContext(inquiry *storage.Inquiry, searchResults []storage.SearchResult) string {
	var contextParts []string

	// Add inquiry details
	contextParts = append(contextParts, fmt.Sprintf("Original inquiry: %s", inquiry.MessageText))
	contextParts = append(contextParts, "")

	if len(searchResults) == 0 {
		contextParts = append(contextParts, "No relevant historical information found.")
		return strings.Join(contextParts, "\n")
	}

	// Group results by source
	slackResults := []storage.SearchResult{}
	confluenceResults := []storage.SearchResult{}

	for _, result := range searchResults {
		switch result.Source {
		case "slack":
			slackResults = append(slackResults, result)
		case "confluence":
			confluenceResults = append(confluenceResults, result)
		}
	}

	// Add Slack context
	if len(slackResults) > 0 {
		contextParts = append(contextParts, "Similar past Slack discussions:")
		for i, result := range slackResults {
			contextParts = append(contextParts, fmt.Sprintf("%d. %s", i+1, result.Content))
			if result.Author != "" {
				contextParts = append(contextParts, fmt.Sprintf("   (by %s)", result.Author))
			}
			contextParts = append(contextParts, "")
		}
	}

	// Add Confluence context
	if len(confluenceResults) > 0 {
		contextParts = append(contextParts, "Relevant documentation:")
		for i, result := range confluenceResults {
			contextParts = append(contextParts, fmt.Sprintf("%d. %s", i+1, result.Title))
			if result.Content != "" {
				contextParts = append(contextParts, fmt.Sprintf("   %s", result.Content))
			}
			if result.URL != "" {
				contextParts = append(contextParts, fmt.Sprintf("   Link: %s", result.URL))
			}
			contextParts = append(contextParts, "")
		}
	}

	return strings.Join(contextParts, "\n")
}

// buildPrompt creates the final prompt for the LLM
func (s *LLMService) buildPrompt(inquiry, context string) string {
	return fmt.Sprintf(`Based on the following context and inquiry, please provide a helpful and accurate response.

Inquiry: %s

Context:
%s

Please provide a comprehensive answer that:
1. Directly addresses the inquiry
2. References relevant information from the context
3. Is clear and actionable
4. Includes links to documentation when available
5. Suggests next steps if appropriate

Keep the response concise but thorough.`, inquiry, context)
}

// getSystemPrompt returns the system prompt for the LLM
func (s *LLMService) getSystemPrompt() string {
	return `You are a helpful assistant for a company's internal inquiry system. You help answer questions from team members by referencing past discussions and documentation.

Your role is to:
- Provide accurate, helpful responses based on available context
- Reference specific past discussions or documentation when relevant
- Be concise but comprehensive
- Suggest follow-up actions when appropriate
- Maintain a professional but friendly tone

If you don't have enough information to provide a complete answer, acknowledge this and suggest where the person might find more information or who they should contact.`
}
