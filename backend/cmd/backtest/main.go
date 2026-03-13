// backtest runs an indicator-based simulation and evaluates stored Claude decision accuracy.
//
// Usage:
//
//	go run ./cmd/backtest -ticker AAPL
//	go run ./cmd/backtest -all
//	go run ./cmd/backtest -ticker NVDA -months 24 -hold 10 -capital 10000
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"jobot/internal/backtester"
	"jobot/internal/config"
	"jobot/internal/finnhub"
	"jobot/internal/memory"
)

func main() {
	_ = godotenv.Load()

	tickerFlag := flag.String("ticker", "", "Ticker to backtest (e.g. AAPL)")
	allFlag := flag.Bool("all", false, "Backtest all portfolio tickers")
	months := flag.Int("months", 24, "Months of historical data to fetch (default: 24)")
	hold := flag.Int("hold", 10, "Max hold days before forced exit (default: 10)")
	capital := flag.Float64("capital", 10_000, "Initial capital in USD (default: 10000)")
	flag.Parse()

	if !*allFlag && *tickerFlag == "" {
		fmt.Fprintln(os.Stderr, "Usage: backtest -ticker AAPL  or  backtest -all")
		os.Exit(1)
	}

	tickers := config.Tickers
	if !*allFlag {
		tickers = []string{strings.ToUpper(*tickerFlag)}
	}

	cfg := backtester.Config{
		MaxHoldDays:    *hold,
		InitialCapital: *capital,
	}

	fmt.Printf("\n  Backtest  |  %d months of data  |  max hold %d days  |  capital $%.0f\n",
		*months, *hold, *capital)

	for i, ticker := range tickers {
		fmt.Printf("\n  [%d/%d] Fetching %s...\n", i+1, len(tickers), ticker)

		candles, err := finnhub.FetchCandlesMonths(ticker, *months)
		if err != nil {
			fmt.Printf("  [ERROR] %s: %v\n", ticker, err)
			continue
		}

		entries, _ := memory.LoadMemory(ticker)

		result := backtester.Run(ticker, candles, entries, cfg)
		backtester.Print(result)

		if i < len(tickers)-1 {
			time.Sleep(800 * time.Millisecond)
		}
	}
}
