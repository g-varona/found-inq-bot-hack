package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/storage"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InquiryService orchestrates the entire inquiry processing pipeline
type InquiryService struct {
	search *SearchService
	slack  *SlackService
	llm    *LLMService
	db     *gorm.DB
	config *config.Config
}

// NewInquiryService creates a new inquiry service instance
func NewInquiryService(search *SearchService, slack *SlackService, llm *LLMService, db *gorm.DB, cfg *config.Config) *InquiryService {
	return &InquiryService{
		search: search,
		slack:  slack,
		llm:    llm,
		db:     db,
		config: cfg,
	}
}

// ProcessInquiry processes an inquiry from start to finish
func (s *InquiryService) ProcessInquiry(ctx context.Context, messageID, channelID, userID, messageText, timestamp string) error {
	logrus.WithFields(logrus.Fields{
		"message_id": messageID,
		"channel_id": channelID,
		"user_id":    userID,
	}).Info("Starting inquiry processing")

	// Create inquiry record
	inquiry := &storage.Inquiry{
		MessageID:   messageID,
		ChannelID:   channelID,
		UserID:      userID,
		MessageText: messageText,
		Timestamp:   timestamp,
		Status:      "pending",
	}

	if err := s.db.Create(inquiry).Error; err != nil {
		logrus.WithError(err).Error("Failed to create inquiry record")
		return fmt.Errorf("failed to create inquiry: %w", err)
	}

	// Update status to processing
	inquiry.Status = "processing"
	s.db.Save(inquiry)

	// Search for relevant information
	searchResults, err := s.search.SearchAll(ctx, messageText, inquiry.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to search for relevant information")
		inquiry.Status = "failed"
		s.db.Save(inquiry)
		return fmt.Errorf("search failed: %w", err)
	}

	// Generate AI response
	response, err := s.llm.GenerateResponse(ctx, inquiry, searchResults)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate AI response")

		// Send fallback response
		fallbackResponse := s.generateFallbackResponse(searchResults)
		if err := s.sendResponse(ctx, inquiry, fallbackResponse); err != nil {
			logrus.WithError(err).Error("Failed to send fallback response")
		}

		inquiry.Status = "failed"
		inquiry.ResponseText = fallbackResponse
		s.db.Save(inquiry)
		return fmt.Errorf("AI response generation failed: %w", err)
	}

	// Send response to Slack
	if err := s.sendResponse(ctx, inquiry, response); err != nil {
		logrus.WithError(err).Error("Failed to send response to Slack")
		inquiry.Status = "failed"
		inquiry.ResponseText = response
		s.db.Save(inquiry)
		return fmt.Errorf("failed to send response: %w", err)
	}

	// Update inquiry record
	now := time.Now()
	inquiry.Status = "completed"
	inquiry.ProcessedAt = &now
	inquiry.ResponseSent = true
	inquiry.ResponseText = response
	s.db.Save(inquiry)

	logrus.WithFields(logrus.Fields{
		"inquiry_id":      inquiry.ID,
		"search_results":  len(searchResults),
		"response_length": len(response),
	}).Info("Inquiry processing completed successfully")

	return nil
}

// sendResponse sends the response to Slack as a thread reply
func (s *InquiryService) sendResponse(ctx context.Context, inquiry *storage.Inquiry, response string) error {
	_, cancelFn := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancelFn()
	// Format the response with a header
	formattedResponse := fmt.Sprintf("🤖 *AI Assistant Response*\n\n%s", response)

	// Send as a thread reply to the original message
	threadTS, err := s.slack.PostThreadReply(inquiry.ChannelID, inquiry.Timestamp, formattedResponse)
	if err != nil {
		return err
	}

	// Update inquiry with thread timestamp
	inquiry.ThreadTimestamp = threadTS
	s.db.Save(inquiry)

	return nil
}

// generateFallbackResponse generates a fallback response when AI fails
func (s *InquiryService) generateFallbackResponse(searchResults []storage.SearchResult) string {
	if len(searchResults) == 0 {
		return "I couldn't find relevant information to answer your inquiry. You might want to check our documentation or reach out to the relevant team directly."
	}

	response := "I found some potentially relevant information:\n\n"

	for i, result := range searchResults {
		if i >= 3 { // Limit to top 3 results
			break
		}

		response += fmt.Sprintf("• **%s** (%s)\n", result.Title, result.Source)
		if result.Content != "" {
			// Truncate content
			content := result.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			response += fmt.Sprintf("  %s\n", content)
		}
		if result.URL != "" {
			response += fmt.Sprintf("  %s\n", result.URL)
		}
		response += "\n"
	}

	response += "Please review these resources or contact the relevant team for more specific assistance."
	return response
}

// GetInquiry retrieves an inquiry by ID
func (s *InquiryService) GetInquiry(inquiryID uint) (*storage.Inquiry, error) {
	var inquiry storage.Inquiry
	if err := s.db.Preload("SearchResults").First(&inquiry, inquiryID).Error; err != nil {
		return nil, err
	}
	return &inquiry, nil
}

// GetInquiryByMessageID retrieves an inquiry by message ID
func (s *InquiryService) GetInquiryByMessageID(messageID string) (*storage.Inquiry, error) {
	var inquiry storage.Inquiry
	if err := s.db.Preload("SearchResults").Where("message_id = ?", messageID).First(&inquiry).Error; err != nil {
		return nil, err
	}
	return &inquiry, nil
}

// ListRecentInquiries lists recent inquiries
func (s *InquiryService) ListRecentInquiries(limit int) ([]storage.Inquiry, error) {
	var inquiries []storage.Inquiry
	if err := s.db.Order("created_at DESC").Limit(limit).Find(&inquiries).Error; err != nil {
		return nil, err
	}
	return inquiries, nil
}

// ProcessReactionEvent processes a reaction event from Slack
func (s *InquiryService) ProcessReactionEvent(ctx context.Context, messageID, channelID, userID, reaction, eventType, timestamp string) error {
	// Only process if it's the trigger emoji being added
	if reaction != s.config.TriggerEmoji || eventType != "added" {
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"message_id": messageID,
		"channel_id": channelID,
		"reaction":   reaction,
	}).Info("Processing trigger emoji reaction")

	// Record the reaction event
	reactionEvent := &storage.ReactionEvent{
		MessageID: messageID,
		ChannelID: channelID,
		UserID:    userID,
		Reaction:  reaction,
		EventType: eventType,
		Timestamp: timestamp,
		Processed: false,
	}

	if err := s.db.Create(reactionEvent).Error; err != nil {
		logrus.WithError(err).Error("Failed to create reaction event record")
		return err
	}

	// Check if we've already processed this message
	var existingInquiry storage.Inquiry
	if err := s.db.Where("message_id = ?", messageID).First(&existingInquiry).Error; err == nil {
		logrus.Info("Message already processed, skipping")
		reactionEvent.Processed = true
		reactionEvent.InquiryID = &existingInquiry.ID
		s.db.Save(reactionEvent)
		return nil
	}

	// Get the original message
	slackMessage, err := s.slack.GetMessage(channelID, messageID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get original message")
		return err
	}

	if slackMessage.Text == "" {
		logrus.Info("Slack message is empty")
		return fmt.Errorf("empty Slack message")
	}

	// Process the inquiry
	if err := s.ProcessInquiry(ctx, messageID, channelID, slackMessage.User, slackMessage.Text, slackMessage.Timestamp); err != nil {
		logrus.WithError(err).Error("Failed to process inquiry")
		return err
	}

	// Update reaction event as processed
	if inquiry, err := s.GetInquiryByMessageID(messageID); err == nil {
		reactionEvent.Processed = true
		reactionEvent.InquiryID = &inquiry.ID
		s.db.Save(reactionEvent)
	}

	return nil
}
