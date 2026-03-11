package portfolio

import (
	"fmt"
	"math"
	"strings"

	"jobot/internal/store"
)

// Holding represents a single position in the portfolio.
type Holding struct {
	Ticker  string  `json:"ticker"`
	Qty     float64 `json:"qty"`
	AvgCost float64 `json:"avg_cost"`
}

// UnrealizedPL returns the unrealized profit/loss given the current price.
func (h Holding) UnrealizedPL(currentPrice float64) float64 {
	return (currentPrice - h.AvgCost) * h.Qty
}

// UnrealizedPLPct returns the unrealized P&L as a percentage.
func (h Holding) UnrealizedPLPct(currentPrice float64) float64 {
	if h.AvgCost == 0 {
		return 0
	}
	return (currentPrice - h.AvgCost) / h.AvgCost * 100
}

// CostBasis returns the total cost basis of the holding.
func (h Holding) CostBasis() float64 {
	return h.AvgCost * h.Qty
}

// MarketValue returns the current market value of the holding.
func (h Holding) MarketValue(currentPrice float64) float64 {
	return currentPrice * h.Qty
}


var Holdings = []Holding{
	{Ticker: "GLD", Qty: 0, AvgCost: 0},
	{Ticker: "AMZN", Qty: 8.0819, AvgCost: 222.16},
	{Ticker: "AMD", Qty: 8.02, AvgCost: 221.36},
	{Ticker: "GOOG", Qty: 4.1518, AvgCost: 331.98},
	{Ticker: "MSFT", Qty: 2.66, AvgCost: 412.24},
	{Ticker: "BABA", Qty: 4.6199, AvgCost: 129.55},
	{Ticker: "UBER", Qty: 6.1534, AvgCost: 81.05},
	{Ticker: "ORCL", Qty: 2.9628, AvgCost: 162.86},
	{Ticker: "NVDA", Qty: 1.4292, AvgCost: 180.06},
	{Ticker: "META", Qty: 0.3177, AvgCost: 699.02},
	{Ticker: "AAPL", Qty: 0.525, AvgCost: 256.44},
	{Ticker: "TSM", Qty: 0, AvgCost: 0},
	{Ticker: "TSLA", Qty: 0, AvgCost: 0},
	{Ticker: "INTC", Qty: 0, AvgCost: 0},
	{Ticker: "SNDK", Qty: 0, AvgCost: 0},
}

// Tickers returns the list of tickers from the portfolio.
func Tickers() []string {
	tickers := make([]string, len(Holdings))
	for i, h := range Holdings {
		tickers[i] = h.Ticker
	}
	return tickers
}

// GetHoldings returns live holdings from the store (falls back to static Holdings
// if the store has not been initialised yet).
func GetHoldings() []Holding {
	sh := store.GetHoldings()
	if len(sh) == 0 {
		return Holdings
	}
	out := make([]Holding, len(sh))
	for i, s := range sh {
		out[i] = Holding{Ticker: s.Ticker, Qty: s.Qty, AvgCost: s.AvgCost}
	}
	return out
}

// Lookup returns the live holding for a ticker from the store, falling back to
// the static Holdings slice if the store is not yet initialised.
func Lookup(ticker string) *Holding {
	if sh := store.GetHolding(ticker); sh != nil {
		return &Holding{Ticker: sh.Ticker, Qty: sh.Qty, AvgCost: sh.AvgCost}
	}
	for i := range Holdings {
		if strings.EqualFold(Holdings[i].Ticker, ticker) {
			return &Holdings[i]
		}
	}
	return nil
}

// PortfolioSummary computes aggregate stats given a map of current prices.
type PortfolioSummary struct {
	TotalCostBasis   float64
	TotalMarketValue float64
	TotalPL          float64
	TotalPLPct       float64
}

// Summary computes portfolio-level totals given current prices keyed by ticker.
func Summary(prices map[string]float64) PortfolioSummary {
	var s PortfolioSummary
	for _, h := range GetHoldings() {
		price, ok := prices[h.Ticker]
		if !ok {
			continue
		}
		s.TotalCostBasis += h.CostBasis()
		s.TotalMarketValue += h.MarketValue(price)
	}
	s.TotalPL = s.TotalMarketValue - s.TotalCostBasis
	if s.TotalCostBasis > 0 {
		s.TotalPLPct = s.TotalPL / s.TotalCostBasis * 100
	}
	return s
}

// BuildPortfolioContext returns a text block describing the user's position in
// a specific ticker, suitable for injecting into the Claude prompt.
func BuildPortfolioContext(ticker string, currentPrice float64) string {
	h := Lookup(ticker)
	if h == nil {
		return "No position in this ticker."
	}

	pl := h.UnrealizedPL(currentPrice)
	plPct := h.UnrealizedPLPct(currentPrice)
	plSign := "+"
	if pl < 0 {
		plSign = ""
	}

	return fmt.Sprintf(
		`You own this stock. Factor the existing position into your recommendation.
Shares Held:      %.4g
Avg Cost Basis:   $%.2f
Current Price:    $%.2f
Cost Basis Total: $%s
Market Value:     $%s
Unrealized P&L:   %s$%.2f (%s%.2f%%)`,
		h.Qty,
		h.AvgCost,
		currentPrice,
		formatMoney(h.CostBasis()),
		formatMoney(h.MarketValue(currentPrice)),
		plSign, math.Abs(pl),
		plSign, math.Abs(plPct),
	)
}

func formatMoney(v float64) string {
	n := int64(v)
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return fmt.Sprintf("%s.%02d", s, int64(math.Round((v-float64(n))*100)))
	}
	var result []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	cents := int64(math.Round((v - float64(n)) * 100))
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("%s.%02d", string(result), cents)
}