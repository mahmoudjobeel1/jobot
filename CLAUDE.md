# CLAUDE.md — Jobot

## Project Overview

AI-powered stock market decision agent written in Go. Monitors a portfolio of US equity tickers on a cron schedule, fetches live market data, computes technical indicators locally, and calls Claude to generate BUY / SELL / HOLD decisions. Optionally sends alerts to Discord and Telegram.

## Repository Layout

```
backend/                   Go application root
├── main.go                Entry point: env validation, cron setup, runCycle
├── go.mod                 Module: jobot, Go 1.26
├── internal/
│   ├── config/config.go   Constants + tickers (derived from portfolio)
│   ├── portfolio/         Holdings, P&L math, BuildPortfolioContext for prompt
│   ├── finnhub/client.go  Finnhub (quote, news) + Yahoo Finance (candles)
│   ├── indicators/        RSI, EMA, MACD, SMA, AvgVol — all pure functions
│   ├── memory/            JSON persistence: data/<TICKER>.json
│   ├── analyst/           Builds Claude prompt, calls Anthropic SDK, parses response
│   └── notifier/          Console formatter + Discord/Telegram webhook senders
└── data/                  Auto-created at runtime; one JSON file per ticker
```

## Commands

```bash
cd backend

# Run (loads .env automatically via godotenv)
go run .

# Build binary
go build -o jobot .

# Vet + build check
go vet ./...
go build ./...

# Add/update dependencies
go get <package>
go mod tidy
```

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.13` | Claude API |
| `github.com/joho/godotenv v1.5.1` | `.env` file loading |
| `github.com/robfig/cron/v3 v3.0.1` | Cron scheduling with timezone support |

## Environment Variables

Loaded from `backend/.env` (copy from `.env.example`):

| Variable | Required | Default | Description |
|---|---|---|---|
| `FINNHUB_API_KEY` | Yes | — | Finnhub free-tier key |
| `ANTHROPIC_API_KEY` | Yes | — | Anthropic API key |
| `CRON_SCHEDULE` | No | `*/15 9-16 * * 1-5` | Cron expression (ET timezone) |
| `RUN_ON_START` | No | `true` | Run one cycle immediately on startup |
| `MARKET_HOURS_ONLY` | No | `true` | Skip cycles when NYSE is closed |
| `DISCORD_WEBHOOK_URL` | No | — | Discord channel webhook |
| `TELEGRAM_BOT_TOKEN` | No | — | Telegram bot token |
| `TELEGRAM_CHAT_ID` | No | — | Telegram chat ID |

## Key Architectural Decisions

**Portfolio-driven tickers** — `config.Tickers` is derived from `portfolio.Holdings`, not a hardcoded list. To add/remove a ticker, edit `internal/portfolio/portfolio.go` (`Holdings` slice). Set `Qty: 0` and `AvgCost: 0` for watchlist-only positions.

**Nullable indicators** — RSI, MACD components, MAs, trend, and volume values are `*float64` / `*int64` to mirror the JS `null` semantics. Always nil-check before dereferencing.

**Memory** — Each analysis is appended to `data/<TICKER>.json` (capped at `MemoryLimit = 40`). The last `MemoryContextWindow = 8` entries are injected into the Claude prompt so the model has historical context.

**Claude model** — Currently uses `claude-sonnet-4-20250514` via the Anthropic Go SDK. The prompt is built in `internal/analyst/analyst.go:buildPrompt`.

**Cron timezone** — Scheduler runs in `America/New_York`. The `isMarketOpen` check in `main.go` also uses ET.

**Rate limiting** — 1200ms sleep between tickers; 500ms between Finnhub calls within a ticker fetch.

## Modifying the Portfolio

Edit `internal/portfolio/portfolio.go`:

```go
var Holdings = []Holding{
    {Ticker: "AAPL", Qty: 10, AvgCost: 182.50},  // owned position
    {Ticker: "NVDA", Qty: 0,  AvgCost: 0},        // watchlist only
}
```

`config.Tickers` and the prompt's portfolio context section update automatically.

## Notification Thresholds

Controlled by constants in `internal/config/config.go`:

- `NotifyOn` — which decisions send alerts (default: all three)
- `MinConfidenceToNotify` — minimum confidence level (`"Low"`, `"Medium"`, `"High"`)

Notifications are skipped silently when env vars for Discord/Telegram are absent.
