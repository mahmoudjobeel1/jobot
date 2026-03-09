# Jobot Agent

An autonomous Go agent that monitors a list of US stock tickers, fetches live market data, computes technical indicators, and uses Claude AI to generate BUY / SELL / HOLD decisions on a cron schedule.

Built to be extended — Discord and Telegram notifications are already wired up.

---

## Architecture

```
main.go                     Main entry point + cron scheduler
internal/
├── config/config.go        Ticker list + behaviour settings
├── finnhub/client.go       Fetches quotes, candles, and news from Finnhub
├── indicators/indicators.go Computes RSI, MACD, MA20/50/200 locally
├── analyst/analyst.go      Builds prompt and calls Claude for decisions
├── memory/memory.go        Persists analysis history per ticker as JSON
└── notifier/notifier.go    Sends decisions to console / Discord / Telegram

data/
└── AAPL.json               Auto-created memory files, one per ticker
```

---

## Prerequisites

- **Go 1.22+**
- **Finnhub API key** — free at [finnhub.io](https://finnhub.io) (60 calls/min free)
- **Anthropic API key** — get one at [console.anthropic.com](https://console.anthropic.com)

---

## Setup

### 1. Install dependencies

```bash
go mod download
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` and fill in your API keys:

```env
FINNHUB_API_KEY=ch_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 3. Edit your ticker list

Open `internal/config/config.go` and edit the `Tickers` slice:

```go
var Tickers = []string{
    "AAPL",
    "NVDA",
    "MSFT",
    // Add whatever you want here
}
```

### 4. Run

```bash
# Build and run
go run .

# Or build a binary first
go build -o jobot .
./jobot
```

---

## Configuration Reference

All runtime config lives in two places:

### `internal/config/config.go` — Static settings
| Setting | Default | Description |
|---|---|---|
| `Tickers` | 5 stocks | List of tickers to monitor |
| `HistoryDays` | 120 | Days of candle history to fetch |
| `NewsLimit` | 8 | Recent news articles per ticker |
| `MemoryLimit` | 40 | Max stored analysis sessions per ticker |
| `MemoryContextWindow` | 8 | How many past sessions to include in AI prompt |
| `NotifyOn` | All | Which decisions trigger notifications |
| `MinConfidenceToNotify` | Medium | Minimum confidence to notify |

### `.env` — Runtime settings
| Variable | Default | Description |
|---|---|---|
| `CRON_SCHEDULE` | `*/15 9-16 * * 1-5` | Cron expression (ET timezone) |
| `RUN_ON_START` | `true` | Run one cycle immediately on startup |
| `MARKET_HOURS_ONLY` | `true` | Skip cycles when US market is closed |

---

## Adding Discord Notifications

1. Create a Discord webhook in your server: Channel Settings → Integrations → Webhooks
2. Copy the webhook URL
3. Add to `.env`:

```env
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_ID/YOUR_TOKEN
```

That's it — the agent will start sending formatted messages automatically.

---

## Adding Telegram Notifications

1. Create a bot via [@BotFather](https://t.me/BotFather) — get your `BOT_TOKEN`
2. Start a chat with your bot, then get your `CHAT_ID`:
   ```
   https://api.telegram.org/bot<BOT_TOKEN>/getUpdates
   ```
3. Add to `.env`:

```env
TELEGRAM_BOT_TOKEN=123456789:ABCdefGhIjKlMnOpQrStuVwXyZ
TELEGRAM_CHAT_ID=-1001234567890
```

---

## How Memory Works

Every analysis result is saved to `data/<TICKER>.json`. On the next run, the last 8 sessions are included in the Claude prompt — so the AI remembers what it previously decided, at what price, and why. This lets it:

- Detect trend reversals more accurately
- Avoid repeating the same decision if nothing has changed
- Build conviction over multiple sessions before flipping direction

Memory files grow automatically and are trimmed to `MEMORY_LIMIT` entries.