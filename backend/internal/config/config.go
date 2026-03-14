package config

import "jobot/internal/portfolio"

var Tickers = portfolio.Tickers()

const (
	HistoryDays           = 120
	NewsLimit             = 8
	MemoryLimit           = 40
	MemoryContextWindow   = 8
	MinConfidenceToNotify = "Medium"

	// Trading parameters — match v11 backtest config.
	StopATRMultiplier  = 2.5 // initial stop-loss = entry - 2.5×ATR14
	TrailATRMultiplier = 1.5 // trailing stop = peak - 1.5×ATR14
	TrailActivatePct   = 7.0 // activate trailing stop once trade is up 7%
	MaxHoldDays        = 10  // base timeout (disabled when trailing stop is active)
	TrendExtendMult    = 2.0 // extend hold in strong 60d trends
	SignalThreshold    = 6   // minimum bull score for BUY (v11 default)
	SidewaysThreshold  = 7   // higher threshold when ADX < 20 (sideways regime)
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