package storage

import (
	"time"

	"gorm.io/gorm"
)

// Inquiry represents an inquiry received from Slack
type Inquiry struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Slack message details
	MessageID   string `gorm:"uniqueIndex;not null" json:"message_id"`
	ChannelID   string `json:"channel_id"`
	UserID      string `json:"user_id"`
	MessageText string `json:"message_text"`
	Timestamp   string `json:"timestamp"`

	// Processing details
	Status          string     `json:"status"` // pending, processing, completed, failed
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ResponseSent    bool       `json:"response_sent"`
	ResponseText    string     `json:"response_text"`
	ThreadTimestamp string     `json:"thread_timestamp"`

	// Search results relationship
	SearchResults []SearchResult `gorm:"foreignKey:InquiryID;constraint:OnDelete:CASCADE" json:"search_results,omitempty"`
}

// SearchResult represents a search result from Slack or Confluence
type SearchResult struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	InquiryID uint `gorm:"not null;index" json:"inquiry_id"`

	// Source information
	Source   string `json:"source"`    // slack, confluence
	SourceID string `json:"source_id"` // message timestamp or page ID
	Title    string `json:"title"`
	Content  string `json:"content"`
	URL      string `json:"url"`

	// Relevance scoring
	Score float64 `json:"score"`

	// Additional metadata
	Author      string    `json:"author"`
	CreatedDate time.Time `json:"created_date"`
}

// ReactionEvent represents a reaction event from Slack
type ReactionEvent struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Event details
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id"`
	UserID    string `json:"user_id"`
	Reaction  string `json:"reaction"`
	EventType string `json:"event_type"` // added, removed
	Timestamp string `json:"timestamp"`

	// Processing status
	Processed bool  `json:"processed"`
	InquiryID *uint `json:"inquiry_id,omitempty"`
}
