package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/storage"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SearchService handles searching across multiple sources
type SearchService struct {
	slack      *SlackService
	confluence *ConfluenceService
	db         *gorm.DB
	config     *config.Config
}

// NewSearchService creates a new search service instance
func NewSearchService(slack *SlackService, confluence *ConfluenceService, db *gorm.DB, cfg *config.Config) *SearchService {
	return &SearchService{
		slack:      slack,
		confluence: confluence,
		db:         db,
		config:     cfg,
	}
}

// SearchAll searches across all available sources (Slack and Confluence)
func (s *SearchService) SearchAll(ctx context.Context, query string, inquiryID uint) ([]storage.SearchResult, error) {
	var allResults []storage.SearchResult

	// Extract keywords from the query for better searching
	keywords := s.extractKeywords(query)
	searchQuery := strings.Join(keywords, " ")

	logrus.WithFields(logrus.Fields{
		"original_query": query,
		"search_query":   searchQuery,
		"inquiry_id":     inquiryID,
	}).Info("Starting search across all sources")

	// Search Slack messages
	if slackResults, err := s.searchSlack(ctx, searchQuery, inquiryID); err != nil {
		logrus.WithError(err).Error("Failed to search Slack")
	} else {
		allResults = append(allResults, slackResults...)
	}

	// Search Confluence pages
	if confluenceResults, err := s.searchConfluence(ctx, searchQuery, inquiryID); err != nil {
		logrus.WithError(err).Error("Failed to search Confluence")
	} else {
		allResults = append(allResults, confluenceResults...)
	}

	// Filter and rank results
	filteredResults := s.filterAndRankResults(allResults)

	logrus.WithFields(logrus.Fields{
		"total_results":    len(allResults),
		"filtered_results": len(filteredResults),
		"inquiry_id":       inquiryID,
	}).Info("Search completed")

	return filteredResults, nil
}

// searchSlack searches for relevant messages in Slack
func (s *SearchService) searchSlack(ctx context.Context, query string, inquiryID uint) ([]storage.SearchResult, error) {
	_, cancelFn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()
	messages, err := s.slack.SearchMessages(query, s.config.SearchDaysBack)
	if err != nil {
		return nil, err
	}

	var results []storage.SearchResult
	for _, msg := range messages {
		// Get user info for author name
		author := msg.User
		if user, err := s.slack.GetUserInfo(msg.User); err == nil && user.RealName != "" {
			author = user.RealName
		}

		// Create search result
		result := storage.SearchResult{
			InquiryID:   inquiryID,
			Source:      "slack",
			SourceID:    msg.Timestamp,
			Title:       "Slack Message",
			Content:     msg.Text,
			URL:         s.buildSlackMessageURL(msg.Channel, msg.Timestamp),
			Score:       s.calculateRelevanceScore(msg.Text, query),
			Author:      author,
			CreatedDate: s.timestampToTime(msg.Timestamp),
		}

		results = append(results, result)
	}

	// Save results to database
	for _, result := range results {
		if err := s.db.Create(&result).Error; err != nil {
			logrus.WithError(err).Error("Failed to save Slack search result")
		}
	}

	return results, nil
}

// searchConfluence searches for relevant pages in Confluence
func (s *SearchService) searchConfluence(ctx context.Context, query string, inquiryID uint) ([]storage.SearchResult, error) {
	_, cancelFn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()
	pages, err := s.confluence.SearchPages(query)
	if err != nil {
		return nil, err
	}

	var results []storage.SearchResult
	for _, page := range pages {
		result := storage.SearchResult{
			InquiryID:   inquiryID,
			Source:      "confluence",
			SourceID:    page.ID,
			Title:       page.Title,
			Content:     page.Content,
			URL:         page.URL,
			Score:       s.calculateRelevanceScore(page.Title+" "+page.Content, query),
			Author:      page.Author,
			CreatedDate: time.Now(), // Confluence API doesn't always provide creation date
		}

		results = append(results, result)
	}

	// Save results to database
	for _, result := range results {
		if err := s.db.Create(&result).Error; err != nil {
			logrus.WithError(err).Error("Failed to save Confluence search result")
		}
	}

	return results, nil
}

// extractKeywords extracts meaningful keywords from a query
func (s *SearchService) extractKeywords(query string) []string {
	// Simple keyword extraction - in production, you might want more sophisticated NLP
	words := strings.Fields(strings.ToLower(query))

	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "should": true, "could": true,
		"how": true, "what": true, "where": true, "when": true, "why": true, "who": true,
	}

	var keywords []string
	for _, word := range words {
		// Remove punctuation and keep only words longer than 2 characters
		cleaned := strings.Trim(word, ".,!?;:")
		if len(cleaned) > 2 && !stopWords[cleaned] {
			keywords = append(keywords, cleaned)
		}
	}

	return keywords
}

// calculateRelevanceScore calculates a simple relevance score
func (s *SearchService) calculateRelevanceScore(content, query string) float64 {
	content = strings.ToLower(content)
	query = strings.ToLower(query)

	// Simple scoring based on keyword matches
	keywords := s.extractKeywords(query)
	score := 0.0

	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			score += 1.0
		}
	}

	// Normalize by number of keywords
	if len(keywords) > 0 {
		score = score / float64(len(keywords))
	}

	return score
}

// filterAndRankResults filters and ranks search results
func (s *SearchService) filterAndRankResults(results []storage.SearchResult) []storage.SearchResult {
	// Filter by minimum score
	var filtered []storage.SearchResult
	for _, result := range results {
		if result.Score >= s.config.SimilarityThreshold {
			filtered = append(filtered, result)
		}
	}

	// Sort by score (highest first)
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].Score < filtered[j].Score {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	// Limit results
	if len(filtered) > s.config.MaxSearchResults {
		filtered = filtered[:s.config.MaxSearchResults]
	}

	return filtered
}

// buildSlackMessageURL builds a URL to a Slack message
func (s *SearchService) buildSlackMessageURL(channelID, timestamp string) string {
	// Remove the dot from timestamp for URL
	ts := strings.ReplaceAll(timestamp, ".", "")
	return "https://slack.com/archives/" + channelID + "/p" + ts
}

// timestampToTime converts a Slack timestamp to time.Time
func (s *SearchService) timestampToTime(timestamp string) time.Time {
	// Slack timestamps are in format "1234567890.123456"
	if timestamp == "" {
		return time.Now()
	}

	// Parse the timestamp properly
	// Split on the dot to separate seconds and microseconds
	parts := strings.Split(timestamp, ".")
	if len(parts) != 2 {
		logrus.WithField("timestamp", timestamp).Warn("Invalid Slack timestamp format")
		return time.Now()
	}

	// Parse seconds since epoch
	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"timestamp": timestamp,
			"error":     err,
		}).Warn("Failed to parse timestamp seconds")
		return time.Now()
	}

	// Parse microseconds
	microseconds, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"timestamp": timestamp,
			"error":     err,
		}).Warn("Failed to parse timestamp microseconds")
		// Continue with just seconds if microseconds parsing fails
		microseconds = 0
	}

	// Convert to time.Time
	return time.Unix(seconds, microseconds*1000) // Convert microseconds to nanoseconds
}
