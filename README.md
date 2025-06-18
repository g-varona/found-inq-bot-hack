# Inquiry Slack Bot

An intelligent Slack bot that automatically answers team inquiries by searching through historical Slack messages and Confluence documentation, then generating AI-powered responses using LiteLLM.

## Features

- **🤖 AI-Powered Responses**: Uses LiteLLM to generate intelligent responses based on context
- **🔍 Multi-Source Search**: Searches both Slack message history and Confluence pages
- **⚡ Emoji Trigger**: Simply react with 👀 (eyes) emoji or the configured trigger emoji to trigger the bot
- **🧵 Thread Replies**: Responses are posted as thread replies to keep conversations organized
- **📊 Analytics**: Tracks inquiry processing and maintains search result history
- **🔒 Secure**: Validates Slack signatures and uses proper authentication

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Slack App     │───▶│   Go API        │───▶│   LiteLLM       │
│   (Webhooks)    │    │   Service       │    │   (AI)          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                               │
                               ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Search        │───▶│   Confluence    │
                       │   Engine        │    │   API           │
                       └─────────────────┘    └─────────────────┘
                               │
                               ▼
                       ┌─────────────────┐
                       │   SQLite        │
                       │   Database      │
                       └─────────────────┘
```

## Setup

### Prerequisites

- Go 1.21 or later
- Slack workspace with admin access
- Confluence space access (optional)
- LiteLLM API access at https://litellm.your-company.com

### 1. Environment Configuration

Copy the example configuration file:

```bash
cp config.env.example .env
```

Fill in your configuration values:

```env
# Slack Configuration
SLACK_BOT_TOKEN=xoxb-your-bot-token-here
SLACK_SIGNING_SECRET=your-signing-secret-here
SLACK_APP_TOKEN=xapp-your-app-token-here
SLACK_CHANNEL_ID=C1234567890

# Confluence Configuration (Optional)
CONFLUENCE_BASE_URL=https://your-company.atlassian.net
CONFLUENCE_USERNAME=your-username@company.com
CONFLUENCE_API_TOKEN=your-api-token-here
CONFLUENCE_SPACE_KEY=DOCS

# Server Configuration
PORT=8080
ENV=development

# Database Configuration
DB_PATH=./data/inquiries.db

# AI/Search Configuration
SIMILARITY_THRESHOLD=0.7
MAX_SEARCH_RESULTS=10
SEARCH_DAYS_BACK=90

# LiteLLM Configuration
LITELLM_API_KEY=your-litellm-api-key-here
LITELLM_BASE_URL=https://litellm.your-company.com
LLM_MODEL=gpt-4o-mini
LLM_TEMPERATURE=0.3
LLM_MAX_TOKENS=1000
TRIGGER_EMOJI=eyes
```

### 2. Slack App Setup

1. Create a new Slack app at https://api.slack.com/apps
2. Configure OAuth & Permissions with these scopes:
   - `channels:history` - Read message history
   - `chat:write` - Send messages
   - `reactions:read` - Read emoji reactions
   - `users:read` - Read user information
   - `channels:read` - Read channel information

3. Configure Event Subscriptions:
   - Enable Events: ON
   - Request URL: `https://your-domain.com/api/v1/slack/events`
   - Subscribe to Bot Events:
     - `reaction_added`
     - `reaction_removed`

4. Configure Slash Commands (optional):
   - `/inquiry-help` - Request URL: `https://your-domain.com/api/v1/slack/slash`
   - `/inquiry-status` - Request URL: `https://your-domain.com/api/v1/slack/slash`

5. Install the app to your workspace

### 3. LiteLLM Setup

Ensure you have access to the LiteLLM service at https://litellm.your-company.com and obtain your API key.

### 4. Build and Run

```bash
# Install dependencies
go mod tidy

# Build the application
go build -o found-inq-bot-hack

# Run the application
./found-inq-bot-hack
```

Or run directly:

```bash
go run main.go
```

### 5. Deploy

For production deployment, consider using:

- **Docker**: Create a Dockerfile for containerized deployment
- **Kubernetes**: Deploy as a pod with health checks
- **Cloud Services**: Use services like Google Cloud Run, AWS Lambda, etc.

Example Dockerfile:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o found-inq-bot-hack

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/found-inq-bot-hack .
COPY --from=builder /app/config.env.example .
CMD ["./found-inq-bot-hack"]
```

## Usage

### Basic Workflow

1. **Someone posts a question** in your Slack channel
2. **Any team member reacts** with the 👀 (eyes) emoji
3. **The bot processes** the message:
   - Searches historical Slack messages
   - Searches Confluence documentation
   - Generates an AI response using LiteLLM
4. **Bot responds** in a thread with relevant information

### Slash Commands

- `/inquiry-help` - Shows help information
- `/inquiry-status` - Shows bot status and recent activity

### Configuration Options

| Variable | Description | Default |
|----------|-------------|---------|
| `TRIGGER_EMOJI` | Emoji that triggers the bot | `eyes` |
| `SIMILARITY_THRESHOLD` | Minimum relevance score (0-1) | `0.7` |
| `MAX_SEARCH_RESULTS` | Maximum results to consider | `10` |
| `SEARCH_DAYS_BACK` | Days of history to search | `90` |
| `LLM_TEMPERATURE` | AI creativity level (0-1) | `0.3` |
| `LLM_MAX_TOKENS` | Maximum response length | `1000` |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/v1/slack/events` | POST | Slack Events API webhook |
| `/api/v1/slack/slash` | POST | Slack slash commands |
| `/api/v1/slack/interactive` | POST | Slack interactive components |

## Database Schema

The bot uses SQLite to store:

- **Inquiries**: Original messages and processing status
- **Search Results**: Results from Slack and Confluence searches
- **Reaction Events**: Emoji reaction events for auditing

## Monitoring and Logging

The application uses structured logging with logrus. Log levels:

- **INFO**: Normal operations
- **WARN**: Non-critical issues
- **ERROR**: Processing failures
- **DEBUG**: Detailed debugging (development only)

Example log monitoring with tools like:
- **ELK Stack** for centralized logging
- **Prometheus + Grafana** for metrics
- **Sentry** for error tracking

## Security Considerations

- **Slack Signature Verification**: All requests are validated
- **Environment Variables**: Sensitive data stored in environment variables
- **Database**: Local SQLite database (consider encryption for production)
- **API Keys**: Secure storage and rotation of API keys

## Troubleshooting

### Common Issues

1. **Bot not responding to emoji reactions**
   - Check Slack app permissions
   - Verify webhook URL is accessible
   - Check bot is added to the channel

2. **LiteLLM API errors**
   - Verify API key and base URL
   - Check network connectivity
   - Review API rate limits

3. **Confluence search not working**
   - Verify API token and permissions
   - Check space key configuration
   - Test API connectivity

### Debug Mode

Run with debug logging:

```bash
ENV=development go run main.go
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For questions or issues:
- Create an issue in this repository