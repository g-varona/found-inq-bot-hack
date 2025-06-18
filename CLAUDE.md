# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based Slack bot that automatically answers team inquiries by searching historical Slack messages and Confluence documentation, then generating AI-powered responses using LiteLLM. The bot is triggered by emoji reactions (👀 by default) and responds in thread replies.

## Development Commands

### Building and Running
- `make build` - Build the application binary
- `make run` - Run directly with `go run .` (uses .env by default)
- `make run ENV_FILE=.env.test` - Run with specific environment file
- `make run-dev` - Run in development mode with detailed logging (`ENV=development`)
- `make test` - Run all tests
- `make test-e2e` - Run end-to-end tests (uses .env.test, requires server running)

### Development Tools
- `make fmt` - Format Go code with gofmt
- `make lint` - Run golangci-lint (install with `make install-tools`)
- `make vet` - Run go vet
- `make ci` - Run full CI pipeline (fmt, vet, test, build)

### Database Management
- `make db-clean` - Clean database files in data/ directory
- `make db-backup` - Backup current database with timestamp

### Setup
- `make setup` - Set up development environment (downloads deps, creates dirs, copies config example)
- `make quick-start` - Complete setup for new developers
- `make install-tools` - Install development tools (golangci-lint)

### Quality Assurance
IMPORTANT: After making code changes, always run:
- `make fmt` - Format code
- `make vet` - Check for Go issues
- `make lint` - Run linter (if available)
- `make test` - Run tests
Or use `make ci` to run the complete pipeline.

## Architecture

### Core Components
1. **main.go** - Application entry point with graceful shutdown and service initialization
2. **internal/config** - Configuration management loading from environment variables
3. **internal/handlers** - HTTP handlers for Slack webhooks and API endpoints
4. **internal/services** - Business logic services:
   - `SlackService` - Slack API integration
   - `ConfluenceService` - Confluence API integration
   - `LLMService` - LiteLLM integration for AI responses
   - `SearchService` - Multi-source search (Slack + Confluence)
   - `InquiryService` - Main orchestration service
5. **internal/storage** - Database models and operations using GORM with SQLite

### Service Dependencies
```
InquiryService
├── SearchService
│   ├── SlackService
│   ├── ConfluenceService
│   └── Database
├── SlackService
├── LLMService
└── Database
```

### Data Models
- **Inquiry** - Main inquiry record with status, message details, and response
- **SearchResult** - Results from Slack/Confluence searches with relevance scoring
- **ReactionEvent** - Emoji reaction events for auditing

### Configuration
Environment variables are loaded from `.env` file (copy from `config.env.example`). For testing, use `.env.test` file (also copied from `config.env.example`).

**Environment Files:**
- `.env` - Main development/production configuration
- `.env.test` - Test-specific configuration (used by `make test-e2e`)
- Use `ENV_FILE=.env.test make run` to run with test configuration

Key variables:
- Slack tokens and signing secret for webhook verification
- Confluence API credentials (optional)
- LiteLLM API settings
- Search parameters (similarity threshold, max results, days back)
- Database path (SQLite file in data/ directory)

### API Endpoints
- `GET /health` - Health check
- `POST /api/v1/slack/events` - Slack Events API webhook
- `POST /api/v1/slack/slash` - Slack slash commands (/inquiry-help, /inquiry-status)
- `POST /api/v1/slack/interactive` - Slack interactive components

### Workflow
1. User reacts to Slack message with trigger emoji (default: 👀)
2. Slack sends reaction event to `/api/v1/slack/events`
3. `InquiryService.ProcessReactionEvent()` orchestrates:
   - Message text extraction via Slack API
   - Search across Slack history and Confluence
   - AI response generation via LiteLLM
   - Response posting as thread reply
4. All events and results stored in SQLite database

### Testing
- Run `go test ./...` for all Go unit tests
- Test files follow `*_test.go` naming convention
- Run `make test-e2e` for end-to-end API tests (uses `.env.test` configuration)
- E2E tests require the server to be running separately
- Main test file `main_test.go` available for integration testing

### Logging
Uses logrus for structured logging:
- Production: JSON format, INFO level
- Development: Text format with timestamps, DEBUG level
- Set via `ENV` environment variable

### Docker Support
- Dockerfile available for containerized deployment
- `make docker-build` and `make docker-run` commands available
- Supports volume mounting for data persistence