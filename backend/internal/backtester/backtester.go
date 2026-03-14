package backtester

import (
	"fmt"
	"math"
	"strings"
	"time"

	"jobot/internal/indicators"
	"jobot/internal/memory"
)

const minBarsNeeded = 35 // minimum bars required to compute MACD

// Config controls backtest parameters.
type Config struct {
	MaxHoldDays             int
	MinHoldDays             int     // minimum bars before a SELL signal is respected
	StopLossATRMultiple     float64 // exit if price drops more than N×ATR14 from entry (0 = disabled)
	TrailingStopATRMultiple float64 // once trade up 10%+, trail stop at peak - N×ATR14 (0 = disabled)
	TrendExtendFactor       float64 // multiplier on MaxHoldDays when Trend60d is strong (0 = disabled)
	InitialCapital          float64
	MarketCandles           *indicators.Candles // SPY candles for market regime filter (nil = disabled)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxHoldDays:             10,
		MinHoldDays:             2,
		StopLossATRMultiple:     2.5, // stop out when price drops 2.5×ATR14 from entry (v9: widened from 2.0)
		TrailingStopATRMultiple: 1.5, // once up 7%, trail at peak - 1.5×ATR14
		TrendExtendFactor:       2.0,
		InitialCapital:          10_000,
	}
}

// tradeMaxHold computes the effective max hold days for a new trade.
// If the trend at entry is strong, the hold period is extended so trending
// stocks have room to run rather than being exited too early.
func tradeMaxHold(ind indicators.Indicators, cfg Config) int {
	if cfg.TrendExtendFactor <= 0 || ind.Trend60d == nil {
		return cfg.MaxHoldDays
	}
	trend := *ind.Trend60d
	switch {
	case trend > 20:
		return int(float64(cfg.MaxHoldDays) * cfg.TrendExtendFactor * 1.5) // e.g. 30 days
	case trend > 10:
		return int(float64(cfg.MaxHoldDays) * cfg.TrendExtendFactor) // e.g. 20 days
	default:
		return cfg.MaxHoldDays
	}
}

// Trade records a single completed round-trip trade.
type Trade struct {
	EntryDate  string
	ExitDate   string
	EntryPrice float64
	ExitPrice  float64
	ReturnPct  float64
	HoldDays   int
	ExitReason string // "signal" | "timeout" | "end-of-data"
}

// MemoryDecision evaluates a single stored Claude decision against actual forward price.
type MemoryDecision struct {
	Date      string
	Decision  string
	Price     float64
	Price5d   float64 // actual price 5 trading days later
	ReturnPct float64
	Correct   bool
}

// DecisionStat tracks per-decision accuracy.
type DecisionStat struct {
	Total   int
	Correct int
}

// MemoryEval aggregates Claude decision accuracy from stored memory entries.
type MemoryEval struct {
	Total       int
	Correct     int
	AccuracyPct float64
	AvgReturn5d float64            // avg return 5d after BUY decisions
	ByDecision  map[string]*DecisionStat
	Details     []MemoryDecision
}

// Result is the full output of one backtest run.
type Result struct {
	Ticker        string
	StartDate     string
	EndDate       string
	TotalBars     int
	// Indicator-strategy simulation metrics
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalReturn   float64 // pct vs initial capital
	BuyHoldReturn float64 // pct
	AvgWinPct     float64
	AvgLossPct    float64
	MaxDrawdownPct float64
	SharpeRatio   float64
	Trades        []Trade
	// Claude decision accuracy from stored memory
	Memory        *MemoryEval
}

// signal returns "BUY", "SELL", or "HOLD" using a multi-factor scoring model.
//
// Each indicator contributes to a bull or bear score. A signal fires only when
// the net score reaches the threshold AND leads the opposing side — this avoids
// single-indicator false triggers and falling-knife buys.
//
//	bull/bear score components:
//	  RSI extremes          ±1..3
//	  MACD histogram cross  ±3  (strongest signal)
//	  MACD histogram trend  ±1  (building momentum)
//	  Price vs MA50         ±1  (trend filter)
//	  Price vs MA20         ±1  (short-term trend)
//	  60d trend             ±1  (broader momentum)
//	  MA50 slope            ±2  (regime: rising vs declining MA50)
//	  Price vs MA200        ±2  (long-term trend regime — v9)
//
//	Threshold: net score ≥ 7 required to fire BUY or SELL (raised from 6 in v9)
//
//	Additional gates (v8):
//	  Volume filter:  CurVol < 70% AvgVol → no BUY (thin-volume day)
//	  Weekly MACD:    weekly histogram < 0 → no BUY (weekly momentum bearish)
//
func signal(ind indicators.Indicators, currentClose, prevHist, prevMA50 float64, weeklyHist *float64) string {
	if ind.RSI == nil || ind.MACD == nil || ind.MACD.Histogram == nil {
		return "HOLD"
	}
	rsi := *ind.RSI
	hist := *ind.MACD.Histogram

	macdCrossedUp := prevHist <= 0 && hist > 0
	macdCrossedDown := prevHist >= 0 && hist < 0
	histRising := hist > prevHist
	histFalling := hist < prevHist

	// Hard gate: block BUY when MA50 is declining AND price is below it.
	// A single bar of declining MA50 is enough — this catches prolonged downtrends
	// (ORCL H2 2025, NVDA Q1 2025) while still allowing entries on any dip recovery.
	if ind.MA50 != nil && prevMA50 > 0 {
		if *ind.MA50 < prevMA50 && currentClose < *ind.MA50 {
			goto computeSell
		}
	}

	{
		var bull, bear int

		switch {
		case rsi < 30:
			bull += 3
		case rsi < 40:
			bull += 2
		case rsi < 50:
			bull += 1
		case rsi > 70:
			bear += 3
		case rsi > 60:
			bear += 2
		case rsi > 55:
			bear += 1
		}

		if macdCrossedUp {
			bull += 3
		}
		if macdCrossedDown {
			bear += 3
		}

		if hist > 0 && histRising {
			bull += 1
		}
		if hist < 0 && histFalling {
			bear += 1
		}

		if ind.MA50 != nil {
			if currentClose > *ind.MA50 {
				bull += 1
			} else {
				bear += 1
			}
		}

		if ind.MA50 != nil && prevMA50 > 0 {
			if *ind.MA50 > prevMA50 {
				bull += 2
			} else {
				bear += 2
			}
		}

		if ind.MA20 != nil {
			if currentClose > *ind.MA20 {
				bull += 1
			} else {
				bear += 1
			}
		}

		if ind.Trend60d != nil {
			if *ind.Trend60d > 5 {
				bull += 1
			} else if *ind.Trend60d < -5 {
				bear += 1
			}
		}

		// MA200: long-term trend regime — strongest regime filter (v9)
		if ind.MA200 != nil {
			if currentClose > *ind.MA200 {
				bull += 2
			} else {
				bear += 2
			}
		}

		const threshold = 7
		if bull >= threshold && bull > bear {
			// Volume filter: skip thin-volume days — low participation means unreliable signal
			if ind.CurVol != nil && ind.AvgVol != nil {
				if *ind.CurVol < 0.7*float64(*ind.AvgVol) {
					return "HOLD"
				}
			}
			// Weekly MACD confirmation: require positive weekly momentum before entering
			if weeklyHist != nil && *weeklyHist < 0 {
				return "HOLD"
			}
			return "BUY"
		}
		if bear >= threshold && bear > bull {
			return "SELL"
		}
		return "HOLD"
	}

computeSell:
	{
		// In bear regime: only look for SELL signals (short exits or overbought)
		var bear int
		if rsi > 65 {
			bear += 2
		}
		if macdCrossedDown {
			bear += 3
		}
		if hist < 0 && histFalling {
			bear += 1
		}
		if ind.MA50 != nil && currentClose < *ind.MA50 {
			bear += 1
		}
		if ind.MA20 != nil && currentClose < *ind.MA20 {
			bear += 1
		}
		if bear >= 4 {
			return "SELL"
		}
		return "HOLD"
	}
}

// buildSPYRegimeMap returns a date → isBullish map for every bar in the SPY candles.
// A day is bullish when SPY close > SPY MA50.
func buildSPYRegimeMap(candles indicators.Candles) map[string]bool {
	result := make(map[string]bool, len(candles.T))
	for i := 50; i < len(candles.C); i++ {
		sub := subCandles(candles, i+1)
		ind := indicators.ComputeAll(sub)
		day := time.Unix(candles.T[i], 0).UTC().Format("2006-01-02")
		if ind.MA50 != nil {
			result[day] = candles.C[i] > *ind.MA50
		}
	}
	return result
}

// resampleWeekly converts daily closes to weekly by taking every 5th bar.
// The last bar is always included so the most recent week is represented.
func resampleWeekly(closes []float64) []float64 {
	n := len(closes)
	if n == 0 {
		return nil
	}
	var weekly []float64
	for j := 4; j < n; j += 5 {
		weekly = append(weekly, closes[j])
	}
	// Always include the latest bar to represent the current (partial) week
	if (n-1)%5 != 4 {
		weekly = append(weekly, closes[n-1])
	}
	return weekly
}

// Run executes the backtest and returns full metrics.
func Run(ticker string, candles indicators.Candles, memEntries []memory.Entry, cfg Config) Result {
	closes := candles.C
	timestamps := candles.T
	n := len(closes)

	r := Result{Ticker: ticker, TotalBars: n}
	if n < minBarsNeeded+cfg.MaxHoldDays+2 {
		return r
	}

	barDate := func(i int) string {
		if i >= 0 && i < len(timestamps) {
			return time.Unix(timestamps[i], 0).UTC().Format("2006-01-02")
		}
		return fmt.Sprintf("bar%d", i)
	}

	r.StartDate = barDate(0)
	r.EndDate = barDate(n - 1)

	// Build SPY regime map if market candles provided
	var spyBull map[string]bool
	if cfg.MarketCandles != nil {
		spyBull = buildSPYRegimeMap(*cfg.MarketCandles)
	}

	// ── Indicator strategy simulation ────────────────────────────────

	capital := cfg.InitialCapital
	entryCapital := 0.0
	entryPrice := 0.0
	entryStopPrice := 0.0   // ATR-based stop price, set at trade entry
	peakPriceInTrade := 0.0 // highest price seen while in trade, for trailing stop
	entryBar := 0
	entryMaxHold := cfg.MaxHoldDays // effective hold limit set at trade entry
	inTrade := false
	peak := capital
	maxDD := 0.0
	var prevHist float64
	var ma50Buf []float64 // rolling MA50 values to derive prevMA50 (10 bars back)
	var trades []Trade
	equity := make([]float64, n)
	// Pre-fill warm-up bars with initial capital so max drawdown isn't falsely 100%
	// (bars 0..minBarsNeeded-1 are skipped by `continue` in the main loop, so they
	// would stay at zero — causing peak=capital, trough=0 → 100% drawdown).
	for i := range equity {
		equity[i] = capital
	}

	// Reserve enough bars at the end for the longest possible trade to close
	maxPossibleHold := int(float64(cfg.MaxHoldDays)*cfg.TrendExtendFactor*1.5) + 2
	if cfg.TrendExtendFactor <= 0 {
		maxPossibleHold = cfg.MaxHoldDays + 2
	}
	lastSignalBar := n - maxPossibleHold - 2

	for i := 1; i < n; i++ {
		// Mark equity to market
		if inTrade {
			equity[i] = entryCapital * (closes[i] / entryPrice)
		} else {
			equity[i] = equity[i-1]
		}

		// Update drawdown on current equity
		if equity[i] > peak {
			peak = equity[i]
		}
		if dd := (peak - equity[i]) / peak * 100; dd > maxDD {
			maxDD = dd
		}

		if i < minBarsNeeded || i > lastSignalBar {
			if i >= minBarsNeeded {
				sub := subCandles(candles, i+1)
				if ind := indicators.ComputeAll(sub); ind.MACD != nil && ind.MACD.Histogram != nil {
					prevHist = *ind.MACD.Histogram
					if ind.MA50 != nil {
						ma50Buf = append(ma50Buf, *ind.MA50)
					} else {
						ma50Buf = append(ma50Buf, 0)
					}
				}
			}
			continue
		}

		sub := subCandles(candles, i+1)
		ind := indicators.ComputeAll(sub)

		// prevMA50: MA50 value from 10 bars ago (tracks slope of the 50-day MA)
		var prevMA50 float64
		if len(ma50Buf) >= 10 {
			prevMA50 = ma50Buf[len(ma50Buf)-10]
		}

		// Weekly MACD histogram: resample closes to weekly, compute MACD
		var weeklyHist *float64
		wcloses := resampleWeekly(closes[:i+1])
		if wm := indicators.CalcMACD(wcloses); wm != nil && wm.Histogram != nil {
			weeklyHist = wm.Histogram
		}

		sig := signal(ind, closes[i], prevHist, prevMA50, weeklyHist)

		// SPY regime filter: block new BUY entries when market is in a downtrend
		if sig == "BUY" && spyBull != nil {
			date := barDate(i)
			if bull, ok := spyBull[date]; ok && !bull {
				sig = "HOLD"
			}
		}

		if !inTrade && sig == "BUY" {
			inTrade = true
			entryPrice = closes[i]
			entryCapital = equity[i]
			entryBar = i
			peakPriceInTrade = closes[i]
			entryMaxHold = tradeMaxHold(ind, cfg) // set hold limit based on trend at entry
			// ATR-based stop: exit if price drops more than N×ATR14 below entry
			entryStopPrice = 0
			if cfg.StopLossATRMultiple > 0 && ind.ATR14 != nil {
				entryStopPrice = entryPrice - cfg.StopLossATRMultiple**ind.ATR14
			}
		} else if inTrade {
			// Update peak price for trailing stop
			if closes[i] > peakPriceInTrade {
				peakPriceInTrade = closes[i]
			}

			holdDays := i - entryBar
			exitReason := ""

			// Trailing stop — kicks in once trade has gained 7%+ from entry (v9: lowered from 10%).
			if cfg.TrailingStopATRMultiple > 0 && ind.ATR14 != nil && holdDays >= cfg.MinHoldDays {
				profitFromEntry := (peakPriceInTrade - entryPrice) / entryPrice * 100
				if profitFromEntry >= 7.0 {
					trailingStop := peakPriceInTrade - cfg.TrailingStopATRMultiple**ind.ATR14
					if closes[i] < trailingStop {
						exitReason = "trail-stop"
					}
				}
			}

			// Entry stop-loss — ATR-based: exit if price fell below computed stop level
			if exitReason == "" && entryStopPrice > 0 && holdDays >= cfg.MinHoldDays && closes[i] < entryStopPrice {
				exitReason = "stop-loss"
			}
			// SELL signal — only respect after MinHoldDays
			if exitReason == "" && holdDays >= cfg.MinHoldDays && sig == "SELL" {
				exitReason = "signal"
			}
			// Timeout — use the dynamic hold limit set at entry
			if exitReason == "" && holdDays >= entryMaxHold {
				exitReason = "timeout"
			}
			if exitReason != "" {
				retPct := (closes[i] - entryPrice) / entryPrice * 100
				// Realise P&L: equity[i] already reflects this via mark-to-market
				capital = equity[i]
				trades = append(trades, Trade{
					EntryDate:  barDate(entryBar),
					ExitDate:   barDate(i),
					EntryPrice: entryPrice,
					ExitPrice:  closes[i],
					ReturnPct:  round2(retPct),
					HoldDays:   holdDays,
					ExitReason: exitReason,
				})
				inTrade = false
				entryPrice = 0
				entryCapital = 0
			}
		}

		if ind.MACD != nil && ind.MACD.Histogram != nil {
			prevHist = *ind.MACD.Histogram
		}
		if ind.MA50 != nil {
			ma50Buf = append(ma50Buf, *ind.MA50)
		} else {
			ma50Buf = append(ma50Buf, 0)
		}
	}

	// Close any open position at end of data

	if inTrade && n > 0 {
		exitPrice := closes[n-1]
		retPct := (exitPrice - entryPrice) / entryPrice * 100
		capital = entryCapital * (exitPrice / entryPrice)
		trades = append(trades, Trade{
			EntryDate:  barDate(entryBar),
			ExitDate:   barDate(n - 1),
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			ReturnPct:  round2(retPct),
			HoldDays:   n - 1 - entryBar,
			ExitReason: "end-of-data",
		})
	}

	// ── Aggregate trade metrics ───────────────────────────────────────
	r.Trades = trades
	r.TotalTrades = len(trades)
	var winSum, lossSum float64
	for _, t := range trades {
		if t.ReturnPct > 0 {
			r.WinningTrades++
			winSum += t.ReturnPct
		} else {
			r.LosingTrades++
			lossSum += t.ReturnPct
		}
	}
	if r.TotalTrades > 0 {
		r.WinRate = round2(float64(r.WinningTrades) / float64(r.TotalTrades) * 100)
	}
	if r.WinningTrades > 0 {
		r.AvgWinPct = round2(winSum / float64(r.WinningTrades))
	}
	if r.LosingTrades > 0 {
		r.AvgLossPct = round2(lossSum / float64(r.LosingTrades))
	}
	r.TotalReturn = round2((capital - cfg.InitialCapital) / cfg.InitialCapital * 100)
	r.MaxDrawdownPct = round2(maxDD)
	if n > 1 {
		r.BuyHoldReturn = round2((closes[n-1] - closes[0]) / closes[0] * 100)
	}
	r.SharpeRatio = round2(sharpe(equity))

	// ── Memory / Claude accuracy evaluation ──────────────────────────
	if len(memEntries) > 0 {
		r.Memory = evaluateMemory(memEntries, candles)
	}

	return r
}

// evaluateMemory checks each stored Claude decision against actual forward prices.
func evaluateMemory(entries []memory.Entry, candles indicators.Candles) *MemoryEval {
	// Build date → bar index map (YYYY-MM-DD → i)
	dateIndex := make(map[string]int, len(candles.T))
	for i, ts := range candles.T {
		day := time.Unix(ts, 0).UTC().Format("2006-01-02")
		dateIndex[day] = i
	}

	eval := &MemoryEval{
		ByDecision: map[string]*DecisionStat{
			"BUY":  {},
			"SELL": {},
			"HOLD": {},
		},
	}

	var buyReturns []float64

	for _, e := range entries {
		day := e.Date
		if len(day) >= 10 {
			day = day[:10]
		}
		barIdx, ok := dateIndex[day]
		if !ok {
			continue
		}
		fwdBar := barIdx + 5
		if fwdBar >= len(candles.C) {
			continue
		}

		entryPrice := candles.C[barIdx]
		fwdPrice := candles.C[fwdBar]
		retPct := (fwdPrice - entryPrice) / entryPrice * 100

		var correct bool
		switch e.Decision {
		case "BUY":
			correct = fwdPrice > entryPrice
			buyReturns = append(buyReturns, retPct)
		case "SELL":
			correct = fwdPrice < entryPrice
		case "HOLD":
			correct = math.Abs(retPct) <= 3.0 // ±3% accounts for normal daily noise
		}

		stat, exists := eval.ByDecision[e.Decision]
		if !exists {
			stat = &DecisionStat{}
			eval.ByDecision[e.Decision] = stat
		}
		stat.Total++
		if correct {
			stat.Correct++
		}

		eval.Total++
		if correct {
			eval.Correct++
		}

		eval.Details = append(eval.Details, MemoryDecision{
			Date:      day,
			Decision:  e.Decision,
			Price:     entryPrice,
			Price5d:   fwdPrice,
			ReturnPct: round2(retPct),
			Correct:   correct,
		})
	}

	if eval.Total > 0 {
		eval.AccuracyPct = round2(float64(eval.Correct) / float64(eval.Total) * 100)
	}
	if len(buyReturns) > 0 {
		eval.AvgReturn5d = round2(avg(buyReturns))
	}
	return eval
}

// Print renders the result to stdout in a readable format.
func Print(r Result) {
	sep := strings.Repeat("─", 58)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  BACKTEST  %-6s  %s → %s  (%d bars)\n", r.Ticker, r.StartDate, r.EndDate, r.TotalBars)
	fmt.Printf("%s\n", sep)

	// Strategy vs buy-and-hold
	stratSign := "+"
	if r.TotalReturn < 0 {
		stratSign = ""
	}
	bhSign := "+"
	if r.BuyHoldReturn < 0 {
		bhSign = ""
	}
	fmt.Printf("  Strategy return:   %s%.2f%%\n", stratSign, r.TotalReturn)
	fmt.Printf("  Buy-and-hold:      %s%.2f%%\n", bhSign, r.BuyHoldReturn)
	alpha := r.TotalReturn - r.BuyHoldReturn
	alphaSign := "+"
	if alpha < 0 {
		alphaSign = ""
	}
	fmt.Printf("  Alpha:             %s%.2f%%\n", alphaSign, alpha)
	fmt.Println()

	// Trade stats
	fmt.Printf("  Trades:            %d  (W:%d / L:%d)\n", r.TotalTrades, r.WinningTrades, r.LosingTrades)
	fmt.Printf("  Win rate:          %.1f%%\n", r.WinRate)
	fmt.Printf("  Avg win:           +%.2f%%\n", r.AvgWinPct)
	fmt.Printf("  Avg loss:          %.2f%%\n", r.AvgLossPct)
	fmt.Printf("  Max drawdown:      %.2f%%\n", r.MaxDrawdownPct)
	fmt.Printf("  Sharpe ratio:      %.2f\n", r.SharpeRatio)
	fmt.Println()

	// Individual trades
	if len(r.Trades) > 0 {
		fmt.Printf("  %-12s %-12s %-8s %-8s %-8s %-4s  %s\n",
			"Entry", "Exit", "Buy@", "Sell@", "Return", "Days", "Exit")
		for _, t := range r.Trades {
			retSign := "+"
			if t.ReturnPct < 0 {
				retSign = ""
			}
			fmt.Printf("  %-12s %-12s $%-7.2f $%-7.2f %s%-6.2f%% %-4d  %s\n",
				t.EntryDate, t.ExitDate,
				t.EntryPrice, t.ExitPrice,
				retSign, t.ReturnPct,
				t.HoldDays, t.ExitReason)
		}
		fmt.Println()
	}

	// Claude memory accuracy
	if r.Memory != nil && r.Memory.Total > 0 {
		m := r.Memory
		fmt.Printf("%s\n", sep)
		fmt.Printf("  CLAUDE DECISION ACCURACY  (%d decisions evaluated)\n", m.Total)
		fmt.Printf("%s\n", sep)
		fmt.Printf("  Overall accuracy:  %.1f%%  (%d/%d correct)\n", m.AccuracyPct, m.Correct, m.Total)
		if m.AvgReturn5d != 0 {
			avgSign := "+"
			if m.AvgReturn5d < 0 {
				avgSign = ""
			}
			fmt.Printf("  Avg return 5d after BUY:  %s%.2f%%\n", avgSign, m.AvgReturn5d)
		}
		fmt.Println()
		for _, dec := range []string{"BUY", "SELL", "HOLD"} {
			s := m.ByDecision[dec]
			if s == nil || s.Total == 0 {
				continue
			}
			pct := float64(s.Correct) / float64(s.Total) * 100
			fmt.Printf("  %-4s  %3d decisions  →  %.1f%% correct\n", dec, s.Total, pct)
		}
		fmt.Println()
	}

	fmt.Printf("%s\n\n", sep)
}

// ── helpers ──────────────────────────────────────────────────────────

func subCandles(c indicators.Candles, n int) indicators.Candles {
	if n > len(c.C) {
		n = len(c.C)
	}
	return indicators.Candles{
		T: c.T[:n],
		O: c.O[:n],
		H: c.H[:n],
		L: c.L[:n],
		C: c.C[:n],
		V: c.V[:n],
		S: c.S,
	}
}

func sharpe(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	returns := make([]float64, 0, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1] == 0 {
			continue
		}
		returns = append(returns, (equity[i]-equity[i-1])/equity[i-1])
	}
	if len(returns) == 0 {
		return 0
	}
	mean := avg(returns)
	rfr := 0.05 / 252 // 5% annual risk-free rate
	excess := mean - rfr
	variance := 0.0
	for _, r := range returns {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(returns))
	stddev := math.Sqrt(variance)
	if stddev == 0 {
		return 0
	}
	return excess / stddev * math.Sqrt(252)
}

func avg(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	return sum / float64(len(xs))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
