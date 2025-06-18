package storage

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDatabase(t *testing.T) *gorm.DB {
	config := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Execute pragma statements for in-memory database
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Auto migrate the schema in specific order
	if err := db.AutoMigrate(&Inquiry{}); err != nil {
		t.Fatalf("Failed to migrate Inquiry: %v", err)
	}

	if err := db.AutoMigrate(&SearchResult{}); err != nil {
		t.Fatalf("Failed to migrate SearchResult: %v", err)
	}

	if err := db.AutoMigrate(&ReactionEvent{}); err != nil {
		t.Fatalf("Failed to migrate ReactionEvent: %v", err)
	}

	return db
}

func TestDatabase_CreateInquiry(t *testing.T) {
	db := setupTestDatabase(t)

	inquiry := &Inquiry{
		MessageID:   "msg-123",
		ChannelID:   "C1234567890",
		UserID:      "U1234567890",
		MessageText: "How do I deploy the service?",
		Timestamp:   "1234567890.123456",
		Status:      "pending",
	}

	result := db.Create(inquiry)
	if result.Error != nil {
		t.Errorf("Failed to create inquiry: %v", result.Error)
	}

	if inquiry.ID == 0 {
		t.Error("Expected inquiry ID to be set after creation")
	}

	// Verify it was saved
	var savedInquiry Inquiry
	db.First(&savedInquiry, inquiry.ID)
	if savedInquiry.MessageID != "msg-123" {
		t.Errorf("Expected MessageID 'msg-123', got '%s'", savedInquiry.MessageID)
	}
}

func TestDatabase_CreateSearchResult(t *testing.T) {
	db := setupTestDatabase(t)

	// First create an inquiry
	inquiry := &Inquiry{
		MessageID:   "msg-123",
		ChannelID:   "C1234567890",
		UserID:      "U1234567890",
		MessageText: "Test inquiry",
		Status:      "pending",
	}
	db.Create(inquiry)

	searchResult := &SearchResult{
		InquiryID:   inquiry.ID,
		Source:      "slack",
		SourceID:    "1234567890.123456",
		Title:       "Previous Discussion",
		Content:     "This is a previous discussion",
		URL:         "https://slack.com/archives/C123/p123456",
		Score:       0.85,
		Author:      "john.doe",
		CreatedDate: time.Now(),
	}

	result := db.Create(searchResult)
	if result.Error != nil {
		t.Errorf("Failed to create search result: %v", result.Error)
	}

	if searchResult.ID == 0 {
		t.Error("Expected search result ID to be set after creation")
	}

	// Verify it was saved
	var savedResult SearchResult
	db.First(&savedResult, searchResult.ID)
	if savedResult.Source != "slack" {
		t.Errorf("Expected Source 'slack', got '%s'", savedResult.Source)
	}
}

func TestDatabase_CreateReactionEvent(t *testing.T) {
	db := setupTestDatabase(t)

	reactionEvent := &ReactionEvent{
		MessageID: "msg-123",
		ChannelID: "C1234567890",
		UserID:    "U1234567890",
		Reaction:  "eyes",
		EventType: "added",
		Timestamp: "1234567890.123456",
		Processed: false,
	}

	result := db.Create(reactionEvent)
	if result.Error != nil {
		t.Errorf("Failed to create reaction event: %v", result.Error)
	}

	if reactionEvent.ID == 0 {
		t.Error("Expected reaction event ID to be set after creation")
	}

	// Verify it was saved
	var savedEvent ReactionEvent
	db.First(&savedEvent, reactionEvent.ID)
	if savedEvent.Reaction != "eyes" {
		t.Errorf("Expected Reaction 'eyes', got '%s'", savedEvent.Reaction)
	}
}

func TestDatabase_InquiryWithSearchResults(t *testing.T) {
	db := setupTestDatabase(t)

	// Create inquiry
	inquiry := &Inquiry{
		MessageID:   "msg-123",
		ChannelID:   "C1234567890",
		UserID:      "U1234567890",
		MessageText: "Test inquiry",
		Status:      "pending",
	}
	db.Create(inquiry)

	// Create search results
	searchResults := []SearchResult{
		{
			InquiryID:   inquiry.ID,
			Source:      "slack",
			SourceID:    "123.456",
			Title:       "Slack Result",
			Content:     "Slack content",
			Score:       0.8,
			Author:      "user1",
			CreatedDate: time.Now(),
		},
		{
			InquiryID:   inquiry.ID,
			Source:      "confluence",
			SourceID:    "789",
			Title:       "Confluence Result",
			Content:     "Confluence content",
			Score:       0.9,
			Author:      "user2",
			CreatedDate: time.Now(),
		},
	}

	for _, result := range searchResults {
		db.Create(&result)
	}

	// Load inquiry with search results
	var loadedInquiry Inquiry
	db.Preload("SearchResults").First(&loadedInquiry, inquiry.ID)

	if len(loadedInquiry.SearchResults) != 2 {
		t.Errorf("Expected 2 search results, got %d", len(loadedInquiry.SearchResults))
	}

	if loadedInquiry.SearchResults[0].Source != "slack" {
		t.Errorf("Expected first search result to be from Slack, got %s", loadedInquiry.SearchResults[0].Source)
	}

	if loadedInquiry.SearchResults[1].Source != "confluence" {
		t.Errorf("Expected second search result to be from Confluence, got %s", loadedInquiry.SearchResults[1].Source)
	}
}

func TestDatabase_QueryInquiries(t *testing.T) {
	db := setupTestDatabase(t)

	// Create multiple inquiries
	inquiries := []Inquiry{
		{
			MessageID:   "msg-1",
			ChannelID:   "C1234567890",
			UserID:      "U1234567890",
			MessageText: "First inquiry",
			Status:      "pending",
		},
		{
			MessageID:   "msg-2",
			ChannelID:   "C1234567890",
			UserID:      "U9876543210",
			MessageText: "Second inquiry",
			Status:      "completed",
		},
		{
			MessageID:   "msg-3",
			ChannelID:   "C1234567890",
			UserID:      "U1234567890",
			MessageText: "Third inquiry",
			Status:      "failed",
		},
	}

	for _, inquiry := range inquiries {
		db.Create(&inquiry)
	}

	// Query by status
	var pendingInquiries []Inquiry
	db.Where("status = ?", "pending").Find(&pendingInquiries)
	if len(pendingInquiries) != 1 {
		t.Errorf("Expected 1 pending inquiry, got %d", len(pendingInquiries))
	}

	// Query by user
	var userInquiries []Inquiry
	db.Where("user_id = ?", "U1234567890").Find(&userInquiries)
	if len(userInquiries) != 2 {
		t.Errorf("Expected 2 inquiries for user U1234567890, got %d", len(userInquiries))
	}

	// Query by channel
	var channelInquiries []Inquiry
	db.Where("channel_id = ?", "C1234567890").Find(&channelInquiries)
	if len(channelInquiries) != 3 {
		t.Errorf("Expected 3 inquiries for channel C1234567890, got %d", len(channelInquiries))
	}
}

func TestDatabase_UpdateInquiry(t *testing.T) {
	db := setupTestDatabase(t)

	// Create inquiry
	inquiry := &Inquiry{
		MessageID:    "msg-123",
		ChannelID:    "C1234567890",
		UserID:       "U1234567890",
		MessageText:  "Test inquiry",
		Status:       "pending",
		ResponseSent: false,
	}
	db.Create(inquiry)

	// Update inquiry
	processedTime := time.Now()
	inquiry.Status = "completed"
	inquiry.ProcessedAt = &processedTime
	inquiry.ResponseSent = true
	inquiry.ResponseText = "This is the response"
	inquiry.ThreadTimestamp = "1234567891.123456"

	result := db.Save(inquiry)
	if result.Error != nil {
		t.Errorf("Failed to update inquiry: %v", result.Error)
	}

	// Verify update
	var updatedInquiry Inquiry
	db.First(&updatedInquiry, inquiry.ID)
	if updatedInquiry.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", updatedInquiry.Status)
	}
	if !updatedInquiry.ResponseSent {
		t.Error("Expected ResponseSent to be true")
	}
	if updatedInquiry.ResponseText != "This is the response" {
		t.Errorf("Expected ResponseText 'This is the response', got '%s'", updatedInquiry.ResponseText)
	}
	if updatedInquiry.ThreadTimestamp != "1234567891.123456" {
		t.Errorf("Expected ThreadTimestamp '1234567891.123456', got '%s'", updatedInquiry.ThreadTimestamp)
	}
}

func TestDatabase_DeleteInquiry(t *testing.T) {
	db := setupTestDatabase(t)

	// Create inquiry
	inquiry := &Inquiry{
		MessageID:   "msg-123",
		ChannelID:   "C1234567890",
		UserID:      "U1234567890",
		MessageText: "Test inquiry",
		Status:      "pending",
	}
	db.Create(inquiry)

	// Delete inquiry
	result := db.Delete(inquiry)
	if result.Error != nil {
		t.Errorf("Failed to delete inquiry: %v", result.Error)
	}

	// Verify deletion
	var count int64
	db.Model(&Inquiry{}).Where("id = ?", inquiry.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected inquiry to be deleted, but found %d records", count)
	}
}

func TestDatabase_UniqueConstraints(t *testing.T) {
	db := setupTestDatabase(t)

	// Create first inquiry
	inquiry1 := &Inquiry{
		MessageID:   "msg-123",
		ChannelID:   "C1234567890",
		UserID:      "U1234567890",
		MessageText: "First inquiry",
		Status:      "pending",
	}
	result1 := db.Create(inquiry1)
	if result1.Error != nil {
		t.Errorf("Failed to create first inquiry: %v", result1.Error)
	}

	// Try to create second inquiry with same MessageID
	inquiry2 := &Inquiry{
		MessageID:   "msg-123", // Same MessageID
		ChannelID:   "C1234567890",
		UserID:      "U9876543210",
		MessageText: "Second inquiry",
		Status:      "pending",
	}
	result2 := db.Create(inquiry2)
	if result2.Error == nil {
		t.Error("Expected error when creating inquiry with duplicate MessageID")
	}
}

func TestDatabase_Timestamps(t *testing.T) {
	db := setupTestDatabase(t)

	beforeCreate := time.Now()

	inquiry := &Inquiry{
		MessageID:   "msg-123",
		ChannelID:   "C1234567890",
		UserID:      "U1234567890",
		MessageText: "Test inquiry",
		Status:      "pending",
	}
	db.Create(inquiry)

	afterCreate := time.Now()

	// Check that GORM automatically set CreatedAt and UpdatedAt
	if inquiry.CreatedAt.Before(beforeCreate) || inquiry.CreatedAt.After(afterCreate) {
		t.Error("CreatedAt should be set to current time during creation")
	}
	if inquiry.UpdatedAt.Before(beforeCreate) || inquiry.UpdatedAt.After(afterCreate) {
		t.Error("UpdatedAt should be set to current time during creation")
	}

	// Store original CreatedAt for comparison before update
	originalCreatedAt := inquiry.CreatedAt

	// Update the inquiry
	beforeUpdate := time.Now()
	inquiry.Status = "completed"
	db.Save(inquiry)
	afterUpdate := time.Now()

	// UpdatedAt should be changed, CreatedAt should remain the same
	if inquiry.UpdatedAt.Before(beforeUpdate) || inquiry.UpdatedAt.After(afterUpdate) {
		t.Error("UpdatedAt should be updated during save")
	}
	if !inquiry.CreatedAt.Equal(originalCreatedAt) {
		t.Error("CreatedAt should not change during updates")
	}
}
