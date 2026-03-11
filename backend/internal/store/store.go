// Package store provides thread-safe, file-persisted state for the portfolio
// and stop-loss orders. It is the single source of truth at runtime.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Holding mirrors portfolio.Holding but lives here to avoid circular imports.
type Holding struct {
	Ticker  string  `json:"ticker"`
	Qty     float64 `json:"qty"`
	AvgCost float64 `json:"avg_cost"`
}

// StopOrder represents a pending stop-loss order.
type StopOrder struct {
	ID        string  `json:"id"`
	Ticker    string  `json:"ticker"`
	Qty       float64 `json:"qty"`
	StopPrice float64 `json:"stop_price"`
	CreatedAt string  `json:"created_at"`
}

var (
	mu            sync.RWMutex
	holdings      []Holding
	stops         []StopOrder
	portfolioFile string
	stopsFile     string
)

// Init loads state from disk, seeding from initialHoldings if no file exists yet.
func Init(initialHoldings []Holding) error {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("store: create data dir: %w", err)
	}
	portfolioFile = filepath.Join(dir, "portfolio.json")
	stopsFile = filepath.Join(dir, "stops.json")

	if raw, err := os.ReadFile(portfolioFile); err == nil {
		_ = json.Unmarshal(raw, &holdings)
	}
	if len(holdings) == 0 {
		holdings = make([]Holding, len(initialHoldings))
		copy(holdings, initialHoldings)
		if err := writePortfolio(); err != nil {
			return err
		}
	}

	if raw, err := os.ReadFile(stopsFile); err == nil {
		_ = json.Unmarshal(raw, &stops)
	}
	if stops == nil {
		stops = []StopOrder{}
	}
	return nil
}

func dataDir() string {
	if dir := os.Getenv("JOBOT_DATA_DIR"); dir != "" {
		return dir
	}
	return "data"
}

func writePortfolio() error {
	raw, err := json.MarshalIndent(holdings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(portfolioFile, raw, 0o644)
}

func writeStops() error {
	raw, err := json.MarshalIndent(stops, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stopsFile, raw, 0o644)
}

// GetHoldings returns a copy of all holdings.
func GetHoldings() []Holding {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Holding, len(holdings))
	copy(out, holdings)
	return out
}

// GetHolding returns the holding for a ticker, or nil if not found.
func GetHolding(ticker string) *Holding {
	mu.RLock()
	defer mu.RUnlock()
	for i := range holdings {
		if strings.EqualFold(holdings[i].Ticker, ticker) {
			h := holdings[i]
			return &h
		}
	}
	return nil
}

// Buy adds shares to a holding (or creates one) and recalculates the avg cost.
func Buy(ticker string, qty, price float64) error {
	mu.Lock()
	defer mu.Unlock()
	ticker = strings.ToUpper(ticker)
	for i := range holdings {
		if strings.EqualFold(holdings[i].Ticker, ticker) {
			newQty := holdings[i].Qty + qty
			holdings[i].AvgCost = (holdings[i].Qty*holdings[i].AvgCost + qty*price) / newQty
			holdings[i].Qty = newQty
			return writePortfolio()
		}
	}
	holdings = append(holdings, Holding{Ticker: ticker, Qty: qty, AvgCost: price})
	return writePortfolio()
}

// Sell reduces a holding's qty.
func Sell(ticker string, qty float64) error {
	mu.Lock()
	defer mu.Unlock()
	ticker = strings.ToUpper(ticker)
	for i := range holdings {
		if strings.EqualFold(holdings[i].Ticker, ticker) {
			if holdings[i].Qty < qty {
				return fmt.Errorf("insufficient shares: hold %.4g, selling %.4g", holdings[i].Qty, qty)
			}
			holdings[i].Qty -= qty
			if holdings[i].Qty == 0 {
				holdings[i].AvgCost = 0
			}
			return writePortfolio()
		}
	}
	return fmt.Errorf("ticker %s not in portfolio", ticker)
}

// AddStop creates a new stop-loss order after validating available shares.
func AddStop(ticker string, qty, stopPrice float64) (StopOrder, error) {
	mu.Lock()
	defer mu.Unlock()
	ticker = strings.ToUpper(ticker)

	var held float64
	for _, h := range holdings {
		if strings.EqualFold(h.Ticker, ticker) {
			held = h.Qty
			break
		}
	}
	if held < qty {
		return StopOrder{}, fmt.Errorf("insufficient shares: hold %.4g, stop qty %.4g", held, qty)
	}

	s := StopOrder{
		ID:        fmt.Sprintf("stop_%d", time.Now().UnixNano()),
		Ticker:    ticker,
		Qty:       qty,
		StopPrice: stopPrice,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	stops = append(stops, s)
	return s, writeStops()
}

// GetStops returns a copy of all stop-loss orders.
func GetStops() []StopOrder {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]StopOrder, len(stops))
	copy(out, stops)
	return out
}

// DeleteStop removes a stop-loss order by ID.
func DeleteStop(id string) error {
	mu.Lock()
	defer mu.Unlock()
	for i, s := range stops {
		if s.ID == id {
			stops = append(stops[:i], stops[i+1:]...)
			return writeStops()
		}
	}
	return fmt.Errorf("stop %s not found", id)
}

// ExecuteStop sells the shares for a stop order and removes it.
func ExecuteStop(id string) error {
	mu.Lock()
	defer mu.Unlock()

	stopIdx := -1
	for i, s := range stops {
		if s.ID == id {
			stopIdx = i
			break
		}
	}
	if stopIdx == -1 {
		return fmt.Errorf("stop %s not found", id)
	}
	s := stops[stopIdx]

	for i := range holdings {
		if strings.EqualFold(holdings[i].Ticker, s.Ticker) {
			if holdings[i].Qty < s.Qty {
				return fmt.Errorf("insufficient shares for stop execution")
			}
			holdings[i].Qty -= s.Qty
			if holdings[i].Qty == 0 {
				holdings[i].AvgCost = 0
			}
			break
		}
	}
	stops = append(stops[:stopIdx], stops[stopIdx+1:]...)

	if err := writePortfolio(); err != nil {
		return err
	}
	return writeStops()
}
