package services

import (
	"testing"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
)

func TestSanitizeCQLQuery(t *testing.T) {
	service := &ConfluenceService{
		config: &config.Config{},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic query",
			input:    "deployment guide",
			expected: "deployment guide",
		},
		{
			name:     "query with CQL operators",
			input:    "deployment AND guide OR setup",
			expected: "deployment guide setup",
		},
		{
			name:     "query with parentheses",
			input:    "(deployment OR guide) AND setup",
			expected: "deployment guide setup",
		},
		{
			name:     "query with quotes",
			input:    `"deployment guide" AND setup`,
			expected: "deployment guide setup",
		},
		{
			name:     "query with special characters",
			input:    "deploy* ~guide [setup]",
			expected: "deploy guide setup",
		},
		{
			name:     "query with backslashes",
			input:    "deploy\\guide\\setup",
			expected: "deploy guide setup",
		},
		{
			name:     "very long query",
			input:    "this is a very very very very very very very very very very very very very long query that should be truncated",
			expected: "this is a very very very very very very very very very very very very very long query that should be",
		},
		{
			name:     "empty query",
			input:    "",
			expected: "",
		},
		{
			name:     "query with only special characters",
			input:    "() [] {} \" ' \\ ~ * ?",
			expected: "",
		},
		{
			name:     "query with multiple spaces",
			input:    "deployment     guide     setup",
			expected: "deployment guide setup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.sanitizeCQLQuery(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeCQLQuery(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeCQLQuery_Length(t *testing.T) {
	service := &ConfluenceService{
		config: &config.Config{},
	}

	// Test that very long queries are truncated to 100 characters
	longQuery := "word " // 5 characters
	for i := 0; i < 25; i++ {
		longQuery += "word " // Build a 125 character string
	}

	result := service.sanitizeCQLQuery(longQuery)
	if len(result) > 100 {
		t.Errorf("Expected sanitized query to be <= 100 characters, got %d", len(result))
	}
}

func TestSanitizeCQLQuery_PreventInjection(t *testing.T) {
	service := &ConfluenceService{
		config: &config.Config{},
	}

	// Test potential CQL injection attempts
	injectionAttempts := []string{
		"normal OR (malicious)",
		"text AND space!=DOCS",
		"search) OR (1=1",
		"query\" OR \"admin",
		"search* AND NOT restricted",
	}

	for _, attempt := range injectionAttempts {
		result := service.sanitizeCQLQuery(attempt)
		
		// Check that dangerous operators are removed
		dangerousChars := []string{"(", ")", "[", "]", "{", "}", "\"", "'", "\\", "~", "*", "?"}
		for _, char := range dangerousChars {
			if contains(result, char) {
				t.Errorf("Sanitized query still contains dangerous character '%s': %q", char, result)
			}
		}
		
		// Check that AND, OR, NOT operators are removed
		dangerousOps := []string{" AND ", " OR ", " NOT "}
		for _, op := range dangerousOps {
			if contains(result, op) {
				t.Errorf("Sanitized query still contains dangerous operator '%s': %q", op, result)
			}
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}