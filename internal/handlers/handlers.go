package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/services"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests
type Handler struct {
	inquiry *services.InquiryService
	slack   *services.SlackService
	config  *config.Config
}

// SlackEvent represents a Slack event
type SlackEvent struct {
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Type      string `json:"type"`
	Event     struct {
		Type           string `json:"type"`
		Channel        string `json:"channel"`
		User           string `json:"user"`
		Text           string `json:"text"`
		Timestamp      string `json:"ts"`
		EventTimestamp string `json:"event_ts"`
		Reaction       string `json:"reaction"`
		Item           struct {
			Type    string `json:"type"`
			Channel string `json:"channel"`
			TS      string `json:"ts"`
		} `json:"item"`
	} `json:"event"`
}

// New creates a new handler instance
func New(inquiry *services.InquiryService, slack *services.SlackService, cfg *config.Config) *Handler {
	return &Handler{
		inquiry: inquiry,
		slack:   slack,
		config:  cfg,
	}
}

// HandleSlackEvents handles Slack Events API webhooks
func (h *Handler) HandleSlackEvents(c *gin.Context) {
	// Verify Slack signature
	if !h.verifySlackSignature(c.Request) {
		logrus.Error("Invalid Slack signature bleeeh")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var event SlackEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		logrus.WithError(err).Error("Failed to parse Slack event")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Handle URL verification
	if event.Type == "url_verification" {
		c.JSON(http.StatusOK, gin.H{"challenge": event.Challenge})
		return
	}

	// Handle events
	if event.Type == "event_callback" {
		go h.processSlackEvent(event)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleSlashCommands handles Slack slash commands
func (h *Handler) HandleSlashCommands(c *gin.Context) {
	// Verify Slack signature
	if !h.verifySlackSignature(c.Request) {
		logrus.Error("Invalid Slack signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// Parse form data
	command := c.PostForm("command")
	text := c.PostForm("text")
	userID := c.PostForm("user_id")
	channelID := c.PostForm("channel_id")

	logrus.WithFields(logrus.Fields{
		"command":    command,
		"text":       text,
		"user_id":    userID,
		"channel_id": channelID,
	}).Info("Received slash command")

	// Handle different commands
	switch command {
	case "/inquiry-help":
		response := h.generateHelpResponse()
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          response,
		})
	case "/inquiry-status":
		response := h.generateStatusResponse()
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          response,
		})
	default:
		c.JSON(http.StatusOK, gin.H{
			"response_type": "ephemeral",
			"text":          "Unknown command. Use `/inquiry-help` for help.",
		})
	}
}

// HandleInteractiveComponents handles Slack interactive components
func (h *Handler) HandleInteractiveComponents(c *gin.Context) {
	// Verify Slack signature
	if !h.verifySlackSignature(c.Request) {
		logrus.Error("Invalid Slack signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	logrus.Info("Received interactive component")

	// Parse the payload if needed
	// For now, just acknowledge
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// processSlackEvent processes different types of Slack events
func (h *Handler) processSlackEvent(event SlackEvent) {
	ctx := context.Background()

	switch event.Event.Type {
	case "reaction_added":
		h.handleReactionEvent(ctx, event, "added")
	case "reaction_removed":
		h.handleReactionEvent(ctx, event, "removed")
	case "message":
		// Handle direct message events if needed
		logrus.WithField("event", event).Debug("Received message event")
	default:
		logrus.WithField("event_type", event.Event.Type).Debug("Unhandled event type")
	}
}

// handleReactionEvent handles emoji reaction events
func (h *Handler) handleReactionEvent(ctx context.Context, event SlackEvent, eventType string) {
	if event.Event.Item.Type != "message" {
		return
	}

	err := h.inquiry.ProcessReactionEvent(
		ctx,
		event.Event.Item.TS,        // message timestamp
		event.Event.Item.Channel,   // channel ID
		event.Event.User,           // user who added reaction
		event.Event.Reaction,       // emoji name
		eventType,                  // added or removed
		event.Event.EventTimestamp, // event timestamp
	)

	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"message_ts": event.Event.Item.TS,
			"channel":    event.Event.Item.Channel,
			"reaction":   event.Event.Reaction,
			"event_type": eventType,
		}).Error("Failed to process reaction event")
	}
}

// verifySlackSignature verifies the Slack request signature
func (h *Handler) verifySlackSignature(r *http.Request) bool {
	if h.config.SlackSigningSecret == "" {
		logrus.Error("Slack signing secret not configured - signature verification required for security")
		return false
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	if timestamp == "" {
		logrus.Error("Missing X-Slack-Request-Timestamp header")
		return false
	}

	// Check if timestamp is recent (within 5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse timestamp")
		return false
	}

	if time.Now().Unix()-ts > 300 {
		logrus.Error("Request timestamp too old (>5 minutes)")
		return false
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read request body")
		return false
	}

	// Create new reader for the body
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Create signature
	sig := "v0=" + h.calculateSignature(timestamp, string(body))

	// Compare with received signature
	receivedSig := r.Header.Get("X-Slack-Signature")

	return hmac.Equal([]byte(sig), []byte(receivedSig))
}

// calculateSignature calculates the HMAC signature
func (h *Handler) calculateSignature(timestamp, body string) string {
	baseString := "v0:" + timestamp + ":" + body
	mac := hmac.New(sha256.New, []byte(h.config.SlackSigningSecret))
	mac.Write([]byte(baseString))
	return hex.EncodeToString(mac.Sum(nil))
}

// generateHelpResponse generates help text for the slash command
func (h *Handler) generateHelpResponse() string {
	return "*Foundation Inquiry Bot Help*\n\n" +
		"This bot automatically answers team inquiries by searching through past Slack discussions and Confluence documentation.\n\n" +
		"*How to use:*\n" +
		"1. React to any message with the :" + h.config.TriggerEmoji + ": emoji to trigger an AI-powered response\n" +
		"2. The bot will search for similar discussions and documentation\n" +
		"3. An AI-generated response will be posted as a thread reply\n\n" +
		"*Commands:*\n" +
		"• `/inquiry-help` - Show this help message\n" +
		"• `/inquiry-status` - Show bot status and recent activity\n\n" +
		"*Features:*\n" +
		"• Searches Slack messages from the last 90 days\n" +
		"• Searches relevant Confluence pages\n" +
		"• Uses AI to generate comprehensive responses\n" +
		"• Maintains conversation history for learning\n\n" +
		"For questions or issues, contact the Foundation team."
}

// generateStatusResponse generates status information
func (h *Handler) generateStatusResponse() string {
	// Get recent inquiries
	inquiries, err := h.inquiry.ListRecentInquiries(5)
	if err != nil {
		return "❌ Error retrieving status information"
	}

	response := "*Foundation Inquiry Bot Status*\n\n"
	response += "✅ Bot is running and operational\n\n"

	if len(inquiries) == 0 {
		response += "No recent inquiries processed."
	} else {
		response += fmt.Sprintf("*Recent Activity* (last %d inquiries):\n", len(inquiries))
		for _, inquiry := range inquiries {
			status := "❓"
			switch inquiry.Status {
			case "completed":
				status = "✅"
			case "failed":
				status = "❌"
			case "processing":
				status = "⏳"
			}

			response += fmt.Sprintf("%s %s - %s\n%s\n",
				status,
				inquiry.CreatedAt.Format("Jan 2 15:04"),
				inquiry.Status,
				inquiry.MessageText)
		}
	}

	return response
}
