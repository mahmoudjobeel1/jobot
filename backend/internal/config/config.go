package config

import "jobot/internal/portfolio"

var Tickers = portfolio.Tickers()

const (
	HistoryDays           = 120
	NewsLimit             = 8
	MemoryLimit           = 40
	MemoryContextWindow   = 8
	MinConfidenceToNotify = "Medium"
)

var NotifyOn = []string{"BUY", "SELL", "HOLD"}

// Sectors maps each sector label to its tickers. Used for cross-ticker
// correlation context injected into the Claude prompt.
var Sectors = map[string][]string{
	"Semiconductors":    {"NVDA", "AMD", "TSM", "INTC", "SNDK"},
	"Big Tech":          {"AAPL", "MSFT", "GOOG", "META"},
	"E-Commerce/Cloud":  {"AMZN", "BABA", "ORCL"},
	"Transportation":    {"UBER"},
	"Commodities":       {"GLD"},
}

// SectorOf returns the sector name for a ticker, or "".
func SectorOf(ticker string) (string, []string) {
	for sector, tickers := range Sectors {
		for _, t := range tickers {
			if t == ticker {
				return sector, tickers
			}
		}
	}
	return "", nil
}