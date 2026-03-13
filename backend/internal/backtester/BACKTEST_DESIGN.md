# Backtester — Design & Data Flow

## Overview

The backtester has two independent evaluation modes that run together in one pass:

```
Yahoo Finance (24 months of daily OHLCV)
        │
        ▼
┌───────────────────────────────────────┐
│  Sliding Window Loop (bar by bar)     │
│                                       │
│  for i = minBars → lastSignalBar:     │
│    ComputeAll(closes[:i])             │
│    → signal(RSI, MACD hist crossover) │
│    → simulate trade (enter/exit)      │
│    → update equity curve              │
└──────────────┬────────────────────────┘
               │                   │
               ▼                   ▼
    ┌─────────────────┐   ┌──────────────────────┐
    │ Strategy Metrics│   │  Memory Evaluation   │
    │                 │   │                      │
    │  win rate       │   │  Load data/TICKER.json│
    │  total return   │   │  match entry date →  │
    │  vs buy&hold    │   │  candle bar          │
    │  max drawdown   │   │  check price+5d      │
    │  Sharpe ratio   │   │  BUY/SELL/HOLD score │
    └─────────────────┘   └──────────────────────┘
```

---

## Mode 1 — Indicator Strategy Simulation

Replays what the bot's *technical signals alone* would have done, without Claude.

### Signal Rules

```
Given: RSI(14), MACD histogram, prev bar's MACD histogram

macdCrossedUp   = prevHist ≤ 0  AND  currentHist > 0
macdCrossedDown = prevHist ≥ 0  AND  currentHist < 0

BUY  if:  RSI < 35                          ← strongly oversold
      or   RSI < 50  AND  macdCrossedUp     ← recovering + momentum turning positive

SELL if:  RSI > 75                          ← strongly overbought
      or   RSI > 65  AND  macdCrossedDown   ← overextended + momentum turning negative

HOLD:  everything else
```

### Trade Lifecycle

```
State: FLAT
│
├── BUY signal → enter at closes[i], record entryPrice & entryCapital
│   └── State: LONG
│       │
│       ├── SELL signal → exit, realise P&L   (exitReason: "signal")
│       ├── holdDays ≥ maxHoldDays → exit     (exitReason: "timeout")
│       └── end of data → exit                (exitReason: "end-of-data")
│           └── State: FLAT → repeat
```

### Equity Curve

```
Bar:     0      1      2      3      4      5      6      7      8
Price: 100    102    101    103    107    106    104    108    110
Sig:         HOLD   HOLD   BUY          HOLD   HOLD   SELL

Capital: 10000  10000  10000  10000  10412  10315  10122  10511  [flat after exit]
                              ↑ enter                    ↑ exit
                           @ $103                      @ $108
                           return = +4.85%
```

Mark-to-market formula while in trade:
```
equity[i] = entryCapital × (closes[i] / entryPrice)
```

### Metrics Computed

| Metric | Formula |
|---|---|
| Total return | `(finalCapital − initialCapital) / initialCapital × 100` |
| Buy-and-hold | `(closes[last] − closes[0]) / closes[0] × 100` |
| Alpha | `totalReturn − buyHoldReturn` |
| Win rate | `winningTrades / totalTrades × 100` |
| Avg win | mean of all positive trade returns |
| Avg loss | mean of all negative trade returns |
| Max drawdown | max `(peak − trough) / peak × 100` over equity curve |
| Sharpe ratio | `(mean(dailyReturns) − rfrDaily) / std(dailyReturns) × √252` |

Risk-free rate used: **5% annual** → `0.05 / 252 ≈ 0.0198%` per day

---

## Mode 2 — Claude Decision Accuracy (Memory Evaluation)

Checks whether the bot's real past decisions (stored in `data/<TICKER>.json`) were correct
by looking at what the price actually did 5 trading days later.

### Matching Logic

```
Memory entry date:  "2026-03-10T14:30:00Z"
                         ↓ truncate to YYYY-MM-DD
                     "2026-03-10"
                         ↓ lookup in candle timestamp map
                     closes[bar] = $172.45

Forward check:       closes[bar + 5] = $175.80

Return 5d:           (175.80 − 172.45) / 172.45 × 100 = +1.94%
```

### Correctness Criteria

```
Decision   Correct if
─────────────────────────────────────────────────────
BUY        closes[bar+5]  >  closes[bar]    (price went up)
SELL       closes[bar+5]  <  closes[bar]    (price went down)
HOLD       |return5d|    ≤  2.0%            (price stayed flat)
```

### Output Example

```
CLAUDE DECISION ACCURACY  (34 decisions evaluated)
─────────────────────────────────────────────────────
Overall accuracy:        64.7%   (22/34 correct)
Avg return 5d after BUY: +1.83%

BUY    18 decisions  →  72.2% correct
SELL    8 decisions  →  62.5% correct
HOLD    8 decisions  →  37.5% correct   ← HOLD is hardest to be right on (narrow ±2% band)
```

---

## Full Timeline Example (AAPL, 2 years)

```
2024-03-13 ──────────────────────────────────────── 2026-03-13
│                                                          │
│  504 trading bars (daily)                               │
│                                                          │
│  minBarsNeeded = 35  (for MACD to stabilise)           │
│  lastSignalBar = n − maxHoldDays − 2                   │
│                                                          │
│  Bars 0–34:   warm-up, no signals generated             │
│  Bars 35–492: signal generation + trade simulation      │
│  Bars 493–503: no new entries (reserved for exits)     │
│                                                          │
│  Memory entries: whatever Claude stored in data/AAPL.json
│  (evaluated independently, not part of equity sim)      │
```

---

## Sliding Window — Why It Matters

Each bar uses only data available **up to that point** — no look-ahead bias:

```
Bar 50:   ComputeAll( closes[0:51] )   ← only 51 bars of history
Bar 51:   ComputeAll( closes[0:52] )
Bar 52:   ComputeAll( closes[0:53] )
...
Bar 200:  ComputeAll( closes[0:201] )  ← MA200 finally available here
```

Indicators that need more data than available return `nil` and are treated as HOLD.

---

## Running the Backtester

```bash
cd backend

# Single ticker — 24 months, $10k capital, max 10-day hold
go run ./cmd/backtest -ticker AAPL

# All portfolio tickers
go run ./cmd/backtest -all

# Custom parameters
go run ./cmd/backtest -ticker NVDA -months 12 -hold 5 -capital 50000
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-ticker` | — | Single ticker to test |
| `-all` | false | Test every ticker in the portfolio |
| `-months` | 24 | Months of history to fetch from Yahoo Finance |
| `-hold` | 10 | Max days to hold before forced exit |
| `-capital` | 10000 | Starting capital in USD |

---

## Limitations

- **No transaction costs** — brokerage fees and slippage are not modelled
- **Daily bars only** — the live bot runs every 15 min; signals here fire once per day at close
- **No Claude** — signal rules are a deterministic approximation; real Claude decisions incorporate news and memory context that can't be replayed cheaply
- **Memory accuracy is forward-looking only 5 days** — a longer horizon (10d, 20d) would give different accuracy numbers
- **HOLD accuracy is structurally lower** — the ±2% band is narrow; most stocks move more than 2% in 5 days
