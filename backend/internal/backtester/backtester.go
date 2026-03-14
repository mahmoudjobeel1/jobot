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
	TrailingStopATRMultiple float64 // once trade up 7%+, trail stop at peak - N×ATR14 (0 = disabled)
	TrendExtendFactor       float64 // multiplier on MaxHoldDays when Trend60d is strong (0 = disabled)
	InitialCapital          float64
	MarketCandles           *indicators.Candles // SPY candles for market regime filter (nil = disabled)
	SignalThreshold         int                 // bull/bear score required to fire a signal (0 = default 6)
	FastReentryThreshold    int     // lower score threshold after trail-stop exit in uptrend (0 = default 4)
	FastReentryWindowBars   int     // bars after trail-stop where fast re-entry applies (0 = default 20)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxHoldDays:             10,
		MinHoldDays:             2,
		StopLossATRMultiple:     2.5,
		TrailingStopATRMultiple: 1.5,
		TrendExtendFactor:       2.0,
		InitialCapital:          10_000,
	}
}

// tradeMaxHold computes the effective max hold days for a new trade.
// In trending regime, the hold period is extended so trending stocks have room to run.
// Note: once the trailing stop activates, timeout is disabled entirely — so this cap
// only matters for trades that never reach the 7% profit level.
func tradeMaxHold(ind indicators.Indicators, cfg Config) int {
	if cfg.TrendExtendFactor <= 0 || ind.Trend60d == nil {
		return cfg.MaxHoldDays
	}
	trend := *ind.Trend60d
	switch {
	case trend > 20:
		return int(float64(cfg.MaxHoldDays) * cfg.TrendExtendFactor * 1.5)
	case trend > 10:
		return int(float64(cfg.MaxHoldDays) * cfg.TrendExtendFactor)
	default:
		return cfg.MaxHoldDays
	}
}

// signalSizeFactor returns the fraction of available capital to deploy based on signal strength.
// High-conviction (score ≥8) gets full capital; marginal entries get less.
func signalSizeFactor(score int) float64 {
	switch {
	case score >= 8:
		return 1.00
	case score >= 7:
		return 0.85
	default: // score == 6
		return 0.65
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
	ExitReason string // "signal" | "timeout" | "trail-stop" | "stop-loss" | "end-of-data"
}

// MemoryDecision evaluates a single stored Claude decision against actual forward price.
type MemoryDecision struct {
	Date      string
	Decision  string
	Price     float64
	Price5d   float64
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
	AvgReturn5d float64
	ByDecision  map[string]*DecisionStat
	Details     []MemoryDecision
}

// Result is the full output of one backtest run.
type Result struct {
	Ticker        string
	StartDate     string
	EndDate       string
	TotalBars     int
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	WinRate       float64
	TotalReturn   float64
	BuyHoldReturn float64
	AvgWinPct     float64
	AvgLossPct    float64
	MaxDrawdownPct float64
	SharpeRatio   float64
	Trades        []Trade
	Memory        *MemoryEval
}

// signal returns "BUY"/"SELL"/"HOLD" and the winning score using a multi-factor scoring model.
//
// bull/bear score components:
//
//	RSI extremes          ±1..3
//	MACD histogram cross  ±3  (strongest signal)
//	MACD histogram trend  ±1  (building momentum)
//	Price vs MA50         ±1  (trend filter)
//	Price vs MA20         ±1  (short-term trend)
//	60d trend             ±1  (broader momentum)
//	MA50 slope            ±2  (regime: rising vs declining MA50)
//	Price vs MA200        ±2  (long-term trend regime)
//
//	Threshold: configurable (default 6)
//
//	Additional gates:
//	  Volume filter:      CurVol < 70% AvgVol → no BUY
//	  Weekly MACD:        weekly histogram < 0 → no BUY (bypassable for fast re-entry)
//
// Hard gate: MA50 declining AND price below MA50 → skip to bear-only scoring.
func signal(ind indicators.Indicators, currentClose, prevHist, prevMA50 float64, weeklyHist *float64, threshold int, bypassWeeklyGate bool) (string, int) {
	if ind.RSI == nil || ind.MACD == nil || ind.MACD.Histogram == nil {
		return "HOLD", 0
	}
	rsi := *ind.RSI
	hist := *ind.MACD.Histogram

	macdCrossedUp := prevHist <= 0 && hist > 0
	macdCrossedDown := prevHist >= 0 && hist < 0
	histRising := hist > prevHist
	histFalling := hist < prevHist

	// Hard gate: block BUY when MA50 is declining AND price is below it.
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
		if ind.MA200 != nil {
			if currentClose > *ind.MA200 {
				bull += 2
			} else {
				bear += 2
			}
		}

		if bull >= threshold && bull > bear {
			// Volume filter: skip thin-volume days
			if ind.CurVol != nil && ind.AvgVol != nil {
				if *ind.CurVol < 0.7*float64(*ind.AvgVol) {
					return "HOLD", 0
				}
			}
			// Weekly MACD confirmation (bypassable for fast re-entry after trail-stop in uptrend)
			if !bypassWeeklyGate && weeklyHist != nil && *weeklyHist < 0 {
				return "HOLD", 0
			}
			return "BUY", bull
		}
		if bear >= threshold && bear > bull {
			return "SELL", bear
		}
		return "HOLD", 0
	}

computeSell:
	{
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
			return "SELL", bear
		}
		return "HOLD", 0
	}
}

// buildSPYRegimeMap returns a date → isBullish map. A day is bullish when SPY close > SPY MA50.
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

// resampleWeekly converts daily closes to weekly (every 5th bar, always including the last).
func resampleWeekly(closes []float64) []float64 {
	n := len(closes)
	if n == 0 {
		return nil
	}
	var weekly []float64
	for j := 4; j < n; j += 5 {
		weekly = append(weekly, closes[j])
	}
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

	// Resolve configurable parameters
	sigThreshold := cfg.SignalThreshold
	if sigThreshold <= 0 {
		sigThreshold = 6
	}
	fastThreshold := cfg.FastReentryThreshold
	if fastThreshold <= 0 {
		fastThreshold = 4
	}
	fastWindow := cfg.FastReentryWindowBars
	if fastWindow <= 0 {
		fastWindow = 20
	}

	// Build SPY regime map if market candles provided
	var spyBull map[string]bool
	if cfg.MarketCandles != nil {
		spyBull = buildSPYRegimeMap(*cfg.MarketCandles)
	}

	// ── Indicator strategy simulation ────────────────────────────────

	capital := cfg.InitialCapital
	// Trade state
	inTrade := false
	entryPrice := 0.0
	entryCapital := 0.0  // portion of equity deployed in trade
	cashReserve := 0.0   // portion held as cash during trade (position sizing)
	entryStopPrice := 0.0
	peakPriceInTrade := 0.0
	trailingStopActive := false // once true, timeout is disabled
	entryBar := 0
	entryMaxHold := cfg.MaxHoldDays

	// Re-entry tracking
	lastExitReason := ""
	lastExitBar := -999

	peak := capital
	maxDD := 0.0
	var prevHist float64
	var ma50Buf []float64
	var trades []Trade

	equity := make([]float64, n)
	for i := range equity {
		equity[i] = capital
	}

	maxPossibleHold := int(float64(cfg.MaxHoldDays)*cfg.TrendExtendFactor*1.5) + 2
	if cfg.TrendExtendFactor <= 0 {
		maxPossibleHold = cfg.MaxHoldDays + 2
	}
	lastSignalBar := n - maxPossibleHold - 2

	for i := 1; i < n; i++ {
		// ── Mark equity to market ──────────────────────────────────────
		if inTrade {
			// position mark-to-market + held cash
			equity[i] = cashReserve + entryCapital*(closes[i]/entryPrice)
		} else {
			equity[i] = equity[i-1]
		}

		// Update drawdown
		if equity[i] > peak {
			peak = equity[i]
		}
		if dd := (peak - equity[i]) / peak * 100; dd > maxDD {
			maxDD = dd
		}

		// Warm-up and post-signal-range: update prevHist/ma50Buf but skip trade logic
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

		var prevMA50 float64
		if len(ma50Buf) >= 10 {
			prevMA50 = ma50Buf[len(ma50Buf)-10]
		}

		var weeklyHist *float64
		wcloses := resampleWeekly(closes[:i+1])
		if wm := indicators.CalcMACD(wcloses); wm != nil && wm.Histogram != nil {
			weeklyHist = wm.Histogram
		}

		// Determine effective threshold — fast re-entry after trail-stop in uptrend.
		// Requires: within fastWindow bars, MA50 rising, price > MA200, AND price made a new 5-bar high
		// (price at a local high, not in a topping/exhaustion pattern).
		effectiveThreshold := sigThreshold
		bypassWeeklyGate := false
		if lastExitReason == "trail-stop" && i-lastExitBar <= fastWindow {
			if ind.MA50 != nil && prevMA50 > 0 && *ind.MA50 > prevMA50 &&
				ind.MA200 != nil && closes[i] > *ind.MA200 {
				// Require price to have made a new 5-bar high
				lookback := 5
				if i >= lookback {
					recent5High := closes[i-lookback]
					for k := i - lookback + 1; k < i; k++ {
						if closes[k] > recent5High {
							recent5High = closes[k]
						}
					}
					if closes[i] >= recent5High {
						effectiveThreshold = fastThreshold
						bypassWeeklyGate = true
					}
				}
			}
		}

		sig, score := signal(ind, closes[i], prevHist, prevMA50, weeklyHist, effectiveThreshold, bypassWeeklyGate)

		// SPY regime filter: block new BUY entries when market is in a downtrend
		if sig == "BUY" && spyBull != nil {
			date := barDate(i)
			if bull, ok := spyBull[date]; ok && !bull {
				sig = "HOLD"
				score = 0
			}
		}

		// ── Entry ──────────────────────────────────────────────────────
		if !inTrade && sig == "BUY" {
			totalAvail := equity[i]
			sizeFactor := signalSizeFactor(score)
			entryCapital = totalAvail * sizeFactor
			cashReserve = totalAvail - entryCapital

			inTrade = true
			entryPrice = closes[i]
			entryBar = i
			peakPriceInTrade = closes[i]
			trailingStopActive = false
			entryMaxHold = tradeMaxHold(ind, cfg)
			entryStopPrice = 0
			if cfg.StopLossATRMultiple > 0 && ind.ATR14 != nil {
				entryStopPrice = entryPrice - cfg.StopLossATRMultiple**ind.ATR14
			}

		} else if inTrade {
			// Update trailing peak
			if closes[i] > peakPriceInTrade {
				peakPriceInTrade = closes[i]
			}

			holdDays := i - entryBar
			exitReason := ""

			// Trailing stop — activates once trade is up 7%+ from entry
			if cfg.TrailingStopATRMultiple > 0 && ind.ATR14 != nil && holdDays >= cfg.MinHoldDays {
				profitFromEntry := (peakPriceInTrade - entryPrice) / entryPrice * 100
				if profitFromEntry >= 7.0 {
					trailingStopActive = true
					trailingStop := peakPriceInTrade - cfg.TrailingStopATRMultiple**ind.ATR14
					if closes[i] < trailingStop {
						exitReason = "trail-stop"
					}
				}
			}

			// Entry stop-loss
			if exitReason == "" && entryStopPrice > 0 && holdDays >= cfg.MinHoldDays && closes[i] < entryStopPrice {
				exitReason = "stop-loss"
			}
			// SELL signal
			if exitReason == "" && holdDays >= cfg.MinHoldDays && sig == "SELL" {
				exitReason = "signal"
			}
			// Timeout — ONLY fires if trailing stop has never activated.
			// Once the trailing stop is active, we let winners run until the trail catches them.
			if exitReason == "" && !trailingStopActive && holdDays >= entryMaxHold {
				exitReason = "timeout"
			}

			if exitReason != "" {
				retPct := (closes[i] - entryPrice) / entryPrice * 100
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
				lastExitReason = exitReason
				lastExitBar = i
				entryPrice, entryCapital, cashReserve = 0, 0, 0
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
		capital = equity[n-1]
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

	if len(memEntries) > 0 {
		r.Memory = evaluateMemory(memEntries, candles)
	}

	return r
}

// evaluateMemory checks each stored Claude decision against actual forward prices.
func evaluateMemory(entries []memory.Entry, candles indicators.Candles) *MemoryEval {
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
			correct = math.Abs(retPct) <= 3.0
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

	fmt.Printf("  Trades:            %d  (W:%d / L:%d)\n", r.TotalTrades, r.WinningTrades, r.LosingTrades)
	fmt.Printf("  Win rate:          %.1f%%\n", r.WinRate)
	fmt.Printf("  Avg win:           +%.2f%%\n", r.AvgWinPct)
	fmt.Printf("  Avg loss:          %.2f%%\n", r.AvgLossPct)
	fmt.Printf("  Max drawdown:      %.2f%%\n", r.MaxDrawdownPct)
	fmt.Printf("  Sharpe ratio:      %.2f\n", r.SharpeRatio)
	fmt.Println()

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
	rfr := 0.05 / 252
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
