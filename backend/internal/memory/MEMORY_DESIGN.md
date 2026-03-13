# Memory System — Design & Data Flow

## Architecture: Two-Tier Memory

```
Every analysis cycle
        │
        ▼
┌───────────────────────┐
│   AppendMemory()      │  Adds new Entry to rolling window
│   (TICKER.json)       │  Max 40 entries (~2 trading days at 15min intervals)
└───────────┬───────────┘
            │
            │  When window is full (>40 entries)
            │  oldest entries are EVICTED
            │
            ▼
┌───────────────────────┐
│  aggregateEvicted()   │  Groups evicted entries by calendar date
│                       │  Builds/merges DailyDigest per date
└───────────┬───────────┘
            │
            ▼
┌───────────────────────┐
│  TICKER_daily.json    │  Grows indefinitely — one digest per trading day
│  (DailyDigest[])      │  Sorted ascending by date
└───────────────────────┘
```

---

## File Layout (per ticker)

```
data/
├── AAPL.json            ← rolling 40 raw entries  (short-term, full detail)
├── AAPL_daily.json      ← all daily digests        (long-term, aggregated)
├── AAPL_weekly.json     ← latest weekly review     (Claude-generated summary)
├── NVDA.json
├── NVDA_daily.json
└── ...
```

---

## Data Structures

### Entry  (one 15-min analysis cycle)
```json
{
  "date": "2026-03-13T14:30:00Z",
  "decision": "HOLD",
  "confidence": "Medium",
  "price": 172.45,
  "rsi": 58.3,
  "macdHistogram": 0.012,
  "trend60d": 4.2,
  "summary": "Stock consolidating near MA50, awaiting catalyst.",
  "reasoning": "...",
  "keyRisk": "Tariff uncertainty weighing on sector",
  "priceTarget": "$180.00",
  "stopLoss": "$165.00",
  "qty": 10,
  "avgCost": 155.00,
  "unrealizedPL": 174.50,
  "unrealizedPLPct": 11.26
}
```

### DailyDigest  (one full trading day, aggregated)
```json
{
  "date": "2026-03-13",
  "session_count": 8,
  "dominant_decision": "HOLD",
  "avg_price": 172.10,
  "price_high": 174.20,
  "price_low": 170.50,
  "avg_rsi": 57.8,
  "avg_macd_hist": 0.009,
  "avg_trend60d": 4.1,
  "decisions": { "BUY": 2, "HOLD": 5, "SELL": 1 },
  "key_risks": ["Tariff uncertainty weighing on sector", "Valuation stretch near highs"],
  "best_summary": "Stock consolidating near MA50, awaiting catalyst."
}
```

---

## What Claude Sees in the Prompt

`BuildMemoryContext(ticker, 8)` produces two sections:

```
── Daily Aggregates (long-term history) ──
[2026-03-10] 7 sessions | Dominant: BUY  | Avg $168.30 (166.20–170.10) | RSI: 53.2 | MACD: +0.005 | 60d: +3.1% | BUY×4 HOLD×3          | Risks: macro uncertainty | Breaking above 20-day MA with rising volume.
[2026-03-11] 8 sessions | Dominant: HOLD | Avg $170.80 (169.50–172.40) | RSI: 56.1 | MACD: +0.008 | 60d: +3.7% | BUY×2 HOLD×5 SELL×1   | Risks: tariff uncertainty | Consolidating below resistance at $172.
[2026-03-12] 6 sessions | Dominant: HOLD | Avg $171.90 (170.00–173.20) | RSI: 57.4 | MACD: +0.011 | 60d: +3.9% | HOLD×5 BUY×1          | Risks: tariff uncertainty | Tight range, waiting for volume confirmation.

── Recent Sessions (detailed) ──
[2026-03-13T13:30:00Z] Decision: BUY (Medium) @ $171.20 | RSI: 55.1 | MACD hist: 0.007 | Position: 10 shares @ $155.00 | P&L: +$162.00 (+10.45%) | Breaking out of tight consolidation with bullish MACD cross.
[2026-03-13T13:45:00Z] Decision: BUY (High)   @ $172.10 | RSI: 57.8 | MACD hist: 0.009 | Position: 10 shares @ $155.00 | P&L: +$171.00 (+11.03%) | Momentum continuing, volume above average.
...
```

---

## Capacity & Retention

| Layer         | File                  | Entries kept | Approx time span     |
|---------------|-----------------------|-------------|----------------------|
| Raw (rolling) | `TICKER.json`         | 40 max      | ~2 trading days      |
| Daily digest  | `TICKER_daily.json`   | unlimited   | forever              |
| Weekly review | `TICKER_weekly.json`  | 1 (latest)  | refreshed every Friday |

### Digest generation timeline
```
Day 1:  40 raw entries fill up → first eviction → Day 1 digest created
Day 2:  more evictions → Day 1 digest may grow via mergeDigests()
Day 3+: new daily digests appended; file grows by 1 entry per trading day
```

At 30 digests shown in context = ~6 weeks of compressed history always available to Claude.
