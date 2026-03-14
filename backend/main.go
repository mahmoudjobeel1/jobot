package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"jobot/internal/analyst"
	"jobot/internal/api"
	"jobot/internal/config"
	"jobot/internal/finnhub"
	"jobot/internal/notifier"
	"jobot/internal/portfolio"
	"jobot/internal/store"
)

// isMarketOpen returns true if the current time is within NYSE market hours
// (Monday–Friday, 09:30–16:00 Eastern Time).
func isMarketOpen() bool {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return true // assume open if we can't load tz
	}
	et := time.Now().In(loc)
	weekday := et.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	hm := et.Hour()*100 + et.Minute()
	return hm >= 930 && hm < 1600
}

type tickerResult struct {
	ticker string
	ok     bool
	errMsg string
	price  float64
}

func processTicker(ticker string) tickerResult {
	data, err := finnhub.FetchTickerData(ticker)
	if err != nil {
		fmt.Printf("  [ERROR] %s: %v\n", ticker, err)
		return tickerResult{ticker: ticker, ok: false, errMsg: err.Error()}
	}

	// ── Check and update all trailing stops for this ticker ──────────
	currentPrice := data.Quote.C
	for _, s := range store.GetStops() {
		if !s.IsTrailing || !strings.EqualFold(s.Ticker, ticker) {
			continue
		}
		triggered, err := store.UpdateTrailingStop(s.ID, currentPrice)
		if err != nil {
			fmt.Printf("  [Stop] %s update error: %v\n", ticker, err)
			continue
		}
		if triggered {
			fmt.Printf("  [Stop] TRAILING STOP TRIGGERED — %s at $%.2f (stop $%.2f, peak $%.2f)\n",
				ticker, currentPrice, s.StopPrice, s.PeakPrice)
			if err := store.ExecuteStop(s.ID); err != nil {
				fmt.Printf("  [Stop] Execute error for %s: %v\n", ticker, err)
			}
		} else if s.Activated {
			fmt.Printf("  [Stop] %s trailing stop: price $%.2f | stop $%.2f | peak $%.2f\n",
				ticker, currentPrice, s.StopPrice, s.PeakPrice)
		}
	}

	result, err := analyst.AnalyzeStock(ticker, data.Quote, data.Candles, data.News)
	if err != nil {
		fmt.Printf("  [ERROR] %s: %v\n", ticker, err)
		return tickerResult{ticker: ticker, ok: false, errMsg: err.Error()}
	}
	notifier.Notify(result)
	return tickerResult{ticker: ticker, ok: true, price: data.Quote.C}
}

func printPortfolioSummary(prices map[string]float64) {
	s := portfolio.Summary(prices)
	plSign := "+"
	if s.TotalPL < 0 {
		plSign = ""
	}
	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════════╗")
	fmt.Println("  ║           PORTFOLIO SUMMARY               ║")
	fmt.Println("  ╠═══════════════════════════════════════════╣")
	fmt.Printf("  ║  Cost Basis:    $%12.2f              ║\n", s.TotalCostBasis)
	fmt.Printf("  ║  Market Value:  $%12.2f              ║\n", s.TotalMarketValue)
	fmt.Printf("  ║  Total P&L:     %s$%10.2f (%s%.2f%%)  ║\n",
		plSign, math.Abs(s.TotalPL), plSign, math.Abs(s.TotalPLPct))
	fmt.Println("  ╚═══════════════════════════════════════════╝")
}

func runWeeklySummary() {
	fmt.Println("\n  [Weekly] Running multi-session reviews...")
	for _, ticker := range config.Tickers {
		if err := analyst.AnalyzeWeekly(ticker); err != nil {
			fmt.Printf("  [Weekly] %s: %v\n", ticker, err)
		} else {
			fmt.Printf("  [Weekly] %s: done\n", ticker)
		}
		time.Sleep(800 * time.Millisecond)
	}
	fmt.Println("  [Weekly] All reviews complete.")
}

func runCycle() {
	sep := strings.Repeat("═", 60)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  STOCK AI AGENT — %s\n", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	fmt.Printf("  Portfolio: %s\n", strings.Join(config.Tickers, ", "))
	fmt.Printf("%s\n", sep)

	// Print holdings table
	fmt.Println("  Holdings:")
	for _, h := range portfolio.GetHoldings() {
		fmt.Printf("    %-6s  %6.4g shares @ $%.2f  (cost basis: $%.2f)\n",
			h.Ticker, h.Qty, h.AvgCost, h.CostBasis())
	}
	fmt.Println()

	if os.Getenv("MARKET_HOURS_ONLY") == "true" && !isMarketOpen() {
		fmt.Println("  [Scheduler] Market is closed — skipping.")
		return
	}

	prices := make(map[string]float64)
	var results []tickerResult
	for i, ticker := range config.Tickers {
		r := processTicker(ticker)
		results = append(results, r)
		if r.ok {
			prices[r.ticker] = r.price
		}
		if i < len(config.Tickers)-1 {
			time.Sleep(1200 * time.Millisecond)
		}
	}

	var ok int
	var failed []tickerResult
	for _, r := range results {
		if r.ok {
			ok++
		} else {
			failed = append(failed, r)
		}
	}

	// Print portfolio summary if we got at least one price
	if len(prices) > 0 {
		printPortfolioSummary(prices)
	}

	fmt.Printf("\n  Cycle complete — %d/%d succeeded\n", ok, len(config.Tickers))
	if len(failed) > 0 {
		msgs := make([]string, 0, len(failed))
		for _, f := range failed {
			msgs = append(msgs, fmt.Sprintf("%s (%s)", f.ticker, f.errMsg))
		}
		fmt.Printf("  Failed: %s\n", strings.Join(msgs, ", "))
	}
	fmt.Printf("%s\n\n", sep)
}

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	schedule := os.Getenv("CRON_SCHEDULE")
	if schedule == "" {
		schedule = "*/15 9-16 * * 1-5"
	}

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║          STOCK AI AGENT — Starting up                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Printf("  Schedule:  %s\n", schedule)
	fmt.Printf("  Portfolio: %s\n\n", strings.Join(config.Tickers, ", "))

	// Initialise store (loads data/portfolio.json or seeds from Holdings)
	seed := make([]store.Holding, len(portfolio.Holdings))
	for i, h := range portfolio.Holdings {
		seed[i] = store.Holding{Ticker: h.Ticker, Qty: h.Qty, AvgCost: h.AvgCost}
	}
	if err := store.Init(seed); err != nil {
		fmt.Fprintf(os.Stderr, "  store init error: %v\n", err)
		os.Exit(1)
	}

	// Start REST API server
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	api.StartServer(":" + apiPort)

	if os.Getenv("FINNHUB_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "  FINNHUB_API_KEY missing")
		os.Exit(1)
	}
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Fprintln(os.Stderr, "  ANTHROPIC_API_KEY missing")
		os.Exit(1)
	}

	if os.Getenv("RUN_ON_START") != "false" {
		fmt.Println("  Running initial cycle on startup...")
		runCycle()
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Could not load America/New_York timezone: %v\n", err)
		os.Exit(1)
	}

	weeklySchedule := os.Getenv("WEEKLY_CRON")
	if weeklySchedule == "" {
		weeklySchedule = "0 16 * * 5" // Friday 4pm ET
	}

	c := cron.New(cron.WithLocation(loc))
	if _, err := c.AddFunc(schedule, runCycle); err != nil {
		fmt.Fprintf(os.Stderr, "  Invalid cron schedule %q: %v\n", schedule, err)
		os.Exit(1)
	}
	if _, err := c.AddFunc(weeklySchedule, runWeeklySummary); err != nil {
		fmt.Fprintf(os.Stderr, "  Invalid weekly cron %q: %v\n", weeklySchedule, err)
		os.Exit(1)
	}
	c.Start()
	fmt.Printf("  Agent is running. Cron: %s | Weekly: %s\n\n", schedule, weeklySchedule)

	// Block forever
	select {}
}