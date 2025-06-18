package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/config"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/handlers"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/services"
	"github.com/kouzoh/foundation-inquiry-slack-bot/internal/storage"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using system environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Set up logging
	setupLogging(cfg.Env)

	// Initialize database
	db, err := storage.InitDB(cfg.DBPath)
	if err != nil {
		logrus.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize services
	slackService := services.NewSlackService(cfg)
	confluenceService := services.NewConfluenceService(cfg)
	llmService := services.NewLLMService(cfg)
	searchService := services.NewSearchService(slackService, confluenceService, db, cfg)
	inquiryService := services.NewInquiryService(searchService, slackService, llmService, db, cfg)

	// Initialize handlers
	handlers := handlers.New(inquiryService, slackService, cfg)

	// Set up router
	router := setupRouter(handlers, cfg)

	// Create server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Starting Foundation Inquiry Slack Bot on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}

func setupLogging(env string) {
	if env == "production" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func setupRouter(h *handlers.Handler, cfg *config.Config) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "foundation-inquiry-slack-bot",
		})
	})

	// Slack webhook endpoints
	api := router.Group("/api/v1")
	{
		api.POST("/slack/events", h.HandleSlackEvents)
		api.POST("/slack/slash", h.HandleSlashCommands)
		api.POST("/slack/interactive", h.HandleInteractiveComponents)
	}

	return router
}
