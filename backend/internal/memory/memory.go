package memory

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"jobot/internal/config"
)

// Entry represents a single historical analysis record stored in memory.
type Entry struct {
	Date            string   `json:"date"`
	Decision        string   `json:"decision"`
	Confidence      string   `json:"confidence"`
	Price           float64  `json:"price"`
	RSI             *float64 `json:"rsi"`
	MACDHistogram   *float64 `json:"macdHistogram"`
	Summary         string   `json:"summary"`
	Reasoning       string   `json:"reasoning"`
	KeyRisk         string   `json:"keyRisk"`
	Trend60d        *float64 `json:"trend60d"`
	PriceTarget     *string  `json:"priceTarget"`
	StopLoss        *string  `json:"stopLoss"`
	// Portfolio fields
	Qty             float64  `json:"qty"`
	AvgCost         float64  `json:"avgCost"`
	UnrealizedPL    float64  `json:"unrealizedPL"`
	UnrealizedPLPct float64  `json:"unrealizedPLPct"`
}

// WeeklyEntry stores the result of a weekly multi-session review.
type WeeklyEntry struct {
	GeneratedAt      string   `json:"generated_at"`
	Outlook          string   `json:"outlook"`
	DominantDecision string   `json:"dominant_decision"`
	Pattern          string   `json:"pattern"`
	KeyThemes        []string `json:"key_themes"`
}

// DailyDigest aggregates all analysis sessions for a single trading day.
// These accumulate indefinitely so long-term patterns are never lost.
type DailyDigest struct {
	Date             string         `json:"date"`           // YYYY-MM-DD
	SessionCount     int            `json:"session_count"`
	DominantDecision string         `json:"dominant_decision"`
	AvgPrice         float64        `json:"avg_price"`
	PriceHigh        float64        `json:"price_high"`
	PriceLow         float64        `json:"price_low"`
	AvgRSI           *float64       `json:"avg_rsi"`
	AvgMACDHist      *float64       `json:"avg_macd_hist"`
	AvgTrend60d      *float64       `json:"avg_trend60d"`
	Decisions        map[string]int `json:"decisions"`
	KeyRisks         []string       `json:"key_risks"`
	BestSummary      string         `json:"best_summary"`
}

func dailyDigestFilePath(ticker string) string {
	return filepath.Join(dataDir(), strings.ToUpper(ticker)+"_daily.json")
}

// LoadDailyDigests reads all stored daily digests for a ticker.
func LoadDailyDigests(ticker string) ([]DailyDigest, error) {
	data, err := os.ReadFile(dailyDigestFilePath(ticker))
	if os.IsNotExist(err) {
		return []DailyDigest{}, nil
	}
	if err != nil {
		return []DailyDigest{}, nil
	}
	var digests []DailyDigest
	if err := json.Unmarshal(data, &digests); err != nil {
		return []DailyDigest{}, nil
	}
	return digests, nil
}

func saveDailyDigests(ticker string, digests []DailyDigest) error {
	if err := ensureDataDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(digests, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dailyDigestFilePath(ticker), data, 0o644)
}

// buildDailyDigest creates a DailyDigest from a slice of same-day entries.
func buildDailyDigest(date string, entries []Entry) DailyDigest {
	d := DailyDigest{
		Date:      date,
		Decisions: make(map[string]int),
		PriceHigh: -math.MaxFloat64,
		PriceLow:  math.MaxFloat64,
	}

	var priceSum, rsiSum, macdSum, trendSum float64
	var rsiCount, macdCount, trendCount int
	riskSeen := make(map[string]bool)

	for _, e := range entries {
		d.SessionCount++
		d.Decisions[e.Decision]++
		priceSum += e.Price
		if e.Price > d.PriceHigh {
			d.PriceHigh = e.Price
		}
		if e.Price < d.PriceLow {
			d.PriceLow = e.Price
		}
		if e.RSI != nil {
			rsiSum += *e.RSI
			rsiCount++
		}
		if e.MACDHistogram != nil {
			macdSum += *e.MACDHistogram
			macdCount++
		}
		if e.Trend60d != nil {
			trendSum += *e.Trend60d
			trendCount++
		}
		if e.KeyRisk != "" && !riskSeen[e.KeyRisk] {
			riskSeen[e.KeyRisk] = true
			d.KeyRisks = append(d.KeyRisks, e.KeyRisk)
		}
		// Use the last entry's summary as the day's best summary
		if e.Summary != "" {
			d.BestSummary = e.Summary
		}
	}

	if d.SessionCount > 0 {
		d.AvgPrice = priceSum / float64(d.SessionCount)
	}
	if rsiCount > 0 {
		v := rsiSum / float64(rsiCount)
		d.AvgRSI = &v
	}
	if macdCount > 0 {
		v := macdSum / float64(macdCount)
		d.AvgMACDHist = &v
	}
	if trendCount > 0 {
		v := trendSum / float64(trendCount)
		d.AvgTrend60d = &v
	}

	// Dominant decision = most frequent
	best, bestN := "", 0
	for dec, n := range d.Decisions {
		if n > bestN {
			best, bestN = dec, n
		}
	}
	d.DominantDecision = best

	return d
}

// mergeDigests merges newer entries into an existing digest for the same date.
func mergeDigests(existing DailyDigest, newer DailyDigest) DailyDigest {
	total := existing.SessionCount + newer.SessionCount
	if total == 0 {
		return existing
	}
	merged := existing
	merged.SessionCount = total
	merged.AvgPrice = (existing.AvgPrice*float64(existing.SessionCount) + newer.AvgPrice*float64(newer.SessionCount)) / float64(total)
	if newer.PriceHigh > existing.PriceHigh {
		merged.PriceHigh = newer.PriceHigh
	}
	if newer.PriceLow < existing.PriceLow {
		merged.PriceLow = newer.PriceLow
	}
	for dec, n := range newer.Decisions {
		merged.Decisions[dec] += n
	}
	riskSeen := make(map[string]bool)
	for _, r := range existing.KeyRisks {
		riskSeen[r] = true
	}
	for _, r := range newer.KeyRisks {
		if !riskSeen[r] {
			merged.KeyRisks = append(merged.KeyRisks, r)
		}
	}
	if newer.BestSummary != "" {
		merged.BestSummary = newer.BestSummary
	}
	// Recalculate dominant
	best, bestN := "", 0
	for dec, n := range merged.Decisions {
		if n > bestN {
			best, bestN = dec, n
		}
	}
	merged.DominantDecision = best
	return merged
}

// aggregateEvicted groups evicted entries by date and persists daily digests.
func aggregateEvicted(ticker string, evicted []Entry) error {
	groups := make(map[string][]Entry)
	var order []string
	seen := make(map[string]bool)
	for _, e := range evicted {
		day := e.Date
		if len(day) >= 10 {
			day = day[:10]
		}
		if !seen[day] {
			order = append(order, day)
			seen[day] = true
		}
		groups[day] = append(groups[day], e)
	}

	digests, _ := LoadDailyDigests(ticker)
	digestIndex := make(map[string]int)
	for i, d := range digests {
		digestIndex[d.Date] = i
	}

	for _, day := range order {
		newDigest := buildDailyDigest(day, groups[day])
		if idx, exists := digestIndex[day]; exists {
			digests[idx] = mergeDigests(digests[idx], newDigest)
		} else {
			digests = append(digests, newDigest)
			digestIndex[day] = len(digests) - 1
		}
	}

	// Keep digests sorted by date ascending
	sort.Slice(digests, func(i, j int) bool {
		return digests[i].Date < digests[j].Date
	})

	return saveDailyDigests(ticker, digests)
}

func weeklyFilePath(ticker string) string {
	return filepath.Join(dataDir(), strings.ToUpper(ticker)+"_weekly.json")
}

// LoadWeekly reads the latest weekly summary for a ticker.
func LoadWeekly(ticker string) (*WeeklyEntry, error) {
	data, err := os.ReadFile(weeklyFilePath(ticker))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var e WeeklyEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// SaveWeekly persists a weekly summary entry.
func SaveWeekly(ticker string, e WeeklyEntry) error {
	if err := ensureDataDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(weeklyFilePath(ticker), data, 0o644)
}

// dataDir returns the path to the data directory relative to the project root.
func dataDir() string {
	if dir := os.Getenv("JOBOT_DATA_DIR"); dir != "" {
		return dir
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "data"
	}
	// filepath.Dir gives …/internal/memory — go up 2 levels to project root
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(root, "data")
}

func ensureDataDir() error {
	return os.MkdirAll(dataDir(), 0o755)
}

func filePath(ticker string) string {
	return filepath.Join(dataDir(), strings.ToUpper(ticker)+".json")
}

// LoadMemory reads all stored entries for a ticker.
func LoadMemory(ticker string) ([]Entry, error) {
	if err := ensureDataDir(); err != nil {
		return nil, err
	}
	fp := filePath(ticker)
	data, err := os.ReadFile(fp)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return []Entry{}, nil
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []Entry{}, nil
	}
	return entries, nil
}

// AppendMemory appends a new entry to the ticker's memory file, capped at MEMORY_LIMIT.
// Entries that fall off the rolling window are aggregated into daily digests so
// long-term history is never permanently lost.
func AppendMemory(ticker string, entry Entry) error {
	if err := ensureDataDir(); err != nil {
		return err
	}
	entries, _ := LoadMemory(ticker)
	entries = append(entries, entry)
	if len(entries) > config.MemoryLimit {
		evicted := make([]Entry, len(entries)-config.MemoryLimit)
		copy(evicted, entries[:len(entries)-config.MemoryLimit])
		if err := aggregateEvicted(ticker, evicted); err != nil {
			fmt.Printf("  [Memory] Warning: could not aggregate evicted entries for %s: %v\n", ticker, err)
		}
		entries = entries[len(entries)-config.MemoryLimit:]
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath(ticker), data, 0o644)
}

// BuildMemoryContext constructs the textual memory context block for the prompt.
// It includes two sections:
//  1. Long-term: up to 30 daily digests (compressed per-day aggregates, indefinite history)
//  2. Short-term: the most recent `limit` raw entries (full detail)
func BuildMemoryContext(ticker string, limit int) string {
	var sections []string

	// ── Long-term: daily digests ─────────────────────────────────────
	digests, _ := LoadDailyDigests(ticker)
	const maxDigests = 30
	if len(digests) > maxDigests {
		digests = digests[len(digests)-maxDigests:]
	}
	if len(digests) > 0 {
		var digestLines []string
		digestLines = append(digestLines, "── Daily Aggregates (long-term history) ──")
		for _, d := range digests {
			rsiStr := "N/A"
			if d.AvgRSI != nil {
				rsiStr = fmt.Sprintf("%.1f", *d.AvgRSI)
			}
			macdStr := "N/A"
			if d.AvgMACDHist != nil {
				macdStr = fmt.Sprintf("%+.3f", *d.AvgMACDHist)
			}
			trendStr := "N/A"
			if d.AvgTrend60d != nil {
				trendStr = fmt.Sprintf("%+.1f%%", *d.AvgTrend60d)
			}
			var decParts []string
			for _, dec := range []string{"BUY", "HOLD", "SELL"} {
				if n := d.Decisions[dec]; n > 0 {
					decParts = append(decParts, fmt.Sprintf("%s×%d", dec, n))
				}
			}
			risksStr := ""
			if len(d.KeyRisks) > 0 {
				risksStr = fmt.Sprintf(" | Risks: %s", strings.Join(d.KeyRisks[:min(3, len(d.KeyRisks))], "; "))
			}
			digestLines = append(digestLines, fmt.Sprintf(
				"[%s] %d sessions | Dominant: %-4s | Avg $%.2f (%.2f–%.2f) | RSI: %s | MACD: %s | 60d: %s | %s%s | %s",
				d.Date, d.SessionCount, d.DominantDecision,
				d.AvgPrice, d.PriceLow, d.PriceHigh,
				rsiStr, macdStr, trendStr,
				strings.Join(decParts, " "),
				risksStr,
				d.BestSummary,
			))
		}
		sections = append(sections, strings.Join(digestLines, "\n"))
	}

	// ── Short-term: raw recent entries ───────────────────────────────
	entries, _ := LoadMemory(ticker)
	if len(entries) == 0 && len(digests) == 0 {
		return "No prior analysis history for this ticker."
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	if len(entries) > 0 {
		var rawLines []string
		rawLines = append(rawLines, "── Recent Sessions (detailed) ──")
		for _, e := range entries {
			rsiStr := "N/A"
			if e.RSI != nil {
				rsiStr = fmt.Sprintf("%g", *e.RSI)
			}
			macdStr := "N/A"
			if e.MACDHistogram != nil {
				macdStr = fmt.Sprintf("%g", *e.MACDHistogram)
			}
			plStr := ""
			if e.Qty > 0 {
				plSign := "+"
				if e.UnrealizedPL < 0 {
					plSign = ""
				}
				plStr = fmt.Sprintf(" | Position: %.4g shares @ $%.2f | P&L: %s$%.2f (%s%.2f%%)",
					e.Qty, e.AvgCost,
					plSign, math.Abs(e.UnrealizedPL),
					plSign, math.Abs(e.UnrealizedPLPct))
			}
			rawLines = append(rawLines, fmt.Sprintf(
				"[%s] Decision: %s (%s) @ $%g | RSI: %s | MACD hist: %s%s | Summary: %s",
				e.Date, e.Decision, e.Confidence, e.Price, rsiStr, macdStr, plStr, e.Summary,
			))
		}
		sections = append(sections, strings.Join(rawLines, "\n"))
	}

	return strings.Join(sections, "\n\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}