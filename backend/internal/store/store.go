// Package store provides thread-safe, file-persisted state for the portfolio
// and stop-loss orders. It is the single source of truth at runtime.
package store

import (
	"encoding/json"
	"fmt"
	"math"
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

// StopOrder represents a pending stop-loss order (static or trailing).
type StopOrder struct {
	ID        string  `json:"id"`
	Ticker    string  `json:"ticker"`
	Qty       float64 `json:"qty"`
	StopPrice float64 `json:"stop_price"`
	CreatedAt string  `json:"created_at"`

	// Trailing stop fields (zero-value = static stop)
	IsTrailing bool    `json:"is_trailing"`
	EntryPrice float64 `json:"entry_price"`    // original buy price
	PeakPrice  float64 `json:"peak_price"`     // highest price seen since entry
	TrailATR   float64 `json:"trail_atr"`      // ATR14 captured at entry
	TrailMult  float64 `json:"trail_mult"`     // multiplier (1.5× from config)
	ActivateAt float64 `json:"activate_at_pct"` // profit % required to engage trailing
	Activated  bool    `json:"activated"`       // true once trailing mode is engaged
}

// TrailExitRecord tracks a recent trailing-stop exit for fast-re-entry logic.
// Stored in memory only — intentionally resets on restart.
type TrailExitRecord struct {
	Ticker    string  `json:"ticker"`
	ExitPrice float64 `json:"exit_price"`
	ExitDate  string  `json:"exit_date"`
	PeakPrice float64 `json:"peak_price"`
}

var recentTrailExits []TrailExitRecord

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

// AddTrailingStop creates a trailing stop order using ATR-based parameters.
// The initial stop-loss is set at entry - 2.5×ATR. Trailing activates once
// the trade gains max(7%, 2×ATR%) from entry.
func AddTrailingStop(ticker string, qty, entryPrice, atr14, trailMult, activatePct float64) (StopOrder, error) {
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

	// Initial stop 2.5×ATR below entry; trailing activates at max(7%, 2×ATR%)
	initialStop := entryPrice - (2.5 * atr14)
	atrPct := (atr14 / entryPrice) * 100
	if activatePct <= 0 {
		activatePct = math.Max(7.0, 2.0*atrPct)
	}
	if trailMult <= 0 {
		trailMult = 1.5
	}

	s := StopOrder{
		ID:         fmt.Sprintf("trail_%d", time.Now().UnixNano()),
		Ticker:     ticker,
		Qty:        qty,
		StopPrice:  initialStop,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		IsTrailing: true,
		EntryPrice: entryPrice,
		PeakPrice:  entryPrice,
		TrailATR:   atr14,
		TrailMult:  trailMult,
		ActivateAt: activatePct,
		Activated:  false,
	}
	stops = append(stops, s)
	return s, writeStops()
}

// UpdateTrailingStop advances the trailing stop for the given stop ID using
// the current market price. Returns (triggered, error).
// Call this every analysis cycle for each active trailing stop.
func UpdateTrailingStop(id string, currentPrice float64) (triggered bool, err error) {
	mu.Lock()
	defer mu.Unlock()

	for i, s := range stops {
		if s.ID != id || !s.IsTrailing {
			continue
		}

		// Ratchet peak upward
		if currentPrice > stops[i].PeakPrice {
			stops[i].PeakPrice = currentPrice
		}

		gainPct := (currentPrice - s.EntryPrice) / s.EntryPrice * 100

		if !s.Activated {
			// Not yet in trailing mode — check static initial stop
			if currentPrice <= s.StopPrice {
				return true, writeStops()
			}
			// Activate trailing once gain threshold is reached
			if gainPct >= s.ActivateAt {
				stops[i].Activated = true
				newStop := stops[i].PeakPrice - (s.TrailATR * s.TrailMult)
				if newStop > stops[i].StopPrice {
					stops[i].StopPrice = newStop
				}
			}
			return false, writeStops()
		}

		// Trailing mode: ratchet stop up with peak, never down
		newStop := stops[i].PeakPrice - (s.TrailATR * s.TrailMult)
		if newStop > stops[i].StopPrice {
			stops[i].StopPrice = newStop
		}

		if currentPrice <= stops[i].StopPrice {
			return true, writeStops()
		}
		return false, writeStops()
	}
	return false, fmt.Errorf("trailing stop %s not found", id)
}

// RecordTrailExit stores a trail-stop exit for fast-re-entry context.
func RecordTrailExit(ticker string, exitPrice, peakPrice float64) {
	mu.Lock()
	defer mu.Unlock()
	recentTrailExits = append(recentTrailExits, TrailExitRecord{
		Ticker:    strings.ToUpper(ticker),
		ExitPrice: exitPrice,
		ExitDate:  time.Now().UTC().Format(time.RFC3339),
		PeakPrice: peakPrice,
	})
	// Keep only the last 50 exits
	if len(recentTrailExits) > 50 {
		recentTrailExits = recentTrailExits[len(recentTrailExits)-50:]
	}
}

// RecentTrailExit returns the most recent trail-stop exit for a ticker
// within the last withinDays calendar days, or nil if none.
func RecentTrailExit(ticker string, withinDays int) *TrailExitRecord {
	mu.RLock()
	defer mu.RUnlock()
	cutoff := time.Now().AddDate(0, 0, -withinDays)
	ticker = strings.ToUpper(ticker)
	for i := len(recentTrailExits) - 1; i >= 0; i-- {
		r := recentTrailExits[i]
		if r.Ticker != ticker {
			continue
		}
		t, err := time.Parse(time.RFC3339, r.ExitDate)
		if err != nil {
			continue
		}
		if t.After(cutoff) {
			return &r
		}
	}
	return nil
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
// For trailing stops, also records a TrailExitRecord for fast-re-entry context.
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

	// Record trail exit for fast re-entry context (no lock needed — already held)
	if s.IsTrailing && s.Activated {
		recentTrailExits = append(recentTrailExits, TrailExitRecord{
			Ticker:    s.Ticker,
			ExitPrice: s.StopPrice,
			ExitDate:  time.Now().UTC().Format(time.RFC3339),
			PeakPrice: s.PeakPrice,
		})
	}

	if err := writePortfolio(); err != nil {
		return err
	}
	return writeStops()
}
