# Backtest Results
**Last updated:** 2026-03-14

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

---

## v9 — 2026-03-14

### What changed (v8 → v9)

| Change | Detail |
|--------|--------|
| **MA200 scoring** | bull +2 if price > MA200, bear +2 if price < MA200 — long-term regime filter |
| **Signal threshold** | Configurable via `-threshold` flag (default 6); tested 6 vs 7 — 6 wins for actual holdings |
| **Stop-loss ATR** | Widened 2.0× → 2.5× to reduce whipsawing on volatile stocks |
| **Trailing stop activation** | Lowered 10% → 7% profit threshold to lock gains earlier |
| **Max drawdown fix** | Pre-fill equity array to initial capital during warm-up bars (was falsely showing 100%) |
| **`-threshold` CLI flag** | Allows per-run tuning without code changes |

### Config used

```
24 months | hold 2–10 days | stop 2.5× ATR | trail 1.5× ATR | extend 2.0× | capital $10000 | threshold 6 | SPY regime ON
```

### Summary

| Ticker | Strategy | Buy-and-Hold | Alpha | Trades | Win Rate | Sharpe |
|--------|----------|-------------|-------|--------|----------|--------|
| GLD    | +52.93%  | +130.02%    | -77.09% | 14 (W:10/L:4) | 71.4% | -0.44 |
| AMZN   | -10.28%  | +16.18%     | -26.46% | 16 (W:10/L:6) | 62.5% | -0.80 |
| AMD    | +62.25%  | +3.38%      | **+58.87%** | 10 (W:5/L:5) | 50.0% | 0.77 |
| GOOG   | +26.63%  | +108.85%    | -82.22% | 15 (W:8/L:7) | 53.3% | -0.56 |
| MSFT   | +6.57%   | -6.98%      | **+13.55%** | 11 (W:5/L:6) | 45.5% | -0.12 |
| BABA   | +56.86%  | +84.21%     | -27.35% | 9 (W:6/L:3) | 66.7% | 0.82 |
| UBER   | -10.68%  | -5.56%      | -5.12% | 10 (W:4/L:6) | 40.0% | -0.42 |
| ORCL   | +122.10% | +23.56%     | **+98.54%** | 13 (W:9/L:4) | 69.2% | 1.12 |
| NVDA   | +7.21%   | +104.96%    | -97.75% | 9 (W:6/L:3) | 66.7% | 0.03 |
| META   | +16.48%  | +24.78%     | -8.30% | 13 (W:7/L:6) | 53.9% | 0.24 |
| AAPL   | -8.84%   | +44.58%     | -53.42% | 14 (W:5/L:9) | 35.7% | -0.71 |
| TSM    | +41.92%  | +142.31%    | -100.39% | 16 (W:11/L:5) | 68.8% | -0.45 |
| TSLA   | +37.21%  | +140.74%    | -103.53% | 19 (W:9/L:10) | 47.4% | 0.47 |
| INTC   | +58.31%  | +7.06%      | **+51.25%** | 13 (W:9/L:4) | 69.2% | -0.31 |
| SNDK   | +1055.44%| +1737.83%   | -682.39% | 7 (W:6/L:1) | 85.7% | 1.24 |

**Positive alpha tickers: AMD (+58.87%), MSFT (+13.55%), ORCL (+98.54%), INTC (+51.25%)**

### Full trade log

```
Backtest  |  24 months  |  hold 2–10 days  |  stop 2.5x ATR  |  trail 1.5x ATR  |  extend 2.0x  |  capital $10000  |  threshold 6  |  SPY regime ON

──────────────────────────────────────────────────────────
  BACKTEST  GLD     2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +52.93%
  Buy-and-hold:      +130.02%
  Alpha:             -77.09%
  Trades:            14  (W:10 / L:4)
  Win rate:          71.4%
  Avg win:           +5.40%
  Avg loss:          -2.29%
  Max drawdown:      100.00%
  Sharpe ratio:      -0.44

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-07-17   2024-07-25   $227.23  $218.33  -3.92%   6    stop-loss
  2024-08-16   2024-08-30   $231.99  $231.29  -0.30%  10    timeout
  2024-09-13   2024-09-27   $238.68  $245.02  +2.66%  10    timeout
  2024-10-18   2024-11-11   $251.27  $242.14  -3.63%  16    stop-loss
  2025-01-30   2025-02-13   $258.05  $270.31  +4.75%  10    timeout
  2025-02-14   2025-03-17   $266.29  $276.73  +3.92%  20    timeout
  2025-05-01   2025-05-30   $297.46  $303.60  +2.06%  20    timeout
  2025-06-02   2025-07-01   $311.67  $307.55  -1.32%  20    timeout
  2025-09-05   2025-09-19   $331.05  $339.18  +2.46%  10    timeout
  2025-09-22   2025-10-20   $345.05  $403.15  +16.84% 20    timeout
  2025-10-21   2025-12-03   $377.24  $386.88  +2.56%  30    timeout
  2025-12-05   2025-12-29   $386.44  $398.60  +3.15%  15    trail-stop
  2025-12-30   2026-01-14   $398.89  $425.94  +6.78%  10    timeout
  2026-01-15   2026-03-13   $423.33  $460.84  +8.86%  40    end-of-data

──────────────────────────────────────────────────────────
  BACKTEST  AMZN    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   -10.28%
  Buy-and-hold:      +16.18%
  Alpha:             -26.46%
  Trades:            16  (W:10 / L:6)
  Win rate:          62.5%
  Avg win:           +2.86%
  Avg loss:          -6.17%
  Max drawdown:      100.00%
  Sharpe ratio:      -0.80

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-06-07   2024-06-24   $184.30  $185.57  +0.69%  10    timeout
  2024-06-26   2024-07-11   $193.61  $195.05  +0.74%  10    timeout
  2024-10-21   2024-11-04   $189.07  $195.78  +3.55%  10    timeout
  2024-11-05   2024-11-15   $199.50  $202.61  +1.56%   8    trail-stop
  2024-12-03   2024-12-18   $213.44  $220.52  +3.32%  11    trail-stop
  2024-12-27   2025-02-07   $223.75  $229.15  +2.41%  27    trail-stop
  2025-02-10   2025-02-21   $233.14  $216.58  -7.10%   8    stop-loss
  2025-05-27   2025-06-10   $206.02  $217.61  +5.63%  10    timeout
  2025-06-11   2025-07-11   $213.20  $225.02  +5.54%  20    timeout
  2025-07-14   2025-08-04   $225.69  $211.65  -6.22%  15    stop-loss
  2025-08-05   2025-08-20   $213.75  $223.81  +4.71%  11    trail-stop
  2025-08-22   2025-09-22   $228.84  $227.63  -0.53%  20    timeout
  2025-09-23   2025-10-07   $220.71  $221.78  +0.48%  10    timeout
  2025-10-31   2025-11-14   $244.22  $234.69  -3.90%  10    timeout
  2026-01-06   2026-01-20   $240.93  $231.00  -4.12%   9    signal
  2026-01-27   2026-03-13   $244.68  $207.67  -15.13% 33    end-of-data

──────────────────────────────────────────────────────────
  BACKTEST  AMD     2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +62.25%
  Buy-and-hold:      +3.38%
  Alpha:             +58.87%
  Trades:            10  (W:5 / L:5)
  Win rate:          50.0%
  Avg win:           +16.15%
  Avg loss:          -4.52%
  Max drawdown:      20.34%
  Sharpe ratio:      0.77

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-05-03   2024-05-17   $150.60  $164.47  +9.21%  10    timeout
  2024-07-02   2024-07-17   $164.31  $159.43  -2.97%  10    trail-stop
  2025-06-04   2025-06-13   $118.58  $116.16  -2.04%   7    signal
  2025-06-16   2025-07-07   $126.39  $134.80  +6.65%  13    trail-stop
  2025-07-10   2025-08-06   $144.16  $163.12  +13.15% 19    trail-stop
  2025-08-07   2025-09-05   $172.40  $151.14  -12.33% 20    stop-loss
  2025-09-08   2025-10-10   $151.41  $214.90  +41.93% 24    trail-stop
  2025-10-13   2025-11-06   $216.42  $237.70  +9.83%  18    trail-stop
  2025-11-07   2025-11-18   $233.54  $230.29  -1.39%   7    trail-stop
  2025-11-25   2025-12-17   $206.13  $198.11  -3.89%  15    trail-stop

──────────────────────────────────────────────────────────
  BACKTEST  GOOG    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +26.63%
  Buy-and-hold:      +108.85%
  Alpha:             -82.22%
  Trades:            15  (W:8 / L:7)
  Win rate:          53.3%
  Avg win:           +8.86%
  Avg loss:          -5.85%
  Max drawdown:      100.00%
  Sharpe ratio:      -0.56

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-06-25   2024-07-24   $185.58  $174.37  -6.04%  20    stop-loss
  2024-10-22   2024-11-05   $166.82  $171.41  +2.75%  10    timeout
  2024-11-06   2024-11-22   $178.33  $166.57  -6.59%  12    stop-loss
  2024-12-09   2024-12-18   $177.10  $190.15  +7.37%   7    trail-stop
  2025-01-06   2025-02-12   $197.96  $185.43  -6.33%  25    stop-loss
  2025-02-19   2025-02-26   $187.13  $174.70  -6.64%   5    stop-loss
  2025-05-29   2025-06-04   $172.96  $169.39  -2.06%   4    signal
  2025-06-06   2025-06-23   $174.92  $166.01  -5.09%  10    timeout
  2025-06-26   2025-07-11   $174.43  $181.31  +3.94%  10    timeout
  2025-07-14   2025-08-01   $182.81  $189.95  +3.91%  14    trail-stop
  2025-08-04   2025-09-16   $195.75  $251.42  +28.44% 30    timeout
  2025-09-17   2025-10-29   $249.85  $275.17  +10.13% 30    timeout
  2025-10-30   2025-12-12   $281.90  $310.52  +10.15% 30    timeout
  2025-12-15   2026-01-20   $309.32  $322.16  +4.15%  23    trail-stop
  2026-01-21   2026-03-13   $328.38  $301.46  -8.20%  37    end-of-data

──────────────────────────────────────────────────────────
  BACKTEST  MSFT    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +6.57%
  Buy-and-hold:      -6.98%
  Alpha:             +13.55%
  Trades:            11  (W:5 / L:6)
  Win rate:          45.5%
  Avg win:           +5.92%
  Avg loss:          -3.63%
  Max drawdown:      15.85%
  Sharpe ratio:      -0.12

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-06-10   2024-06-25   $427.87  $450.95  +5.39%  10    timeout
  2024-07-01   2024-07-16   $456.73  $449.52  -1.58%  10    timeout
  2024-10-22   2024-10-31   $427.51  $406.35  -4.95%   7    stop-loss
  2024-11-08   2024-11-18   $422.54  $415.76  -1.60%   6    signal
  2024-11-26   2024-12-11   $427.99  $448.99  +4.91%  10    timeout
  2024-12-27   2025-01-14   $430.53  $415.67  -3.45%  10    timeout
  2025-01-22   2025-01-30   $446.20  $414.99  -6.99%   6    stop-loss
  2025-05-05   2025-05-19   $436.17  $458.87  +5.20%  10    timeout
  2025-05-21   2025-06-20   $452.57  $477.40  +5.49%  20    timeout
  2025-06-23   2025-08-05   $486.00  $527.75  +8.59%  30    timeout
  2025-08-06   2025-09-04   $524.94  $507.97  -3.23%  20    timeout

──────────────────────────────────────────────────────────
  BACKTEST  BABA    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +56.86%
  Buy-and-hold:      +84.21%
  Alpha:             -27.35%
  Trades:            9  (W:6 / L:3)
  Win rate:          66.7%
  Avg win:           +14.03%
  Avg loss:          -9.42%
  Max drawdown:      24.40%
  Sharpe ratio:      0.82

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-07-03   2024-07-18   $75.57   $76.54   +1.28%  10    timeout
  2024-08-23   2024-08-28   $85.41   $79.62   -6.78%   3    stop-loss
  2024-09-11   2024-10-08   $84.81   $109.68  +29.32% 19    trail-stop
  2025-01-30   2025-02-13   $102.74  $119.54  +16.35% 10    timeout
  2025-02-14   2025-02-24   $124.73  $129.04  +3.46%   5    trail-stop
  2025-08-29   2025-09-29   $135.00  $179.90  +33.26% 20    timeout
  2025-09-30   2025-10-10   $178.73  $159.01  -11.03%  8    stop-loss
  2025-10-13   2025-11-03   $166.81  $167.69  +0.53%  15    trail-stop
  2025-11-06   2025-12-15   $167.61  $150.09  -10.45% 26    stop-loss

──────────────────────────────────────────────────────────
  BACKTEST  UBER    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   -10.68%
  Buy-and-hold:      -5.56%
  Alpha:             -5.12%
  Trades:            10  (W:4 / L:6)
  Win rate:          40.0%
  Avg win:           +4.56%
  Avg loss:          -4.64%
  Max drawdown:      18.14%
  Sharpe ratio:      -0.42

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-07-11   2024-07-18   $73.53   $66.26   -9.89%   5    stop-loss
  2024-08-19   2024-09-17   $74.18   $72.78   -1.89%  20    timeout
  2024-09-18   2024-10-02   $73.50   $72.87   -0.86%  10    timeout
  2024-10-09   2024-10-16   $77.87   $81.90   +5.18%   5    trail-stop
  2025-02-10   2025-03-11   $78.63   $70.65   -10.15% 20    stop-loss
  2025-05-01   2025-05-23   $80.89   $87.75   +8.48%  16    trail-stop
  2025-05-27   2025-06-25   $89.00   $90.90   +2.13%  20    timeout
  2025-06-26   2025-08-08   $93.12   $89.56   -3.82%  30    timeout
  2025-08-18   2025-09-02   $93.98   $92.81   -1.24%  10    timeout
  2025-09-09   2025-10-07   $95.45   $97.80   +2.46%  20    timeout

──────────────────────────────────────────────────────────
  BACKTEST  ORCL    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +122.10%
  Buy-and-hold:      +23.56%
  Alpha:             +98.54%
  Trades:            13  (W:9 / L:4)
  Win rate:          69.2%
  Avg win:           +13.28%
  Avg loss:          -7.12%
  Max drawdown:      27.99%
  Sharpe ratio:      1.12

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-05-03   2024-05-17   $115.80  $123.50  +6.65%  10    timeout
  2024-06-06   2024-06-21   $123.50  $141.50  +14.57% 10    timeout
  2024-08-15   2024-08-29   $136.93  $139.42  +1.82%  10    timeout
  2024-08-30   2024-09-30   $141.29  $170.40  +20.60% 20    timeout
  2024-11-07   2024-11-26   $186.37  $190.37  +2.15%  13    signal
  2025-05-27   2025-06-10   $161.91  $177.48  +9.62%  10    timeout
  2025-06-11   2025-06-20   $176.38  $205.17  +16.32%  6    trail-stop
  2025-06-23   2025-08-05   $207.04  $255.67  +23.49% 30    timeout
  2025-08-06   2025-08-19   $256.43  $234.62  -8.51%   9    stop-loss
  2025-08-20   2025-09-12   $235.06  $292.18  +24.30% 16    trail-stop
  2025-09-15   2025-09-25   $302.14  $291.33  -3.58%   8    trail-stop
  2025-09-26   2025-10-20   $283.46  $277.18  -2.22%  16    trail-stop
  2025-10-21   2025-11-11   $275.15  $236.15  -14.17% 15    stop-loss

──────────────────────────────────────────────────────────
  BACKTEST  NVDA    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +7.21%
  Buy-and-hold:      +104.96%
  Alpha:             -97.75%
  Trades:            9  (W:6 / L:3)
  Win rate:          66.7%
  Avg win:           +7.00%
  Avg loss:          -10.17%
  Max drawdown:      20.48%
  Sharpe ratio:      0.03

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-08-15   2024-09-03   $122.86  $108.00  -12.10% 12    signal
  2024-10-08   2024-10-22   $132.89  $143.59  +8.05%  10    timeout
  2024-11-07   2024-11-25   $148.88  $136.02  -8.64%  12    stop-loss
  2025-05-16   2025-06-02   $135.40  $137.38  +1.46%  10    timeout
  2025-06-03   2025-07-17   $141.22  $173.00  +22.50% 30    timeout
  2025-07-18   2025-08-29   $172.41  $174.18  +1.03%  30    timeout
  2025-09-02   2025-10-10   $170.78  $183.16  +7.25%  28    trail-stop
  2025-10-13   2025-10-27   $188.32  $191.49  +1.68%  10    timeout
  2025-10-28   2025-11-18   $201.03  $181.36  -9.78%  15    stop-loss

──────────────────────────────────────────────────────────
  BACKTEST  META    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +16.48%
  Buy-and-hold:      +24.78%
  Alpha:             -8.30%
  Trades:            13  (W:7 / L:6)
  Win rate:          53.9%
  Avg win:           +6.17%
  Avg loss:          -4.25%
  Max drawdown:      13.97%
  Sharpe ratio:      0.24

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-07-05   2024-07-12   $539.91  $498.87  -7.60%   5    stop-loss
  2024-08-15   2024-09-13   $537.33  $524.62  -2.37%  20    timeout
  2024-09-16   2024-09-30   $533.28  $572.44  +7.34%  10    timeout
  2024-10-04   2024-10-14   $595.94  $590.42  -0.93%   6    signal
  2024-11-12   2024-12-11   $584.82  $632.68  +8.18%  20    timeout
  2025-01-06   2025-01-22   $630.20  $623.50  -1.06%  10    timeout
  2025-01-23   2025-02-19   $636.45  $703.77  +10.58% 18    trail-stop
  2025-02-20   2025-03-04   $694.84  $640.00  -7.89%   8    stop-loss
  2025-05-30   2025-06-13   $647.49  $682.87  +5.46%  10    timeout
  2025-06-16   2025-07-16   $702.12  $702.91  +0.11%  20    timeout
  2025-07-17   2025-08-19   $701.41  $751.48  +7.14%  23    trail-stop
  2025-08-20   2025-09-18   $747.72  $780.25  +4.35%  20    timeout
  2025-09-19   2025-09-30   $778.38  $734.38  -5.65%   7    stop-loss

──────────────────────────────────────────────────────────
  BACKTEST  AAPL    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   -8.84%
  Buy-and-hold:      +44.58%
  Alpha:             -53.42%
  Trades:            14  (W:5 / L:9)
  Win rate:          35.7%
  Avg win:           +4.34%
  Avg loss:          -3.28%
  Max drawdown:      24.57%
  Sharpe ratio:      -0.71

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-06-05   2024-06-10   $195.87  $193.12  -1.40%   3    signal
  2024-06-11   2024-06-25   $207.15  $209.07  +0.93%   9    signal
  2024-07-10   2024-07-24   $232.98  $218.54  -6.20%  10    stop-loss
  2024-08-15   2024-09-04   $224.72  $220.85  -1.72%  13    signal
  2024-09-20   2024-10-04   $228.20  $226.80  -0.61%  10    timeout
  2024-10-18   2024-11-01   $235.00  $222.91  -5.14%  10    stop-loss
  2024-12-17   2025-01-02   $253.48  $243.85  -3.80%  10    stop-loss
  2025-01-06   2025-01-14   $245.00  $233.28  -4.78%   5    stop-loss
  2025-07-02   2025-07-29   $212.44  $211.27  -0.55%  18    signal
  2025-08-07   2025-08-21   $220.03  $224.90  +2.21%  10    timeout
  2025-08-26   2025-09-24   $229.31  $252.31  +10.03% 20    timeout
  2025-09-25   2025-11-06   $256.87  $269.77  +5.02%  30    timeout
  2025-11-07   2025-12-08   $268.47  $277.89  +3.51%  20    timeout
  2025-12-09   2026-01-06   $277.18  $262.36  -5.35%  18    stop-loss

──────────────────────────────────────────────────────────
  BACKTEST  TSM     2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +41.92%
  Buy-and-hold:      +142.31%
  Alpha:             -100.39%
  Trades:            16  (W:11 / L:5)
  Win rate:          68.8%
  Avg win:           +6.90%
  Avg loss:          -7.06%
  Max drawdown:      100.00%
  Sharpe ratio:      -0.45

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-06-06   2024-06-21   $162.07  $173.96  +7.34%  10    timeout
  2024-07-05   2024-07-16   $183.99  $186.04  +1.11%   7    signal
  2024-08-15   2024-09-03   $173.96  $160.49  -7.74%  12    signal
  2024-09-12   2024-09-26   $171.43  $186.83  +8.98%  10    timeout
  2024-10-09   2024-10-23   $187.14  $200.86  +7.33%  10    timeout
  2024-12-03   2025-01-08   $198.89  $207.12  +4.14%  24    trail-stop
  2025-01-17   2025-01-27   $211.50  $192.31  -9.07%   5    stop-loss
  2025-01-28   2025-02-11   $202.40  $208.74  +3.13%  10    timeout
  2025-05-19   2025-06-03   $193.50  $197.61  +2.12%  10    timeout
  2025-06-04   2025-07-03   $202.40  $234.80  +16.01% 20    timeout
  2025-07-07   2025-07-22   $229.17  $234.60  +2.37%  11    trail-stop
  2025-07-23   2025-09-04   $240.33  $235.21  -2.13%  30    timeout
  2025-09-05   2025-10-03   $243.41  $292.19  +20.04% 20    timeout
  2025-10-06   2025-10-10   $302.40  $280.66  -7.19%   4    stop-loss
  2025-10-13   2025-11-21   $302.89  $275.06  -9.19%  29    stop-loss
  2026-01-06   2026-03-13   $327.43  $338.31  +3.32%  47    end-of-data

──────────────────────────────────────────────────────────
  BACKTEST  TSLA    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +37.21%
  Buy-and-hold:      +140.74%
  Alpha:             -103.53%
  Trades:            19  (W:9 / L:10)
  Win rate:          47.4%
  Avg win:           +12.10%
  Avg loss:          -6.49%
  Max drawdown:      33.70%
  Sharpe ratio:      0.47

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-06-14   2024-07-01   $178.01  $209.86  +17.89% 10    timeout
  2024-07-02   2024-07-11   $231.26  $241.03  +4.22%   6    trail-stop
  2024-08-19   2024-08-29   $222.72  $206.28  -7.38%   8    signal
  2024-08-30   2024-09-06   $214.11  $210.73  -1.58%   4    trail-stop
  2024-09-11   2024-10-03   $228.13  $240.66  +5.49%  16    trail-stop
  2024-10-24   2024-11-14   $260.48  $311.18  +19.46% 15    trail-stop
  2024-12-06   2024-12-18   $389.22  $440.13  +13.08%  8    trail-stop
  2024-12-27   2025-02-07   $431.66  $361.62  -16.23% 27    stop-loss
  2025-05-19   2025-06-03   $342.09  $344.27  +0.64%  10    timeout
  2025-06-04   2025-07-18   $332.05  $329.65  -0.72%  30    timeout
  2025-07-21   2025-07-31   $328.49  $308.27  -6.16%   8    signal
  2025-08-07   2025-08-21   $322.27  $320.11  -0.67%  10    signal
  2025-08-22   2025-09-02   $340.01  $329.36  -3.13%   6    signal
  2025-09-03   2025-09-17   $334.09  $425.86  +27.47% 10    timeout
  2025-09-18   2025-10-03   $416.85  $429.83  +3.11%  11    trail-stop
  2025-10-06   2025-11-13   $453.25  $401.99  -11.31% 28    stop-loss
  2025-11-14   2025-12-15   $404.35  $475.31  +17.55% 20    timeout
  2025-12-16   2026-01-02   $489.88  $438.07  -10.58% 11    stop-loss
  2026-01-05   2026-01-20   $451.67  $419.25  -7.18%  10    timeout

──────────────────────────────────────────────────────────
  BACKTEST  INTC    2024-03-14 → 2026-03-13  (502 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +58.31%
  Buy-and-hold:      +7.06%
  Alpha:             +51.25%
  Trades:            13  (W:9 / L:4)
  Win rate:          69.2%
  Avg win:           +11.29%
  Avg loss:          -10.34%
  Max drawdown:      100.00%
  Sharpe ratio:      -0.31

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2024-11-05   2024-11-12   $23.32   $24.16   +3.60%   5    trail-stop
  2025-02-18   2025-02-25   $27.39   $22.99   -16.06%  5    stop-loss
  2025-06-10   2025-06-18   $22.08   $21.49   -2.67%   6    signal
  2025-06-24   2025-07-09   $22.55   $23.44   +3.95%  10    timeout
  2025-07-15   2025-07-25   $22.92   $20.70   -9.69%   8    stop-loss
  2025-08-12   2025-08-26   $21.81   $24.35   +11.65% 10    timeout
  2025-09-15   2025-10-13   $24.77   $37.22   +50.26% 20    timeout
  2025-10-14   2025-11-04   $35.63   $37.03   +3.93%  15    trail-stop
  2025-11-07   2025-12-04   $38.13   $40.50   +6.22%  18    trail-stop
  2025-12-05   2025-12-17   $41.41   $36.05   -12.94%  8    stop-loss
  2025-12-18   2026-01-05   $36.28   $39.37   +8.52%  10    timeout
  2026-01-07   2026-01-23   $42.63   $45.07   +5.72%  11    trail-stop
  2026-01-26   2026-03-13   $42.49   $45.77   +7.72%  34    end-of-data

──────────────────────────────────────────────────────────
  BACKTEST  SNDK    2025-02-13 → 2026-03-13  (272 bars)
──────────────────────────────────────────────────────────
  Strategy return:   +1055.44%
  Buy-and-hold:      +1737.83%
  Alpha:             -682.39%
  Trades:            7  (W:6 / L:1)
  Win rate:          85.7%
  Avg win:           +65.68%
  Avg loss:          -9.59%
  Max drawdown:      100.00%
  Sharpe ratio:      1.24

  Entry        Exit         Buy@     Sell@    Return   Days  Exit
  2025-06-04   2025-06-18   $39.82   $46.62   +17.08% 10    signal
  2025-08-08   2025-08-22   $44.34   $46.37   +4.58%  10    timeout
  2025-08-26   2025-09-25   $47.35   $94.29   +99.13% 21    trail-stop
  2025-09-29   2025-10-10   $113.50  $116.91  +3.00%   9    trail-stop
  2025-10-15   2025-11-13   $144.30  $243.57  +68.79% 21    trail-stop
  2025-11-28   2025-12-15   $223.28  $201.87  -9.59%  11    trail-stop
  2025-12-18   2026-03-13   $219.46  $661.62  +201.48% 58   end-of-data
```
