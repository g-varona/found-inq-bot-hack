package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/sirupsen/logrus"
)

// ConfluenceService handles Confluence API interactions
type ConfluenceService struct {
	client  *http.Client
	config  *config.Config
	baseURL string
}

// ConfluencePage represents a Confluence page
type ConfluencePage struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
	Author  string `json:"author"`
}

// ConfluenceSearchResult represents search results from Confluence
type ConfluenceSearchResult struct {
	Results []ConfluencePage `json:"results"`
	Size    int              `json:"size"`
}

// NewConfluenceService creates a new Confluence service instance
func NewConfluenceService(cfg *config.Config) *ConfluenceService {
	return &ConfluenceService{
		client: &http.Client{
			Timeout: 15 * time.Second, // 15 second timeout for Confluence API calls
		},
		config:  cfg,
		baseURL: cfg.ConfluenceBaseURL,
	}
}

// SearchPages searches for pages in Confluence
func (s *ConfluenceService) SearchPages(query string) ([]ConfluencePage, error) {
	if s.config.ConfluenceBaseURL == "" || s.config.ConfluenceAPIToken == "" {
		logrus.Warn("missing Confluence configuration, skipping search")
		return []ConfluencePage{}, nil
	}

	// Build the search URL
	searchURL := fmt.Sprintf("%s/rest/api/content/search", s.baseURL)

	// Build query parameters
	params := url.Values{}
	// Sanitize and escape the query to prevent CQL injection
	sanitizedQuery := s.sanitizeCQLQuery(query)
	params.Add("cql", fmt.Sprintf("space=%s AND text ~ \"%s\"", s.config.ConfluenceSpaceKey, sanitizedQuery))
	params.Add("limit", fmt.Sprintf("%d", s.config.MaxSearchResults))
	params.Add("expand", "body.storage,version,space")

	// Create request
	req, err := http.NewRequest("GET", searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	req.SetBasicAuth(s.config.ConfluenceUsername, s.config.ConfluenceAPIToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Confluence API error")
		return nil, fmt.Errorf("confluence API error: %d", resp.StatusCode)
	}

	// Parse response
	var searchResult ConfluenceSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Process results
	pages := make([]ConfluencePage, 0, len(searchResult.Results))
	for _, result := range searchResult.Results {
		page := ConfluencePage{
			ID:    result.ID,
			Title: result.Title,
			URL:   fmt.Sprintf("%s/pages/viewpage.action?pageId=%s", s.baseURL, result.ID),
		}

		// Extract content from the body if available
		if result.Content != "" {
			page.Content = s.extractContentText(result.Content)
		}

		pages = append(pages, page)
	}

	return pages, nil
}

// GetPage retrieves a specific page from Confluence
func (s *ConfluenceService) GetPage(pageID string) (*ConfluencePage, error) {
	if s.config.ConfluenceBaseURL == "" || s.config.ConfluenceAPIToken == "" {
		return nil, fmt.Errorf("missing Confluence configuration")
	}

	// Build the page URL
	pageURL := fmt.Sprintf("%s/rest/api/content/%s", s.baseURL, pageID)

	// Build query parameters
	params := url.Values{}
	params.Add("expand", "body.storage,version,space")

	// Create request
	req, err := http.NewRequest("GET", pageURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	req.SetBasicAuth(s.config.ConfluenceUsername, s.config.ConfluenceAPIToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("confluence API error: %d", resp.StatusCode)
	}

	// Parse response
	var page ConfluencePage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Set URL
	page.URL = fmt.Sprintf("%s/pages/viewpage.action?pageId=%s", s.baseURL, page.ID)

	// Extract content text
	if page.Content != "" {
		page.Content = s.extractContentText(page.Content)
	}

	return &page, nil
}

// extractContentText extracts plain text from Confluence storage format
func (s *ConfluenceService) extractContentText(content string) string {
	// This is a simplified text extraction
	// In a production environment, you might want to use a proper HTML parser

	// Remove HTML tags
	text := strings.ReplaceAll(content, "<", " <")
	text = strings.ReplaceAll(text, ">", "> ")

	// Remove common HTML elements
	replacements := []string{
		"<p>", "", "</p>", "",
		"<div>", "", "</div>", "",
		"<span>", "", "</span>", "",
		"<strong>", "", "</strong>", "",
		"<em>", "", "</em>", "",
		"<br>", "\n", "<br/>", "\n",
		"&nbsp;", " ",
	}

	for i := 0; i < len(replacements); i += 2 {
		text = strings.ReplaceAll(text, replacements[i], replacements[i+1])
	}

	// Clean up extra whitespace
	words := strings.Fields(text)
	cleanText := strings.Join(words, " ")

	// Limit length
	if len(cleanText) > 500 {
		cleanText = cleanText[:500] + "..."
	}

	return cleanText
}

// ValidateConnection validates the Confluence connection
func (s *ConfluenceService) ValidateConnection() error {
	if s.config.ConfluenceBaseURL == "" || s.config.ConfluenceAPIToken == "" {
		return fmt.Errorf("missing Confluence configuration")
	}

	// Test connection by getting space info
	spaceURL := fmt.Sprintf("%s/rest/api/space/%s", s.baseURL, s.config.ConfluenceSpaceKey)

	req, err := http.NewRequest("GET", spaceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.config.ConfluenceUsername, s.config.ConfluenceAPIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Confluence: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid Confluence credentials or space: %d", resp.StatusCode)
	}

	return nil
}

// sanitizeCQLQuery sanitizes a query string to prevent CQL injection attacks
func (s *ConfluenceService) sanitizeCQLQuery(query string) string {
	// Remove or escape potentially dangerous CQL characters and operators
	// CQL special characters: AND, OR, NOT, (, ), ", ', \, ~, *, ?, [, ], {, }
	
	// Replace potential CQL operators with spaces to avoid injection
	dangerous := []string{
		" AND ", " OR ", " NOT ",
		"(", ")", "[", "]", "{", "}",
		"\"", "'", "\\",
		"~", "*", "?",
	}
	
	sanitized := query
	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, " ")
	}
	
	// Remove multiple spaces and trim
	words := strings.Fields(sanitized)
	sanitized = strings.Join(words, " ")
	
	// Limit length to prevent extremely long queries
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	return sanitized
}
