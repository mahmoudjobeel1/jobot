# Jobot Agent

An autonomous Node.js agent that monitors a list of US stock tickers, fetches live market data, computes technical indicators, and uses Claude AI to generate BUY / SELL / HOLD decisions on a cron schedule.

Built to be extended — Discord and Telegram notifications are already wired up.

---

## Architecture

```
src/
├── index.js        Main entry point + cron scheduler
├── config.js       Ticker list + behaviour settings
├── finnhub.js      Fetches quotes, candles, and news from Finnhub
├── indicators.js   Computes RSI, MACD, MA20/50/200 locally
├── analyst.js      Builds prompt and calls Claude for decisions
├── memory.js       Persists analysis history per ticker as JSON
└── notifier.js     Sends decisions to console / Discord / Telegram

data/
└── AAPL.json       Auto-created memory files, one per ticker
```

---

## Prerequisites

- **Node.js 18+** (uses ES modules and top-level await)
- **Finnhub API key** — free at [finnhub.io](https://finnhub.io) (60 calls/min free)
- **Anthropic API key** — get one at [console.anthropic.com](https://console.anthropic.com)

---

## Setup

### 1. Install dependencies

```bash
npm install
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

Open `src/config.js` and edit the `TICKERS` array:

```js
export const TICKERS = [
  "AAPL",
  "NVDA",
  "MSFT",
  // Add whatever you want here
];
```

### 4. Run

```bash
# Production
npm start

# Development (auto-restarts on file change)
npm run dev
```

---

## Configuration Reference

All runtime config lives in two places:

### `src/config.js` — Static settings
| Setting | Default | Description |
|---|---|---|
| `TICKERS` | 5 stocks | List of tickers to monitor |
| `HISTORY_DAYS` | 120 | Days of candle history to fetch |
| `NEWS_LIMIT` | 8 | Recent news articles per ticker |
| `MEMORY_LIMIT` | 40 | Max stored analysis sessions per ticker |
| `MEMORY_CONTEXT_WINDOW` | 8 | How many past sessions to include in AI prompt |
| `NOTIFY_ON` | All | Which decisions trigger notifications |
| `MIN_CONFIDENCE_TO_NOTIFY` | Medium | Minimum confidence to notify |

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