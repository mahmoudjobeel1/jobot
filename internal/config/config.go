package config

var Tickers = []string{"AAPL", "MSFT", "NVDA", "TSLA", "AMZN"}

const (
	HistoryDays          = 120
	NewsLimit            = 8
	MemoryLimit          = 40
	MemoryContextWindow  = 8
	MinConfidenceToNotify = "Medium"
)

var NotifyOn = []string{"BUY", "SELL", "HOLD"}
