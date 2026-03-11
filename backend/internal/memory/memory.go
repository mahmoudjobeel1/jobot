package memory

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
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
	PriceTarget     *string  `json:"priceTarget"`
	StopLoss        *string  `json:"stopLoss"`
	// Portfolio fields
	Qty             float64  `json:"qty"`
	AvgCost         float64  `json:"avgCost"`
	UnrealizedPL    float64  `json:"unrealizedPL"`
	UnrealizedPLPct float64  `json:"unrealizedPLPct"`
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
func AppendMemory(ticker string, entry Entry) error {
	if err := ensureDataDir(); err != nil {
		return err
	}
	entries, _ := LoadMemory(ticker)
	entries = append(entries, entry)
	if len(entries) > config.MemoryLimit {
		entries = entries[len(entries)-config.MemoryLimit:]
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath(ticker), data, 0o644)
}

// BuildMemoryContext constructs the textual memory context block for the prompt.
func BuildMemoryContext(ticker string, limit int) string {
	entries, _ := LoadMemory(ticker)
	if len(entries) == 0 {
		return "No prior analysis history for this ticker."
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	var lines []string
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

		lines = append(lines, fmt.Sprintf(
			"[%s] Decision: %s (%s) @ $%g | RSI: %s | MACD hist: %s%s | Summary: %s",
			e.Date, e.Decision, e.Confidence, e.Price, rsiStr, macdStr, plStr, e.Summary,
		))
	}
	return strings.Join(lines, "\n")
}