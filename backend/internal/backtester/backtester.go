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
	MaxHoldDays    int
	InitialCapital float64
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{MaxHoldDays: 10, InitialCapital: 10_000}
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

// signal returns "BUY", "SELL", or "HOLD" for the current bar.
// prevHist is the MACD histogram from the previous bar (0 if first bar).
func signal(ind indicators.Indicators, prevHist float64) string {
	if ind.RSI == nil || ind.MACD == nil || ind.MACD.Histogram == nil {
		return "HOLD"
	}
	rsi := *ind.RSI
	hist := *ind.MACD.Histogram

	macdCrossedUp := prevHist <= 0 && hist > 0
	macdCrossedDown := prevHist >= 0 && hist < 0

	// Strongly oversold
	if rsi < 35 {
		return "BUY"
	}
	// Recovering from oversold with MACD momentum turning positive
	if rsi < 50 && macdCrossedUp {
		return "BUY"
	}
	// Strongly overbought
	if rsi > 75 {
		return "SELL"
	}
	// Overextended with MACD momentum turning negative
	if rsi > 65 && macdCrossedDown {
		return "SELL"
	}
	return "HOLD"
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

	// ── Indicator strategy simulation ────────────────────────────────

	capital := cfg.InitialCapital
	entryCapital := 0.0
	entryPrice := 0.0
	entryBar := 0
	inTrade := false
	peak := capital
	maxDD := 0.0
	var prevHist float64
	var trades []Trade
	equity := make([]float64, n)
	equity[0] = capital

	// Stop generating new signals early enough that all trades can complete
	lastSignalBar := n - cfg.MaxHoldDays - 2

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
			// Still update prevHist even outside signal range
			if i >= minBarsNeeded {
				sub := subCandles(candles, i+1)
				if ind := indicators.ComputeAll(sub); ind.MACD != nil && ind.MACD.Histogram != nil {
					prevHist = *ind.MACD.Histogram
				}
			}
			continue
		}

		sub := subCandles(candles, i+1)
		ind := indicators.ComputeAll(sub)
		sig := signal(ind, prevHist)

		if !inTrade && sig == "BUY" {
			inTrade = true
			entryPrice = closes[i]
			entryCapital = equity[i]
			entryBar = i
		} else if inTrade {
			holdDays := i - entryBar
			exitReason := ""
			if sig == "SELL" {
				exitReason = "signal"
			} else if holdDays >= cfg.MaxHoldDays {
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
			correct = math.Abs(retPct) <= 2.0
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
