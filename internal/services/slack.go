package services

import (
	"fmt"
	"time"

	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// SlackService handles Slack API interactions
type SlackService struct {
	client *slack.Client
	config *config.Config
}

// SlackMessage represents a Slack message
type SlackMessage struct {
	ID        string
	Channel   string
	User      string
	Text      string
	Timestamp string
	ThreadTS  string
}

// NewSlackService creates a new Slack service instance
func NewSlackService(cfg *config.Config) *SlackService {
	var client *slack.Client

	if cfg.SlackBotToken != "" {
		client = slack.New(cfg.SlackBotToken)
	}

	return &SlackService{
		client: client,
		config: cfg,
	}
}

// GetMessage retrieves a specific message from Slack
func (s *SlackService) GetMessage(channelID, messageTS string) (*SlackMessage, error) {
	if s.client == nil {
		return nil, fmt.Errorf("missing Slack client configuration")
	}

	// Get conversation history with the specific message
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Latest:    messageTS,
		Limit:     1,
		Inclusive: true,
	}

	history, err := s.client.GetConversationHistory(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if len(history.Messages) == 0 {
		return nil, fmt.Errorf("message not found")
	}

	msg := history.Messages[0]
	return &SlackMessage{
		ID:        msg.Timestamp,
		Channel:   channelID,
		User:      msg.User,
		Text:      msg.Text,
		Timestamp: msg.Timestamp,
		ThreadTS:  msg.ThreadTimestamp,
	}, nil
}

// SearchMessages searches for messages in a channel
func (s *SlackService) SearchMessages(query string, daysBack int) ([]SlackMessage, error) {
	if s.client == nil {
		return nil, fmt.Errorf("missing Slack client configuration")
	}

	// Calculate the date range
	now := time.Now()
	after := now.AddDate(0, 0, -daysBack)

	// Build search query
	searchQuery := fmt.Sprintf("%s in:%s after:%s",
		query,
		s.config.SlackChannelID,
		after.Format("2006-01-02"))

	// Perform search
	searchParams := slack.SearchParameters{
		Count: s.config.MaxSearchResults,
		Sort:  "timestamp",
	}

	searchResult, err := s.client.SearchMessages(searchQuery, searchParams)
	if err != nil {
		logrus.WithError(err).WithField("query", searchQuery).Error("Failed to search Slack messages")
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	// Convert to our message format
	messages := make([]SlackMessage, 0, len(searchResult.Matches))
	for _, match := range searchResult.Matches {
		messages = append(messages, SlackMessage{
			ID:        match.Timestamp,
			Channel:   match.Channel.ID,
			User:      match.User,
			Text:      match.Text,
			Timestamp: match.Timestamp,
		})
	}

	return messages, nil
}

// PostMessage sends a message to a Slack channel
func (s *SlackService) PostMessage(channelID, text string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("missing Slack client configuration")
	}

	_, timestamp, err := s.client.PostMessage(channelID, slack.MsgOptionText(text, false))
	if err != nil {
		return "", fmt.Errorf("failed to post message: %w", err)
	}

	return timestamp, nil
}

// PostThreadReply sends a reply to a message thread
func (s *SlackService) PostThreadReply(channelID, threadTS, text string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("missing Slack client configuration")
	}

	_, timestamp, err := s.client.PostMessage(
		channelID,
		slack.MsgOptionText(text, false),
		slack.MsgOptionTS(threadTS),
	)
	if err != nil {
		return "", fmt.Errorf("failed to post thread reply: %w", err)
	}

	return timestamp, nil
}

// GetUserInfo retrieves user information
func (s *SlackService) GetUserInfo(userID string) (*slack.User, error) {
	if s.client == nil {
		return nil, fmt.Errorf("missing Slack client configuration")
	}

	user, err := s.client.GetUserInfo(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return user, nil
}

// ValidateToken validates the Slack bot token
func (s *SlackService) ValidateToken() error {
	if s.client == nil {
		return fmt.Errorf("missing Slack client configuration")
	}

	_, err := s.client.AuthTest()
	if err != nil {
		return fmt.Errorf("invalid Slack token: %w", err)
	}

	return nil
}
