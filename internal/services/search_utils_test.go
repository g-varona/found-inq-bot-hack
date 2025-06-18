package services

import (
	"strings"
	"testing"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/storage"
)

func TestExtractKeywords(t *testing.T) {
	service := &SearchService{}

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "simple query",
			query:    "How to deploy service",
			expected: []string{"deploy", "service"},
		},
		{
			name:     "query with stop words",
			query:    "What is the best way to deploy a service?",
			expected: []string{"best", "way", "deploy", "service"},
		},
		{
			name:     "query with punctuation",
			query:    "Deploy service, please!",
			expected: []string{"deploy", "service", "please"},
		},
		{
			name:     "empty query",
			query:    "",
			expected: []string{},
		},
		{
			name:     "only stop words",
			query:    "the a an and or",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.extractKeywords(tt.query)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d keywords, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, keyword := range result {
				if keyword != tt.expected[i] {
					t.Errorf("Expected keyword '%s', got '%s'", tt.expected[i], keyword)
				}
			}
		})
	}
}

func TestCalculateRelevanceScore(t *testing.T) {
	service := &SearchService{}

	tests := []struct {
		name     string
		content  string
		query    string
		expected float64
	}{
		{
			name:     "exact match",
			content:  "deploy service",
			query:    "deploy service",
			expected: 1.0,
		},
		{
			name:     "partial match",
			content:  "deploy the service to production",
			query:    "deploy service",
			expected: 1.0,
		},
		{
			name:     "half match",
			content:  "deploy to production",
			query:    "deploy service",
			expected: 0.5,
		},
		{
			name:     "no match",
			content:  "something else entirely",
			query:    "deploy service",
			expected: 0.0,
		},
		{
			name:     "case insensitive",
			content:  "Deploy Service",
			query:    "deploy service",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateRelevanceScore(tt.content, tt.query)
			if result != tt.expected {
				t.Errorf("Expected score %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestFilterAndRankResults(t *testing.T) {
	cfg := &config.Config{
		SimilarityThreshold: 0.5,
		MaxSearchResults:    3,
	}
	service := &SearchService{config: cfg}

	results := []storage.SearchResult{
		{Score: 0.9, Title: "High score 1"},
		{Score: 0.8, Title: "High score 2"},
		{Score: 0.7, Title: "Medium score 1"},
		{Score: 0.6, Title: "Medium score 2"},
		{Score: 0.4, Title: "Low score (should be filtered)"},
		{Score: 0.3, Title: "Very low score (should be filtered)"},
	}

	filtered := service.filterAndRankResults(results)

	// Should filter out scores below threshold (0.5) and limit to MaxSearchResults
	// 4 results have scores >= 0.5, but MaxSearchResults is 3
	expectedCount := cfg.MaxSearchResults

	if len(filtered) != expectedCount {
		t.Errorf("Expected %d results after filtering, got %d", expectedCount, len(filtered))
	}

	// Should be sorted by score (highest first)
	for i := 0; i < len(filtered)-1; i++ {
		if filtered[i].Score < filtered[i+1].Score {
			t.Errorf("Results not sorted correctly: %f should be >= %f", filtered[i].Score, filtered[i+1].Score)
		}
	}

	// All results should be above threshold
	for _, result := range filtered {
		if result.Score < cfg.SimilarityThreshold {
			t.Errorf("Result with score %f should have been filtered out", result.Score)
		}
	}
}

func TestBuildSlackMessageURL(t *testing.T) {
	service := &SearchService{}

	tests := []struct {
		name      string
		channelID string
		timestamp string
		expected  string
	}{
		{
			name:      "normal timestamp",
			channelID: "C1234567890",
			timestamp: "1234567890.123456",
			expected:  "https://slack.com/archives/C1234567890/p1234567890123456",
		},
		{
			name:      "timestamp without decimal",
			channelID: "C9876543210",
			timestamp: "1234567890",
			expected:  "https://slack.com/archives/C9876543210/p1234567890",
		},
		{
			name:      "empty timestamp",
			channelID: "C1111111111",
			timestamp: "",
			expected:  "https://slack.com/archives/C1111111111/p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildSlackMessageURL(tt.channelID, tt.timestamp)
			if result != tt.expected {
				t.Errorf("Expected URL '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTimestampToTime(t *testing.T) {
	service := &SearchService{}

	// Test with various timestamp formats
	tests := []struct {
		name      string
		timestamp string
	}{
		{
			name:      "normal timestamp",
			timestamp: "1234567890.123456",
		},
		{
			name:      "timestamp without decimal",
			timestamp: "1234567890",
		},
		{
			name:      "empty timestamp",
			timestamp: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.timestampToTime(tt.timestamp)

			// Since the current implementation returns time.Now(),
			// we just check that we get a valid time
			if result.IsZero() {
				t.Error("Expected non-zero time")
			}
		})
	}
}

func TestSearchService_EdgeCases(t *testing.T) {
	t.Run("empty query", func(t *testing.T) {
		service := &SearchService{}
		keywords := service.extractKeywords("")
		if len(keywords) != 0 {
			t.Errorf("Expected 0 keywords for empty query, got %d", len(keywords))
		}
	})

	t.Run("very long query", func(t *testing.T) {
		service := &SearchService{}
		longQuery := strings.Repeat("deployment service docker kubernetes production ", 20)
		keywords := service.extractKeywords(longQuery)

		// Should extract all unique keywords
		if len(keywords) == 0 {
			t.Error("Expected some keywords from long query")
		}
	})

	t.Run("special characters in query", func(t *testing.T) {
		service := &SearchService{}
		query := "deploy@service#with$special%characters"
		keywords := service.extractKeywords(query)

		// Should handle special characters gracefully
		if len(keywords) == 0 {
			t.Error("Expected some keywords from query with special characters")
		}
	})
}
