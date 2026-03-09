package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"jobot/internal/analyst"
	"jobot/internal/config"
	"jobot/internal/finnhub"
	"jobot/internal/notifier"
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
}

func processTicker(ticker string) tickerResult {
	data, err := finnhub.FetchTickerData(ticker)
	if err != nil {
		fmt.Printf("  [ERROR] %s: %v\n", ticker, err)
		return tickerResult{ticker: ticker, ok: false, errMsg: err.Error()}
	}
	result, err := analyst.AnalyzeStock(ticker, data.Quote, data.Candles, data.News)
	if err != nil {
		fmt.Printf("  [ERROR] %s: %v\n", ticker, err)
		return tickerResult{ticker: ticker, ok: false, errMsg: err.Error()}
	}
	notifier.Notify(result)
	return tickerResult{ticker: ticker, ok: true}
}

func runCycle() {
	sep := strings.Repeat("═", 60)
	fmt.Printf("\n%s\n", sep)
	fmt.Printf("  STOCK AI AGENT — %s\n", time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	fmt.Printf("  Tickers: %s\n", strings.Join(config.Tickers, ", "))
	fmt.Printf("%s\n", sep)

	if os.Getenv("MARKET_HOURS_ONLY") == "true" && !isMarketOpen() {
		fmt.Println("  [Scheduler] Market is closed — skipping.")
		return
	}

	var results []tickerResult
	for i, ticker := range config.Tickers {
		r := processTicker(ticker)
		results = append(results, r)
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
	fmt.Printf("  Tickers:   %s\n\n", strings.Join(config.Tickers, ", "))

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

	c := cron.New(cron.WithLocation(loc))
	if _, err := c.AddFunc(schedule, runCycle); err != nil {
		fmt.Fprintf(os.Stderr, "  Invalid cron schedule %q: %v\n", schedule, err)
		os.Exit(1)
	}
	c.Start()
	fmt.Printf("  Agent is running. Cron: %s\n\n", schedule)

	// Block forever
	select {}
}
