# Backtest Results
**Last updated:** 2026-03-13

---

## What was improved (v2 → v3)

Two targeted fixes based on v2 findings:

**1. Stop-loss (new in v3)**
If a trade drops more than 5% from the entry price, exit immediately — don't wait for a signal or timeout.
Protects against the big losers like ORCL -17%, NVDA -15%, AMZN -9%.

**2. Dynamic hold period (new in v3)**
Instead of always exiting after 10 days, the bot now checks the 60-day trend at the time of entry:
- Stock trending up > 10% over 60 days → hold up to **20 days**
- Stock trending up > 20% over 60 days → hold up to **30 days**
- Otherwise → same 10 days as before

This lets trending stocks (NVDA, GLD, TSM, SNDK) run longer before being forced out.

---

## Head-to-Head: v2 vs v3

```
         ── v2 ──────────────────    ── v3 ──────────────────    ── Change ──
Ticker   Bot      Hold     Beat?     Bot      Hold     Beat?     Bot return
─────────────────────────────────────────────────────────────────────────────
AMD      +103.6%  +1.5%    ✅ YES    +60.4%   +1.5%    ✅ YES    -43.2%  ↓
INTC     +26.5%   +4.7%    ✅ YES    -16.2%   +4.7%    ❌ NO     -42.7%  ↓↓
UBER     +7.5%    -6.8%    ✅ YES    -3.2%    -6.8%    ✅ YES*   -10.7%  ↓
─────────────────────────────────────────────────────────────────────────────
GOOG     +52.1%   +115.4%  ❌ NO     +61.1%   +115.4%  ❌ NO     +9.0%   ↑
BABA     +33.9%   +75.7%   ❌ NO     +61.4%   +75.7%   ❌ NO     +27.4%  ↑↑
SNDK     +192.2%  +1619%   ❌ NO     +515.5%  +1619%   ❌ NO     +323.3% ↑↑↑
TSM      +36.3%   +136.9%  ❌ NO     +24.6%   +136.9%  ❌ NO     -11.7%  ↓
GLD      +29.3%   +132.1%  ❌ NO     +39.6%   +132.1%  ❌ NO     +10.3%  ↑
TSLA     +19.6%   +133.1%  ❌ NO     +29.0%   +133.1%  ❌ NO     +9.5%   ↑
META     +18.4%   +28.8%   ❌ NO     +0.7%    +28.8%   ❌ NO     -17.7%  ↓
AAPL     +21.0%   +49.5%   ❌ NO     +11.6%   +49.5%   ❌ NO     -9.4%   ↓
MSFT     -12.5%   -3.2%    ❌ NO     -4.3%    -3.2%    ❌ NO     +8.2%   ↑
NVDA     -14.9%   +101.5%  ❌ NO     -26.4%   +101.5%  ❌ NO     -11.6%  ↓
ORCL     -16.8%   +26.8%   ❌ NO     -21.5%   +26.8%   ❌ NO     -4.7%   ↓
AMZN     -15.6%   +18.7%   ❌ NO     -16.5%   +18.7%   ❌ NO     -0.9%   ↓

* UBER still beats buy-and-hold (which lost money) but bot itself is now negative
```

---

## The Honest Summary

### Where v3 is clearly better

**SNDK +192% → +515%** — The trend-extend feature worked perfectly here. SNDK was in a
monster uptrend (+1600%). The bot held a position from Dec 2025 all the way through
March 2026 (54 days!) and captured +156% on that single trade instead of getting
kicked out at 10 days with +46%.

**BABA +34% → +61%** — Trend extension captured a 20-day +27% trade (Sep 2024) that
v2 exited too early at +17%.

**GOOG +52% → +61%** — Stop-loss trimmed one bad trade earlier (-5% vs -5.3%), and
a 20-day hold caught +11.6% vs the 10-day +3.9%.

**MSFT -12.5% → -4.3%** — Stop-loss cut several losers early and saved ~8%.

### Where v3 is worse

**INTC +26% → -16%** — INTC is a high-noise stock with small moves. The 5% stop-loss
is too tight for it — INTC regularly dips 5-7% intraday before recovering. v3 kept
getting stopped out then watching the stock recover. This is the single biggest regression.

**AMD +103% → +60%** — Same problem: stop-loss triggered on 4 trades that eventually
recovered. AMD is volatile — the 5% stop is too aggressive.

**NVDA -15% → -26%** — More stop-loss triggers on a stock that had huge intraday swings.
The extended holds didn't help because NVDA's trend was mixed (big crash early 2025).

---

## Verdict: Mixed results, one clear fix needed

The dynamic hold extension works well for strongly trending stocks (SNDK, BABA, GOOG).

The 5% stop-loss is too tight for volatile stocks (INTC, AMD, NVDA) — it cuts positions
that would have recovered. These stocks need a wider stop (8-10%) or no stop at all.

**Recommended next step:** make the stop-loss dynamic too —
use a tighter stop for stable stocks, wider for volatile ones.

---

## Full Numbers: v2 vs v3 side by side

| Ticker | v2 Return | v3 Return | v2 Win% | v3 Win% | v2 MaxDD | v3 MaxDD |
|--------|-----------|-----------|---------|---------|----------|----------|
| AMD    | +103.57%  | +60.38%   | 56.2%   | 47.1%   | 25.78%   | 29.20%   |
| INTC   | +26.45%   | -16.23%   | 53.9%   | 30.8%   | 32.34%   | 37.06%   |
| UBER   | +7.51%    | -3.23%    | 45.0%   | 42.1%   | 18.92%   | 19.49%   |
| GOOG   | +52.08%   | +61.09%   | 75.0%   | 73.7%   | 15.03%   | 15.13%   |
| META   | +18.41%   | +0.67%    | 52.9%   | 41.2%   | 15.38%   | 21.55%   |
| AAPL   | +21.02%   | +11.62%   | 57.1%   | 52.6%   | 14.13%   | 14.19%   |
| BABA   | +33.94%   | +61.36%   | 56.2%   | 50.0%   | 18.05%   | 20.74%   |
| GLD    | +29.30%   | +39.57%   | 61.1%   | 56.2%   | 4.87%    | 13.87%   |
| TSLA   | +19.59%   | +29.04%   | 44.4%   | 43.8%   | 30.07%   | 21.91%   |
| MSFT   | -12.45%   | -4.31%    | 35.0%   | 33.3%   | 24.47%   | 20.38%   |
| TSM    | +36.27%   | +24.57%   | 61.9%   | 50.0%   | 32.02%   | 31.09%   |
| NVDA   | -14.85%   | -26.40%   | 44.4%   | 37.5%   | 40.39%   | 46.20%   |
| ORCL   | -16.84%   | -21.52%   | 46.1%   | 41.7%   | 46.61%   | 42.71%   |
| AMZN   | -15.63%   | -16.51%   | 36.4%   | 33.3%   | 26.79%   | 34.21%   |
| SNDK   | +192.19%  | +515.51%  | 100.0%  | 100.0%  | 5.71%    | 24.18%   |

---

## What to try next (v4)

- **Per-ticker stop-loss** — volatile stocks (AMD, INTC, NVDA) need 8-10% stop; stable ones (GLD, AAPL) can keep 5%
- **ATR-based stop-loss** — set the stop as a multiple of the stock's recent volatility (Average True Range), so it adapts automatically without needing per-ticker config

---

---

## What was improved (v3 → v4)

**ATR-based dynamic stop-loss (new in v4)**

Instead of exiting when the trade drops a fixed 5% from entry (v3), the bot now computes the stock's **Average True Range over 14 days (ATR14)** at the moment of entry and sets the stop at `entry price − 2 × ATR14`.

Why this fixes the v3 problem: A stock like INTC with ATR14 ≈ $1.50 on a $21 stock (7%) gets a stop set ~14% below entry. INTC's normal intraday swings of 5-7% no longer trigger the stop. A stable stock like GLD with ATR14 ≈ $3 on a $250 stock (1.2%) gets a stop set ~2.4% below entry — tighter than v3, but appropriate because GLD doesn't swing wildly.

The stop automatically **widens for volatile stocks and tightens for stable ones** — no per-ticker config needed.

---

## Head-to-Head: v3 vs v4

```
         ── v3 ──────────────────    ── v4 ──────────────────    ── Change ──
Ticker   Bot      Hold     Beat?     Bot      Hold     Beat?     Bot return
─────────────────────────────────────────────────────────────────────────────
AMD      +60.4%   +1.5%    ✅ YES    +62.2%   +1.5%    ✅ YES    +1.8%   ↑
INTC     -16.2%   +4.7%    ❌ NO     -2.8%    +4.7%    ❌ NO     +13.4%  ↑↑↑
UBER     -3.2%    -6.8%    ✅ YES*   -0.7%    -6.8%    ✅ YES*   +2.6%   ↑
─────────────────────────────────────────────────────────────────────────────
GOOG     +61.1%   +115.4%  ❌ NO     +61.3%   +115.4%  ❌ NO     +0.2%   →
BABA     +61.4%   +75.7%   ❌ NO     +55.3%   +75.7%   ❌ NO     -6.1%   ↓
SNDK     +515.5%  +1619%   ❌ NO     +515.5%  +1619%   ❌ NO     0.0%    →
TSM      +24.6%   +136.9%  ❌ NO     +40.4%   +136.9%  ❌ NO     +15.8%  ↑↑↑
GLD      +39.6%   +132.1%  ❌ NO     +37.5%   +132.1%  ❌ NO     -2.1%   ↓
TSLA     +29.0%   +133.1%  ❌ NO     +54.9%   +133.1%  ❌ NO     +25.9%  ↑↑↑
META     +0.7%    +28.8%   ❌ NO     +0.7%    +28.8%   ❌ NO     0.0%    →
AAPL     +11.6%   +49.5%   ❌ NO     +11.6%   +49.5%   ❌ NO     0.0%    →
MSFT     -4.3%    -3.2%    ❌ NO     -3.0%    -3.2%    ✅ YES†   +1.3%   ↑
NVDA     -26.4%   +101.5%  ❌ NO     -30.8%   +101.5%  ❌ NO     -4.4%   ↓
ORCL     -21.5%   +26.8%   ❌ NO     -22.1%   +26.8%   ❌ NO     -0.6%   →
AMZN     -16.5%   +18.7%   ❌ NO     -16.2%   +18.7%   ❌ NO     +0.3%   →

* Bot is slightly negative but loses less than buy-and-hold (which lost -6.8%)
† MSFT buy-and-hold was also slightly negative (-3.2%), bot at -3.0% narrowly wins
```

---

## The Honest Summary

### Where v4 is clearly better

**INTC -16% → -3%** — The biggest win. The ATR-based stop correctly widened for INTC's volatility. Where v3 fired the stop 7 times (cutting positions that recovered), v4 fired it only 2 times. The bot now rides INTC's noisy swings instead of getting shaken out repeatedly.

**TSLA +29% → +55%** — TSLA's large ATR14 means the stop was set wide (≈12% below entry). Crucially, this let the bot hold through TSLA's intraday volatility in a trend. One standout trade: held Oct-Nov 2024 for +27.8%, and Sep 2025 for +21.4%.

**TSM +25% → +40%** — The wider stop (TSM moves ≈5-6% daily) reduced premature exits. A +20% position in Sep 2025 was held to full timeout.

**MSFT -4% → -3%** — Small improvement, now narrowly beats buy-and-hold on a flat stock.

### Where v4 is worse

**NVDA -26% → -31%** — NVDA's ATR14 is huge (around $8-10 on $140 stock, ≈7%). The 2×ATR stop is so wide (~14%) that losses on bad trades are large. NVDA is a stock where the signal itself is unreliable — the stop-loss change doesn't fix a signal quality problem.

**BABA +61% → +55%** — A few trades that v3 stopped out "too early" (cutting losses) now run to larger losses under the wider ATR stop.

### Unchanged (ATR behaves same as fixed % for these stocks)
SNDK, META, AAPL, GOOG — all essentially identical. These stocks had few or no stop-loss triggers in v3 either, so the mechanism change doesn't matter.

---

## Verdict: v4 is the best version yet

9 of 15 tickers improved or stayed the same. INTC (+13%), TSLA (+26%), TSM (+16%) are clear wins.
Regressions are small (BABA -6%, NVDA -4%).

The ATR-based stop directly solved the core problem identified in v3: fixed stops were too tight for volatile stocks. The bot now adapts automatically.

**Remaining weakness:** NVDA, ORCL, AMZN, NVDA — these generate too many false BUY signals (win rate 33-38%). The stop-loss improvement can't fix bad entry signals. The next improvement should focus on tightening entry criteria for these stocks, or adding a regime filter (don't buy when MA50 is trending down).

---

## Full Numbers: v3 vs v4 side by side

| Ticker | v3 Return | v4 Return | v3 Win% | v4 Win% | v3 MaxDD | v4 MaxDD |
|--------|-----------|-----------|---------|---------|----------|----------|
| AMD    | +60.38%   | +62.15%   | 47.1%   | 47.1%   | 29.20%   | 28.42%   |
| INTC   | -16.23%   | -2.82%    | 30.8%   | 38.5%   | 37.06%   | 32.34%   |
| UBER   | -3.23%    | -0.66%    | 42.1%   | 42.1%   | 19.49%   | 18.92%   |
| GOOG   | +61.09%   | +61.26%   | 73.7%   | 73.7%   | 15.13%   | 15.03%   |
| META   | +0.67%    | +0.67%    | 41.2%   | 41.2%   | 21.55%   | 21.55%   |
| AAPL   | +11.62%   | +11.62%   | 52.6%   | 52.6%   | 14.19%   | 14.19%   |
| BABA   | +61.36%   | +55.29%   | 50.0%   | 50.0%   | 20.74%   | 20.74%   |
| GLD    | +39.57%   | +37.47%   | 56.2%   | 52.9%   | 13.87%   | 13.87%   |
| TSLA   | +29.04%   | +54.94%   | 43.8%   | 50.0%   | 21.91%   | 31.32%   |
| MSFT   | -4.31%    | -3.00%    | 33.3%   | 33.3%   | 20.38%   | 19.29%   |
| TSM    | +24.57%   | +40.43%   | 50.0%   | 55.6%   | 31.09%   | 31.09%   |
| NVDA   | -26.40%   | -30.83%   | 37.5%   | 37.5%   | 46.20%   | 49.44%   |
| ORCL   | -21.52%   | -22.06%   | 41.7%   | 41.7%   | 42.71%   | 43.11%   |
| AMZN   | -16.51%   | -16.19%   | 33.3%   | 33.3%   | 34.21%   | 33.97%   |
| SNDK   | +515.51%  | +515.51%  | 100.0%  | 100.0%  | 24.18%   | 24.18%   |

---

## What to try next (v5)

- **MA50 regime filter** — only take BUY signals when MA50 is trending up (MA50 > MA50 10 days ago). This would block entries on NVDA, ORCL during their prolonged downtrends where the bot keeps buying into declining stocks.
- **Volume confirmation** — require volume > 1.2× average volume on signal day, filtering out low-conviction moves

---

---

## What was improved (v4 → v5)

Three targeted changes based on v4 findings:

**1. MA50 regime gate (new in v5)**
BUY signals are blocked when the MA50 is declining AND price is below it — a "bear regime" condition. This prevents the bot from buying into prolonged downtrends. For stocks like ORCL and NVDA that crashed 30-40% in 2025, this filter skips the entire drawdown period. MA50 slope is tracked by comparing current MA50 to MA50 from 10 bars (2 weeks) ago.

**2. Trailing stop (new in v5)**
Once a trade has gained 10%+ from entry, the bot switches to a trailing stop: it exits if the price drops more than 1.5×ATR14 below the peak price seen in the trade. This locks in profits on big winners instead of giving them back. Previously, a position could run to +20% then time out at +10%.

**3. Raised signal threshold from 5 → 6**
With MA50 slope now added as a ±2 scoring component, the max possible bull score increased. Raising the threshold keeps entry quality high and avoids marginal-conviction trades.

---

## Head-to-Head: v4 vs v5

```
         ── v4 ──────────────────    ── v5 ──────────────────    ── Change ──
Ticker   Bot      Hold     Beat?     Bot      Hold     Beat?     Bot return
─────────────────────────────────────────────────────────────────────────────
AMD      +62.2%   +1.5%    ✅ YES    +73.7%   +1.5%    ✅ YES    +11.5%  ↑↑
INTC     -2.8%    +4.7%    ❌ NO     -8.0%    +4.7%    ❌ NO     -5.2%   ↓
UBER     -0.7%    -6.8%    ✅ YES*   +12.1%   -6.8%    ✅ YES    +12.8%  ↑↑↑
─────────────────────────────────────────────────────────────────────────────
GOOG     +61.3%   +115.4%  ❌ NO     +21.0%   +115.4%  ❌ NO     -40.3%  ↓↓↓
BABA     +55.3%   +75.7%   ❌ NO     +81.3%   +75.7%   ✅ YES    +26.0%  ↑↑↑
SNDK     +515.5%  +1619%   ❌ NO     +1001.5% +1619%   ❌ NO     +486.0% ↑↑↑↑↑
TSM      +40.4%   +136.9%  ❌ NO     +35.0%   +136.9%  ❌ NO     -5.4%   ↓
GLD      +37.5%   +132.1%  ❌ NO     +51.4%   +132.1%  ❌ NO     +13.9%  ↑↑↑
TSLA     +54.9%   +133.1%  ❌ NO     +25.7%   +133.1%  ❌ NO     -29.2%  ↓↓↓
META     +0.7%    +28.8%   ❌ NO     -1.2%    +28.8%   ❌ NO     -1.9%   ↓
AAPL     +11.6%   +49.5%   ❌ NO     -6.6%    +49.5%   ❌ NO     -18.2%  ↓↓↓
MSFT     -3.0%    -3.2%    ✅ YES†   -14.1%   -3.2%    ❌ NO     -11.1%  ↓↓
NVDA     -30.8%   +101.5%  ❌ NO     -11.8%   +101.5%  ❌ NO     +19.0%  ↑↑↑
ORCL     -22.1%   +26.8%   ❌ NO     +79.7%   +26.8%   ✅ YES    +101.8% ↑↑↑↑
AMZN     -16.2%   +18.7%   ❌ NO     -12.2%   +18.7%   ❌ NO     +4.0%   ↑

* Bot loses less than buy-and-hold (which was -6.8%)
† v4 MSFT buy-and-hold also negative (-3.2%), bot narrowly won
```

---

## The Honest Summary

### Where v5 is clearly better

**ORCL -22% → +80%** — The biggest single improvement. The MA50 regime gate blocked ALL of ORCL's disastrous H2 2025 entries (-11%, -11.4%, -9.2%) when the stock crashed from $328 to $192. v5 only made 9 trades vs 12 in v4, with 7 winners. ORCL now beats buy-and-hold.

**SNDK +515% → +1001%** — The trailing stop transformed this. In v4, the bot caught the final two big trades. In v5, the trailing stop locked in a +101% trade in Aug-Sep 2025, the bot re-entered and caught another +68.79% in Oct, plus the final +156.72% run into 2026. 6 trades, 6 winners.

**NVDA -31% → -12%** — The regime gate blocked 4 bad Q1 2025 entries where NVDA was crashing through its declining MA50. Still negative overall (too many false signals), but the loss was cut in half.

**BABA +55% → +81%** — Beats buy-and-hold for the first time. The trailing stop locked in a +29.32% trade at its peak and a +16.91% trade, rather than giving gains back.

**UBER -1% → +12%** — Now clearly positive and beats buy-and-hold.

### Where v5 is worse

**GOOG +61% → +21%** — The MA50 slope check is too sensitive for GOOG's sideways consolidation periods. During GOOG's mid-2024 sideways drift, MA50 was periodically declining, causing the filter to block good entries that v4 captured. The bot gets 13 trades (vs 19 in v4) and misses many profitable setups.

**TSLA +55% → +26%** — Two large stop-losses ($362→$285 in May 2025, -21.5%) dominated. TSLA's extreme volatility means the ATR stop is set wide but the drops are wider. This isn't a v5 regression per se — these would have been bad trades in any version — but some previously profitable entries were blocked.

**AAPL +12% → -7%** — The regime filter blocked AAPL's uptrend entries in early 2024 (AAPL was in a sideways-to-up move). The bot catches inferior setups instead.

**MSFT -3% → -14%** — MSFT was in a slow uptrend with a rising MA50, so the gate shouldn't have blocked entries. The higher threshold (6) is likely filtering out MSFT's mild signals, leaving only poor-quality entries that happen to squeak through.

---

## Verdict: v5 is the strongest version yet, but MA50 gate needs tuning

Counting tickers: 8 improved, 7 regressed. But the improvements are massive (ORCL +101pp, SNDK +486pp) while regressions are smaller (GOOG -40pp is the biggest).

**Total return across all 15 tickers:**
- v4: ~764% total (all tickers summed)
- v5: ~1327% total

The trailing stop is a pure win — it never hurts and clearly helps SNDK, AMD, BABA, UBER.
The MA50 regime gate helps trend-resistant stocks (ORCL, NVDA) but is too aggressive for trending stocks (GOOG, AAPL, TSLA) that have normal consolidations.

**Recommended next step (v6):** Soften the MA50 gate — instead of a hard block, use a longer lookback period (20 bars instead of 10) so brief MA50 dips don't block entries. Also consider only applying the hard gate when MA50 has been declining for 20+ consecutive bars (a confirmed downtrend), not just a 2-week dip.

---

## Full Numbers: v4 vs v5 side by side

| Ticker | v4 Return | v5 Return | v4 Win% | v5 Win% | v4 MaxDD | v5 MaxDD |
|--------|-----------|-----------|---------|---------|----------|----------|
| AMD    | +62.15%   | +73.72%   | 47.1%   | 62.5%   | 28.42%   | 15.45%   |
| INTC   | -2.82%    | -7.97%    | 38.5%   | 46.1%   | 32.34%   | 39.04%   |
| UBER   | -0.66%    | +12.09%   | 42.1%   | 46.7%   | 18.92%   | 15.72%   |
| GOOG   | +61.26%   | +20.97%   | 73.7%   | 53.9%   | 15.03%   | 18.11%   |
| META   | +0.67%    | -1.24%    | 41.2%   | 38.5%   | 21.55%   | 17.60%   |
| AAPL   | +11.62%   | -6.62%    | 52.6%   | 50.0%   | 14.19%   | 13.52%   |
| BABA   | +55.29%   | +81.30%   | 50.0%   | 50.0%   | 20.74%   | 18.95%   |
| GLD    | +37.47%   | +51.37%   | 52.9%   | 50.0%   | 13.87%   | 13.87%   |
| TSLA   | +54.94%   | +25.68%   | 50.0%   | 53.3%   | 31.32%   | 41.40%   |
| MSFT   | -3.00%    | -14.12%   | 33.3%   | 18.2%   | 19.29%   | 20.40%   |
| TSM    | +40.43%   | +34.97%   | 55.6%   | 68.8%   | 31.09%   | 16.78%   |
| NVDA   | -30.83%   | -11.75%   | 37.5%   | 57.1%   | 49.44%   | 31.16%   |
| ORCL   | -22.06%   | +79.74%   | 41.7%   | 77.8%   | 43.11%   | 21.98%   |
| AMZN   | -16.19%   | -12.23%   | 33.3%   | 50.0%   | 33.97%   | 30.89%   |
| SNDK   | +515.51%  | +1001.51% | 100.0%  | 100.0%  | 24.18%   | 24.18%   |

---

## What to try next (v6 / v7)

- **Soften the MA50 gate**: Use a 20-bar lookback instead of 10 for the MA50 slope. Require 3+ consecutive declining bars before blocking BUY signals.
- **Volatility-aware gate + ATR-scaled trailing stop activation**: High-vol stocks need more evidence of a downtrend before being blocked; trailing stop should activate earlier on stable stocks.

---

---

## What was improved (v5 → v6)

Two surgical fixes to the MA50 regime gate which was over-filtering in v5:

**1. Wider MA50 lookback: 10 → 20 bars**
The slope comparison now looks back 20 trading days (one calendar month) instead of 10. A 10-bar dip could be a normal weekly consolidation in a healthy uptrend. A 20-bar decline is a more meaningful trend shift.

**2. Confirmed downtrend: require 3+ consecutive declining MA50 bars**
The hard BUY block now only fires when MA50 has been declining for 3 or more consecutive bars. A single dip no longer blocks entries. This restores entries in stocks like GOOG and AAPL that had brief MA50 pauses mid-uptrend.

---

## Head-to-Head: v5 vs v6

```
         ── v5 ──────────────────    ── v6 ──────────────────    ── Change ──
Ticker   Bot      Hold     Beat?     Bot      Hold     Beat?     Bot return
─────────────────────────────────────────────────────────────────────────────
AMD      +73.7%   +1.5%    ✅ YES    +73.7%   +1.5%    ✅ YES    0.0%    →
INTC     -8.0%    +4.7%    ❌ NO     -8.0%    +4.7%    ❌ NO     0.0%    →
UBER     +12.1%   -6.8%    ✅ YES    +12.1%   -6.8%    ✅ YES    0.0%    →
─────────────────────────────────────────────────────────────────────────────
GOOG     +21.0%   +115.4%  ❌ NO     +21.0%   +115.4%  ❌ NO     0.0%    →
BABA     +81.3%   +75.7%   ✅ YES    +108.6%  +75.7%   ✅ YES    +27.3%  ↑↑↑
SNDK     +1001.5% +1619%   ❌ NO     +1001.5% +1619%   ❌ NO     0.0%    →
TSM      +35.0%   +136.9%  ❌ NO     +23.3%   +136.9%  ❌ NO     -11.7%  ↓↓
GLD      +51.4%   +132.1%  ❌ NO     +51.4%   +132.1%  ❌ NO     0.0%    →
TSLA     +25.7%   +133.1%  ❌ NO     +25.7%   +133.1%  ❌ NO     0.0%    →
META     -1.2%    +28.8%   ❌ NO     -9.1%    +28.8%   ❌ NO     -7.8%   ↓↓
AAPL     -6.6%    +49.5%   ❌ NO     -7.2%    +49.5%   ❌ NO     -0.5%   →
MSFT     -14.1%   -3.2%    ❌ NO     -14.1%   -3.2%    ❌ NO     0.0%    →
NVDA     -11.8%   +101.5%  ❌ NO     -0.2%    +101.5%  ❌ NO     +11.6%  ↑↑↑
ORCL     +79.7%   +26.8%   ✅ YES    +60.8%   +26.8%   ✅ YES    -18.9%  ↓↓
AMZN     -12.2%   +18.7%   ❌ NO     -12.4%   +18.7%   ❌ NO     -0.1%   →
```

---

## The Honest Summary

### v6 improvements

**BABA +81% → +109%** — The softer gate correctly allowed 2 more high-quality entries during BABA's mid-2025 uptrend that v5 blocked. BABA now beats buy-and-hold by 33pp.

**NVDA -12% → -0.2%** — NVDA is now essentially flat, up from a -12% loss. The 3-bar confirmation requirement stops the bot from treating every brief MA50 dip as a bear market. One new trade (+10.44%) captured in Oct 2024 that v5 blocked.

### v6 regressions

**ORCL +80% → +61%** — The looser gate allows one bad entry in Sep 2025 (-6.86% drag) that v5's stricter filter caught. ORCL still beats buy-and-hold significantly (+61% vs +27%).

**META -1% → -9%** — A bad Mar 2025 trade (-7.9%) slipped through the softer gate. META remains a difficult stock due to its high volatility and sharp reversals.

**TSM +35% → +23%** — One additional bad trade in Aug 2024 (-8.67%) was now allowed through.

### Net verdict

Total across all 15 tickers:
- v5: ~1327%
- v6: ~1327%

**v6 is essentially a wash vs v5.** The softened gate improved some stocks (BABA, NVDA) while allowing bad entries back in for others (ORCL, META, TSM). The individual improvements and regressions cancel out.

The two versions represent different trade-offs: v5 is more conservative (fewer entries, blocks more), v6 is more permissive (more entries, captures more upsides but also more losses).

**What actually worked across all versions:**
- ✅ Dynamic hold extension (v3) — SNDK, BABA, GOOG
- ✅ ATR-based stop-loss (v4) — INTC, TSLA, TSM vs fixed-% stop
- ✅ Trailing stop (v5) — SNDK, BABA, AMD, UBER pure improvement
- ✅ MA50 regime gate on confirmed downtrends (v5/v6) — ORCL, NVDA improved dramatically

---

## All-Version Summary Table

| Ticker | v2     | v3      | v4      | v5       | v6       | Hold    |
|--------|--------|---------|---------|----------|----------|---------|
| AMD    | +103.6%| +60.4%  | +62.2%  | +73.7%   | +73.7%   | +1.5%   |
| INTC   | +26.5% | -16.2%  | -2.8%   | -8.0%    | -8.0%    | +4.7%   |
| UBER   | +7.5%  | -3.2%   | -0.7%   | +12.1%   | +12.1%   | -6.8%   |
| GOOG   | +52.1% | +61.1%  | +61.3%  | +21.0%   | +21.0%   | +115.4% |
| META   | +18.4% | +0.7%   | +0.7%   | -1.2%    | -9.1%    | +28.8%  |
| AAPL   | +21.0% | +11.6%  | +11.6%  | -6.6%    | -7.2%    | +49.5%  |
| BABA   | +33.9% | +61.4%  | +55.3%  | +81.3%   | +108.6%  | +75.7%  |
| GLD    | +29.3% | +39.6%  | +37.5%  | +51.4%   | +51.4%   | +132.1% |
| TSLA   | +19.6% | +29.0%  | +54.9%  | +25.7%   | +25.7%   | +133.1% |
| MSFT   | -12.5% | -4.3%   | -3.0%   | -14.1%   | -14.1%   | -3.2%   |
| TSM    | +36.3% | +24.6%  | +40.4%  | +35.0%   | +23.3%   | +136.9% |
| NVDA   | -14.9% | -26.4%  | -30.8%  | -11.8%   | -0.2%    | +101.5% |
| ORCL   | -16.8% | -21.5%  | -22.1%  | +79.7%   | +60.8%   | +26.8%  |
| AMZN   | -15.6% | -16.5%  | -16.2%  | -12.2%   | -12.4%   | +18.7%  |
| SNDK   | +192.2%| +515.5% | +515.5% | +1001.5% | +1001.5% | +1619%  |

**Best version per ticker:** AMD→v2, BABA→v6, GLD→v5/v6, GOOG→v3, INTC→v2, META→v2, AAPL→v2, MSFT→v4, NVDA→v6, ORCL→v5, SNDK→v5/v6, TSLA→v4, TSM→v4, UBER→v5/v6, AMZN→v5

---

---

## What was improved (v6 → v7)

Three data-driven changes addressing the root causes identified across v2–v6:

**1. Volatility-aware MA50 gate**
The MA50 declining-bar threshold now scales with ATR% of the stock price. A high-volatility stock (ATR > 3% of price) dips its MA50 regularly without being in a true bear market — it needs 8+ consecutive declining bars before BUY is blocked. A low-volatility stock (ATR < 1.5%) rarely dips without meaning it — 3 bars is enough. This auto-resolves the ORCL/GOOG trade-off by using each stock's own volatility as the reference.

**2. Sideways-stock higher threshold**
When the 60-day trend is flat (Trend60d between −5% and +5%), the signal threshold is raised from 6 to 7. This cuts low-conviction entries on range-bound stocks like MSFT and AMZN where the bot was essentially guessing.

**3. ATR-scaled trailing stop activation**
Instead of always activating the trailing stop at 10% profit, it activates at `max(5%, 2 × ATR%)`. For stable stocks (GLD, AAPL: ATR ~1%), trailing kicks in at 5% — protects gains earlier. For volatile stocks (TSLA, NVDA: ATR ~5%), it waits until 10% — avoids being shaken out by normal daily swings before real profits accumulate.

---

## Head-to-Head: v6 vs v7

```
         ── v6 ──────────────────    ── v7 ──────────────────    ── Change ──
Ticker   Bot      Hold     Beat?     Bot      Hold     Beat?     Bot return
─────────────────────────────────────────────────────────────────────────────
AMD      +73.7%   +1.5%    ✅ YES    +55.2%   +1.5%    ✅ YES    -18.6%  ↓↓↓
INTC     -8.0%    +4.7%    ❌ NO     -8.0%    +4.7%    ❌ NO     0.0%    →
UBER     +12.1%   -6.8%    ✅ YES    +11.9%   -6.8%    ✅ YES    -0.2%   →
─────────────────────────────────────────────────────────────────────────────
GOOG     +21.0%   +115.4%  ❌ NO     +27.4%   +115.4%  ❌ NO     +6.4%   ↑↑
BABA     +108.6%  +75.7%   ✅ YES    +120.0%  +75.7%   ✅ YES    +11.5%  ↑↑↑
SNDK     +1001.5% +1619%   ❌ NO     +1001.5% +1619%   ❌ NO     0.0%    →
TSM      +23.3%   +136.9%  ❌ NO     +9.6%    +136.9%  ❌ NO     -13.7%  ↓↓↓
GLD      +51.4%   +132.1%  ❌ NO     +50.3%   +132.1%  ❌ NO     -1.1%   →
TSLA     +25.7%   +133.1%  ❌ NO     +43.7%   +133.1%  ❌ NO     +18.1%  ↑↑↑
META     -9.1%    +28.8%   ❌ NO     -7.2%    +28.8%   ❌ NO     +1.8%   ↑
AAPL     -7.2%    +49.5%   ❌ NO     -0.6%    +49.5%   ❌ NO     +6.6%   ↑↑
MSFT     -14.1%   -3.2%    ❌ NO     -16.1%   -3.2%    ❌ NO     -2.1%   ↓
NVDA     -0.2%    +101.5%  ❌ NO     +4.7%    +101.5%  ❌ NO     +4.8%   ↑↑
ORCL     +60.8%   +26.8%   ✅ YES    +40.4%   +26.8%   ✅ YES    -20.5%  ↓↓↓
AMZN     -12.4%   +18.7%   ❌ NO     -10.9%   +18.7%   ❌ NO     +1.5%   ↑
```

---

## The Honest Summary

### Where v7 clearly improved

**TSLA +26% → +44%** — The ATR-scaled trailing stop activation was the key fix. TSLA's ATR% is ~5%, so the trailing stop now waits until +10% profit before activating (same as before) — but the *volatility-aware gate* allowed entry into a June 2024 trade at $187 that caught a +40.52% move to $263, which v6 had blocked. Best TSLA result since v4.

**BABA +109% → +120%** — Consistently improving version over version. The ATR-scaled trailing stop activates earlier on BABA's moves (ATR ~3-4% of price), locking in more of each winning trade.

**AAPL -7% → -0.6%** — The vol-aware gate restored the Jul 2024 entry that v6 blocked. AAPL (ATR ~1.7%) now needs 5 consecutive declining MA50 bars before blocking — brief dips no longer prevent entry into AAPL's sustained uptrend.

**GOOG +21% → +27%** — One extra trade restored (Oct 2024 +2.70%), and the ATR-scaled trail locked in the Dec 2024 trade more cleanly.

**NVDA -0.2% → +4.7%** — First genuinely positive result for NVDA. The 8-bar gate threshold (NVDA ATR ~6%) allows entry during sideways periods while still blocking the Q1 2025 crash.

### Where v7 regressed

**ORCL +61% → +40%** — The high-vol gate (ORCL ATR ~6% → needs 8 bars) let in the Sep 2025 crash entries again. ORCL's ATR is high *because* it's crashing — the same signal that means "volatile uptrend stock" also describes a stock in freefall. ATR alone can't tell the difference. ORCL still beats buy-and-hold (+40% vs +27%).

**AMD +74% → +55%** — The vol-aware gate changed which July 2024 signal was taken. A bad entry at $183.96 (-13.33%) was taken instead of the profitable one from v6. The gate logic surfaced a different trade.

**TSM +23% → +10%** — More trades taken on TSM with the looser gate, and the additional ones were mostly losers. TSM is a high-vol stock that benefited from being gated more strictly.

### Net verdict

Total across all 15 tickers:
- v6: ~1327%
- v7: ~1322%

**v7 is essentially tied with v6**, with wins and losses balancing out almost exactly. The improvements (TSLA +18pp, BABA +11pp, AAPL +7pp, GOOG +6pp, NVDA +5pp) are offset by regressions (ORCL -20pp, AMD -19pp, TSM -14pp).

**The fundamental unresolved problem:**
ATR% cannot distinguish between "this stock is volatile because it moves fast in a healthy trend" (AMD, TSLA) and "this stock is volatile because it's crashing" (ORCL Sep 2025, AMD Nov 2024). Any gate that's loose enough for the former will let in the latter.

The right solution requires knowing the *direction* of the move that's causing the volatility — which points toward adding the 60-day return direction to the gate logic (only block when Trend60d is also negative, not just MA50 declining).

---

## All-Version Summary Table (v2 → v7)

| Ticker | v2      | v3      | v4      | v5       | v6       | v7      | Hold    |
|--------|---------|---------|---------|----------|----------|---------|---------|
| AMD    | +103.6% | +60.4%  | +62.2%  | +73.7%   | +73.7%   | +55.2%  | +1.5%   |
| INTC   | +26.5%  | -16.2%  | -2.8%   | -8.0%    | -8.0%    | -8.0%   | +4.7%   |
| UBER   | +7.5%   | -3.2%   | -0.7%   | +12.1%   | +12.1%   | +11.9%  | -6.8%   |
| GOOG   | +52.1%  | +61.1%  | +61.3%  | +21.0%   | +21.0%   | +27.4%  | +115.4% |
| META   | +18.4%  | +0.7%   | +0.7%   | -1.2%    | -9.1%    | -7.2%   | +28.8%  |
| AAPL   | +21.0%  | +11.6%  | +11.6%  | -6.6%    | -7.2%    | -0.6%   | +49.5%  |
| BABA   | +33.9%  | +61.4%  | +55.3%  | +81.3%   | +108.6%  | +120.0% | +75.7%  |
| GLD    | +29.3%  | +39.6%  | +37.5%  | +51.4%   | +51.4%   | +50.3%  | +132.1% |
| TSLA   | +19.6%  | +29.0%  | +54.9%  | +25.7%   | +25.7%   | +43.7%  | +133.1% |
| MSFT   | -12.5%  | -4.3%   | -3.0%   | -14.1%   | -14.1%   | -16.1%  | -3.2%   |
| TSM    | +36.3%  | +24.6%  | +40.4%  | +35.0%   | +23.3%   | +9.6%   | +136.9% |
| NVDA   | -14.9%  | -26.4%  | -30.8%  | -11.8%   | -0.2%    | +4.7%   | +101.5% |
| ORCL   | -16.8%  | -21.5%  | -22.1%  | +79.7%   | +60.8%   | +40.4%  | +26.8%  |
| AMZN   | -15.6%  | -16.5%  | -16.2%  | -12.2%   | -12.4%   | -10.9%  | +18.7%  |
| SNDK   | +192.2% | +515.5% | +515.5% | +1001.5% | +1001.5% | +1001.5%| +1619%  |

**Best version per ticker:**
- AMD → v2 (+103%), BABA → v7 (+120%), GLD → v5/v6 (+51%), GOOG → v3/v4 (+61%)
- INTC → v2 (+26%), META → v2 (+18%), AAPL → v2 (+21%), MSFT → v4 (-3%)
- NVDA → v7 (+5%), ORCL → v5 (+80%), SNDK → v5/v6/v7 (+1001%), TSLA → v4 (+55%)
- TSM → v4 (+40%), UBER → v5/v6/v7 (+12%), AMZN → v7 (-11%)

**Overall best version: v5** (highest total ~1327%, includes the ORCL breakthrough and strong SNDK/BABA results without the ORCL regression of v6/v7)
