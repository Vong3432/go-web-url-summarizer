# go-web-url-summarizer

A Go HTTP API that scrapes web pages and returns AI-generated summaries using OpenAI GPT-4o Mini.

## How it works

1. Client sends a list of URLs via `POST /summarize`
2. The server scrapes each URL concurrently, extracting clean body text (up to 5,000 chars)
3. Each page's text is sent to OpenAI GPT-4o Mini, which returns a ~500-character summary
4. Results are returned in a single JSON response — failed URLs include an error instead of a summary

## Requirements

- Go 1.25+
- OpenAI API key
- Docker (optional)

## Getting started

**1. Clone and install dependencies**

```bash
git clone https://github.com/Vong3432/go-web-url-summarizer.git
cd go-web-url-summarizer
go mod download
```

**2. Configure environment**

```bash
cp .env.example .env
```

Edit `.env`:

```env
PORT=8080
MAX_ALLOWED_URLS=10
```

**3. Run**

```bash
go run ./cmd
```

## Running with Docker

```bash
docker compose up --build
```

The app will be available at `http://localhost:8080`.

## API

### POST /summarize

Scrapes each URL and returns an AI-generated summary per URL.

Each caller provides their own OpenAI API key in the request body — the server holds no credentials.

**Request**

```json
{
  "openai_api_key": "sk-...",
  "urls": [
    "https://example.com",
    "https://another-site.com"
  ]
}
```

**Response `200 OK`**

```json
{
  "summaries": [
    {
      "url": "https://example.com",
      "summary": "Example Domain is a placeholder website maintained by IANA...",
      "error": null
    },
    {
      "url": "https://another-site.com",
      "summary": null,
      "error": "fetch https://another-site.com: connection refused"
    }
  ]
}
```

**Error responses**

| Status | Reason |
|--------|--------|
| `400` | Missing or empty `urls` array |
| `400` | Missing `openai_api_key` |
| `400` | `urls` exceeds the configured maximum |
| `429` | Rate limit exceeded (1 request per 10 seconds per IP) |

## Rate limiting

Requests are limited to **1 per 10 seconds per IP address**. Exceeding this returns `429 Too Many Requests`.

## Project structure

```
.
├── cmd/
│   ├── main.go          # Entry point, app bootstrap
│   └── api.go           # Router, middleware, server
├── internal/
│   ├── handler/         # HTTP handler for /summarize
│   ├── scraper/         # URL fetching and text extraction
│   └── summarizer/      # OpenAI API client
├── .env.example         # Environment variable template
├── Dockerfile
└── compose.yml
```
