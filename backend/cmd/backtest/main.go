// backtest runs an indicator-based simulation and evaluates stored Claude decision accuracy.
//
// Usage:
//
//	go run ./cmd/backtest -ticker AAPL
//	go run ./cmd/backtest -all
//	go run ./cmd/backtest -ticker NVDA -months 24 -hold 10 -minhold 2 -capital 10000
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
	months   := flag.Int("months", 24, "Months of historical data to fetch (default: 24)")
	hold     := flag.Int("hold", 10, "Max hold days before forced exit (default: 10)")
	minhold  := flag.Int("minhold", 2, "Min hold days before a SELL signal is respected (default: 2)")
	atrMult   := flag.Float64("atr", 2.5, "Stop-loss ATR multiple: exit if price drops N×ATR14 from entry, 0=off (default: 2.5)")
	trail     := flag.Float64("trail", 1.5, "Trailing stop ATR multiple: once up 7%+, trail at peak-N×ATR14, 0=off (default: 1.5)")
	extend    := flag.Float64("extend", 2.0, "Hold multiplier when 60d trend is strong, 0=off (default: 2.0)")
	capital   := flag.Float64("capital", 10_000, "Initial capital in USD (default: 10000)")
	nospy     := flag.Bool("nospy", false, "Disable SPY regime filter")
	threshold := flag.Int("threshold", 0, "Bull/bear score threshold to fire a signal (default: 6, 0=use default)")
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
		MaxHoldDays:             *hold,
		MinHoldDays:             *minhold,
		StopLossATRMultiple:     *atrMult,
		TrailingStopATRMultiple: *trail,
		TrendExtendFactor:       *extend,
		InitialCapital:          *capital,
		SignalThreshold:         *threshold,
	}

	spyLabel := "SPY regime ON"
	if !*nospy {
		fmt.Printf("\n  Fetching SPY for market regime filter...\n")
		spyCandles, err := finnhub.FetchCandlesMonths("SPY", *months)
		if err != nil {
			fmt.Printf("  [WARN] Could not fetch SPY candles: %v — regime filter disabled\n", err)
		} else {
			cfg.MarketCandles = &spyCandles
		}
		time.Sleep(800 * time.Millisecond)
	} else {
		spyLabel = "SPY regime OFF"
	}

	thr := *threshold
	if thr <= 0 {
		thr = 7
	}
	fmt.Printf("\n  Backtest  |  %d months  |  hold %d–%d days  |  stop %.1fx ATR  |  trail %.1fx ATR  |  extend %.1fx  |  capital $%.0f  |  threshold %d  |  %s\n",
		*months, *minhold, *hold, *atrMult, *trail, *extend, *capital, thr, spyLabel)

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
