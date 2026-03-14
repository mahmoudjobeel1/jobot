package analyst

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"jobot/internal/config"
	"jobot/internal/finnhub"
	"jobot/internal/indicators"
	"jobot/internal/memory"
	"jobot/internal/portfolio"
	"jobot/internal/store"
)

// AnalysisResult is the structured output from one analysis cycle.
type AnalysisResult struct {
	Ticker      string            `json:"ticker"`
	Timestamp   string            `json:"timestamp"`
	Date        string            `json:"date"`
	Price       float64           `json:"price"`
	Decision    string            `json:"decision"`
	Confidence  string            `json:"confidence"`
	Reasoning   string            `json:"reasoning"`
	KeyRisk     string            `json:"key_risk"`
	PriceTarget *string           `json:"price_target"`
	StopLoss    *string           `json:"stop_loss"`
	Summary     string            `json:"summary"`
	Indicators  IndicatorSnapshot `json:"indicators"`
	// Portfolio fields
	Qty            float64  `json:"qty"`
	AvgCost        float64  `json:"avg_cost"`
	UnrealizedPL   float64  `json:"unrealized_pl"`
	UnrealizedPLPct float64 `json:"unrealized_pl_pct"`
}

// IndicatorSnapshot stores the indicator values alongside the result.
type IndicatorSnapshot struct {
	RSI           *float64 `json:"rsi"`
	MACDHistogram *float64 `json:"macdHistogram"`
	MA20          *float64 `json:"ma20"`
	MA50          *float64 `json:"ma50"`
	MA200         *float64 `json:"ma200"`
	Trend60d      *float64 `json:"trend60d"`
	ADX14         *float64 `json:"adx14"`
}

// claudeResponse matches the JSON the model is instructed to return.
type claudeResponse struct {
	Decision    string  `json:"decision"`
	Confidence  string  `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
	KeyRisk     string  `json:"key_risk"`
	PriceTarget *string `json:"price_target"`
	StopLoss    *string `json:"stop_loss"`
	Summary     string  `json:"summary"`
}

func fmtFloat(f *float64, def string) string {
	if f == nil {
		return def
	}
	return fmt.Sprintf("%g", *f)
}

// buildSectorContext loads last-known prices and decisions for sector peers.
func buildSectorContext(ticker string) string {
	sector, peers := config.SectorOf(ticker)
	if sector == "" {
		return "No sector peers identified."
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Sector: %s", sector))
	sameDir := 0
	total := 0
	for _, peer := range peers {
		if strings.EqualFold(peer, ticker) {
			continue
		}
		entries, _ := memory.LoadMemory(peer)
		if len(entries) == 0 {
			continue
		}
		last := entries[len(entries)-1]
		trend := "flat"
		if last.Trend60d != nil {
			if *last.Trend60d > 1 {
				trend = fmt.Sprintf("+%.1f%%", *last.Trend60d)
				sameDir++
			} else if *last.Trend60d < -1 {
				trend = fmt.Sprintf("%.1f%%", *last.Trend60d)
				sameDir++
			}
		}
		total++
		lines = append(lines, fmt.Sprintf("  %-6s last $%g → %s (%s) | 60d: %s", peer, last.Price, last.Decision, last.Confidence, trend))
	}
	if total > 1 && sameDir == total {
		lines = append(lines, "  ⚠ All peers trending same direction — likely sector-wide move, not ticker-specific.")
	}
	return strings.Join(lines, "\n")
}

// weeklyContext returns a short summary block from the stored weekly entry (if any).
func weeklyContext(ticker string) string {
	w, _ := memory.LoadWeekly(ticker)
	if w == nil {
		return "No weekly review available yet."
	}
	themes := strings.Join(w.KeyThemes, ", ")
	return fmt.Sprintf("[Weekly review %s] Dominant: %s | %s | Themes: %s | Outlook: %s",
		w.GeneratedAt[:10], w.DominantDecision, w.Pattern, themes, w.Outlook)
}

// buildRegimeContext returns a one-line regime summary for the Claude prompt.
func buildRegimeContext(ind indicators.Indicators, price float64) string {
	regime := indicators.ClassifyRegime(ind, price)
	adxStr := "N/A"
	if ind.ADX14 != nil {
		adxStr = fmt.Sprintf("%.1f", *ind.ADX14)
	}
	switch regime {
	case indicators.RegimeTrending:
		return fmt.Sprintf("REGIME: TRENDING (ADX=%s, price above MA200). Favor holding winners longer. Trailing stops preferred over timeouts.", adxStr)
	case indicators.RegimeBearish:
		return fmt.Sprintf("REGIME: BEARISH (price below MA200, MA50 declining). Avoid new BUY signals. Tighten stops on existing positions.")
	default:
		return fmt.Sprintf("REGIME: SIDEWAYS (ADX=%s). Use tighter profit targets. Higher conviction needed for BUY.", adxStr)
	}
}

// buildTrailingStopContext returns prompt text about any active trailing stop for this ticker.
// Returns empty string if no trailing stop is active.
func buildTrailingStopContext(ticker string) string {
	for _, s := range store.GetStops() {
		if !s.IsTrailing || !strings.EqualFold(s.Ticker, ticker) {
			continue
		}
		if s.Activated {
			return fmt.Sprintf(
				"TRAILING STOP ACTIVE: stop at $%.2f (trailing 1.5×ATR below peak of $%.2f). "+
					"Do NOT recommend SELL based on time held. Let the trailing stop manage the exit. "+
					"Only recommend SELL if you see fundamental deterioration the trailing stop won't catch.",
				s.StopPrice, s.PeakPrice)
		}
		return fmt.Sprintf(
			"STOP-LOSS ACTIVE: initial stop at $%.2f (entry $%.2f). "+
				"Trailing mode activates once position gains %.1f%%. "+
				"Do not recommend SELL just because the position has been held for several days.",
			s.StopPrice, s.EntryPrice, s.ActivateAt)
	}
	return ""
}

// buildFastReentryContext returns prompt text if this ticker had a recent trail-stop exit
// and current price conditions suggest a fast re-entry is reasonable.
func buildFastReentryContext(ticker string, currentPrice float64) string {
	exit := store.RecentTrailExit(ticker, 20)
	if exit == nil {
		return ""
	}
	// Guardrail: only suggest re-entry if price hasn't fallen far below exit
	if currentPrice < exit.ExitPrice*0.95 {
		return ""
	}
	return fmt.Sprintf(
		"FAST RE-ENTRY ELIGIBLE: A trailing stop exited this position recently at $%.2f "+
			"(peak was $%.2f). Stock is still near exit level — a lower-conviction BUY is "+
			"acceptable if price has pulled back to support and trend remains intact.",
		exit.ExitPrice, exit.PeakPrice)
}

func buildPrompt(ticker string, quote finnhub.Quote, candles indicators.Candles, news []finnhub.NewsItem, memoryContext string) string {
	ind := indicators.ComputeAll(candles)

	priceVsMA := func(ma *float64, label string) string {
		if ma == nil {
			return label + ": N/A"
		}
		diff := (quote.C - *ma) / *ma * 100
		sign := "+"
		dir := "above"
		if diff < 0 {
			sign = ""
			dir = "below"
		}
		return fmt.Sprintf("%s: $%g (price is %s%.2f%% %s)", label, *ma, sign, diff, dir)
	}

	var newsLines []string
	if len(news) > 0 {
		for _, n := range news {
			newsLines = append(newsLines, fmt.Sprintf("  • [%s] %s (%s)", n.Date, n.Headline, n.Source))
		}
	}
	newsBlock := "  No recent news found."
	if len(newsLines) > 0 {
		newsBlock = strings.Join(newsLines, "\n")
	}

	volumeNote := "N/A"
	if ind.AvgVol != nil && ind.CurVol != nil {
		curVol := *ind.CurVol
		avgVol := float64(*ind.AvgVol)
		pct := (curVol/avgVol - 1) * 100
		dir := "above"
		if curVol < avgVol {
			dir = "below"
		}
		volumeNote = fmt.Sprintf("%s vs 20-day avg %s (%.1f%% %s avg)",
			formatVolume(curVol), formatVolume(avgVol), pct, dir)
	}

	rsiStr := "N/A"
	rsiLabel := ""
	if ind.RSI != nil {
		rsiStr = fmt.Sprintf("%g", *ind.RSI)
		if *ind.RSI > 70 {
			rsiLabel = " <- OVERBOUGHT"
		} else if *ind.RSI < 30 {
			rsiLabel = " <- OVERSOLD"
		}
	}

	macdLine := "N/A"
	macdSignal := "N/A"
	macdHist := "N/A"
	if ind.MACD != nil {
		macdLine = fmt.Sprintf("%g", ind.MACD.MACD)
		if ind.MACD.Signal != nil {
			macdSignal = fmt.Sprintf("%g", *ind.MACD.Signal)
		}
		if ind.MACD.Histogram != nil {
			macdHist = fmt.Sprintf("%g", *ind.MACD.Histogram)
		}
	}

	trend60dStr := "N/A"
	if ind.Trend60d != nil {
		sign := ""
		if *ind.Trend60d > 0 {
			sign = "+"
		}
		trend60dStr = fmt.Sprintf("%s%g%%", sign, *ind.Trend60d)
	}

	dpStr := fmt.Sprintf("%.2f", quote.Dp)
	dStr := fmt.Sprintf("%.2f", quote.D)

	utcTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	// Build portfolio context for this ticker
	portfolioCtx   := portfolio.BuildPortfolioContext(ticker, quote.C)
	sectorCtx      := buildSectorContext(ticker)
	weeklyCtx      := weeklyContext(ticker)
	regimeCtx      := buildRegimeContext(ind, quote.C)
	trailStopCtx   := buildTrailingStopContext(ticker)
	fastReentryCtx := buildFastReentryContext(ticker, quote.C)

	// Assemble optional context blocks
	var optionalBlocks []string
	if trailStopCtx != "" {
		optionalBlocks = append(optionalBlocks, "═══ ACTIVE STOP ORDER ═══\n"+trailStopCtx)
	}
	if fastReentryCtx != "" {
		optionalBlocks = append(optionalBlocks, "═══ RE-ENTRY SIGNAL ═══\n"+fastReentryCtx)
	}
	optionalSection := ""
	if len(optionalBlocks) > 0 {
		optionalSection = "\n\n" + strings.Join(optionalBlocks, "\n\n")
	}

	adx14Str := "N/A"
	if ind.ADX14 != nil {
		adx14Str = fmt.Sprintf("%g", *ind.ADX14)
	}

	return fmt.Sprintf(`You are a professional quantitative stock analyst advising a retail investor on their EXISTING portfolio. Analyze %s using all the data below and return a clear trading decision.

Your recommendation must account for the investor's current position — their cost basis, unrealized P&L, and position size. For example:
- If the stock is well above cost basis, consider whether it's time to take profits.
- If the stock is underwater, consider whether the thesis still holds or if it's better to cut losses.
- A "BUY" means add to the existing position; "SELL" means reduce or exit; "HOLD" means keep as-is.
- If sector peers are all moving in the same direction, weight sector macro over ticker-specific signals.
- Respect the regime context below — TRENDING stocks deserve longer holds; BEARISH regime warrants caution.%s

═══ LIVE MARKET DATA — %s ═══
Ticker:         %s
Current Price:  $%g
Open/High/Low:  $%g / $%g / $%g
Prev Close:     $%g
Daily Change:   %s%% ($%s)
60-day Trend:   %s

═══ MARKET REGIME ═══
%s

═══ YOUR PORTFOLIO POSITION ═══
%s

═══ TECHNICAL INDICATORS ═══
RSI (14):       %s%s
MACD Line:      %s
MACD Signal:    %s
MACD Histogram: %s
%s
%s
%s
ADX (14):       %s  (>25 = trending, <20 = sideways)
Volume:         %s

═══ SECTOR PEERS (same-sector correlation) ═══
%s

═══ WEEKLY MULTI-SESSION REVIEW ═══
%s

═══ RECENT NEWS & SENTIMENT ═══
%s

═══ YOUR ANALYSIS HISTORY (accumulated memory) ═══
%s

Respond with ONLY a valid JSON object — no explanation, no markdown fences:
{
  "decision": "BUY" | "SELL" | "HOLD",
  "confidence": "Low" | "Medium" | "High",
  "reasoning": "2–4 sentences integrating technicals + news + portfolio P&L + sector context + memory",
  "key_risk": "The single most important risk factor right now",
  "price_target": "$XX.XX or null",
  "stop_loss": "$XX.XX or null",
  "summary": "One concise sentence for memory storage"
}`,
		ticker,
		optionalSection,
		utcTime,
		ticker,
		quote.C,
		quote.O, quote.H, quote.L,
		quote.Pc,
		dpStr, dStr,
		trend60dStr,
		regimeCtx,
		portfolioCtx,
		rsiStr, rsiLabel,
		macdLine,
		macdSignal,
		macdHist,
		priceVsMA(ind.MA20, "MA20 "),
		priceVsMA(ind.MA50, "MA50 "),
		priceVsMA(ind.MA200, "MA200"),
		adx14Str,
		volumeNote,
		sectorCtx,
		weeklyCtx,
		newsBlock,
		memoryContext,
	)
}

func formatVolume(v float64) string {
	n := int64(v)
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	return string(result)
}

// AnalyzeStock calls Claude to analyze a stock and persists the result to memory.
func AnalyzeStock(ticker string, quote finnhub.Quote, candles indicators.Candles, news []finnhub.NewsItem) (AnalysisResult, error) {
	ind := indicators.ComputeAll(candles)
	memCtx := memory.BuildMemoryContext(ticker, config.MemoryContextWindow)
	prompt := buildPrompt(ticker, quote, candles, news, memCtx)

	fmt.Printf("  [Claude] Analyzing %s...\n", ticker)

	client := anthropic.NewClient(option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model("claude-sonnet-4-20250514")),
		MaxTokens: anthropic.F(int64(1024)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("claude API error: %w", err)
	}

	var rawParts []string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawParts = append(rawParts, block.Text)
		}
	}
	rawText := strings.TrimSpace(strings.NewReplacer("```json", "", "```", "").Replace(strings.Join(rawParts, "")))

	var parsed claudeResponse
	if err := json.Unmarshal([]byte(rawText), &parsed); err != nil {
		return AnalysisResult{}, fmt.Errorf("claude returned non-JSON for %s: %s", ticker, rawText[:min(len(rawText), 200)])
	}

	now := time.Now()

	// Enrich result with portfolio data
	var qty, avgCost, unrealizedPL, unrealizedPLPct float64
	if h := portfolio.Lookup(ticker); h != nil {
		qty = h.Qty
		avgCost = h.AvgCost
		unrealizedPL = h.UnrealizedPL(quote.C)
		unrealizedPLPct = h.UnrealizedPLPct(quote.C)
	}

	result := AnalysisResult{
		Ticker:      ticker,
		Timestamp:   now.UTC().Format(time.RFC3339),
		Date:        now.UTC().Format(time.RFC3339),
		Price:       quote.C,
		Decision:    parsed.Decision,
		Confidence:  parsed.Confidence,
		Reasoning:   parsed.Reasoning,
		KeyRisk:     parsed.KeyRisk,
		PriceTarget: parsed.PriceTarget,
		StopLoss:    parsed.StopLoss,
		Summary:     parsed.Summary,
		Indicators: IndicatorSnapshot{
			RSI:           ind.RSI,
			MACDHistogram: func() *float64 { if ind.MACD != nil { return ind.MACD.Histogram }; return nil }(),
			MA20:          ind.MA20,
			MA50:          ind.MA50,
			MA200:         ind.MA200,
			Trend60d:      ind.Trend60d,
			ADX14:         ind.ADX14,
		},
		Qty:             qty,
		AvgCost:         avgCost,
		UnrealizedPL:    unrealizedPL,
		UnrealizedPLPct: unrealizedPLPct,
	}

	memEntry := memory.Entry{
		Date:            result.Date,
		Decision:        result.Decision,
		Confidence:      result.Confidence,
		Price:           result.Price,
		RSI:             result.Indicators.RSI,
		MACDHistogram:   result.Indicators.MACDHistogram,
		Summary:         result.Summary,
		Reasoning:       result.Reasoning,
		KeyRisk:         result.KeyRisk,
		Trend60d:        result.Indicators.Trend60d,
		PriceTarget:     result.PriceTarget,
		StopLoss:        result.StopLoss,
		Qty:             qty,
		AvgCost:         avgCost,
		UnrealizedPL:    unrealizedPL,
		UnrealizedPLPct: unrealizedPLPct,
	}
	if err := memory.AppendMemory(ticker, memEntry); err != nil {
		fmt.Printf("  [Memory] Warning: could not save memory for %s: %v\n", ticker, err)
	}

	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure fmtFloat is used (it's a helper exposed for potential tests).
var _ = fmtFloat

// weeklyClaudeResponse matches the JSON for the weekly review call.
type weeklyClaudeResponse struct {
	Outlook          string   `json:"outlook"`
	DominantDecision string   `json:"dominant_decision"`
	Pattern          string   `json:"pattern"`
	KeyThemes        []string `json:"key_themes"`
}

// AnalyzeWeekly runs a multi-session review using all stored memory entries
// for a ticker and persists the result to a separate weekly summary file.
func AnalyzeWeekly(ticker string) error {
	entries, err := memory.LoadMemory(ticker)
	if err != nil || len(entries) < 3 {
		return fmt.Errorf("not enough history for weekly review of %s (have %d entries)", ticker, len(entries))
	}

	// Format all entries as a numbered list
	var lines []string
	for i, e := range entries {
		trend := "N/A"
		if e.Trend60d != nil {
			trend = fmt.Sprintf("%+.1f%%", *e.Trend60d)
		}
		lines = append(lines, fmt.Sprintf(
			"%d. [%s] %s (%s) @ $%g | 60d: %s | Risk: %s | %s",
			i+1, e.Date[:10], e.Decision, e.Confidence, e.Price, trend, e.KeyRisk, e.Summary,
		))
	}
	history := strings.Join(lines, "\n")

	prompt := fmt.Sprintf(`You are a senior portfolio analyst performing a WEEKLY MULTI-SESSION REVIEW of %s.

Below are %d analysis sessions recorded over the past weeks. Each entry is one 15-minute analysis cycle during market hours.

FULL SESSION HISTORY:
%s

Identify:
1. The dominant trend direction across all sessions
2. Consistency of BUY/SELL/HOLD decisions — is the model flip-flopping or stable?
3. Recurring risk themes
4. A forward-looking weekly outlook

Respond with ONLY valid JSON — no markdown, no explanation:
{
  "outlook": "2-3 sentence forward-looking weekly outlook",
  "dominant_decision": "BUY" | "SELL" | "HOLD",
  "pattern": "1 sentence on decision consistency and trend",
  "key_themes": ["theme1", "theme2", "theme3"]
}`, ticker, len(entries), history)

	client := anthropic.NewClient(option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")))
	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model("claude-sonnet-4-20250514")),
		MaxTokens: anthropic.F(int64(512)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})
	if err != nil {
		return fmt.Errorf("weekly claude error for %s: %w", ticker, err)
	}

	var rawParts []string
	for _, block := range msg.Content {
		if block.Type == "text" {
			rawParts = append(rawParts, block.Text)
		}
	}
	rawText := strings.TrimSpace(strings.NewReplacer("```json", "", "```", "").Replace(strings.Join(rawParts, "")))

	var parsed weeklyClaudeResponse
	if err := json.Unmarshal([]byte(rawText), &parsed); err != nil {
		return fmt.Errorf("weekly parse error for %s: %w", ticker, err)
	}

	entry := memory.WeeklyEntry{
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		Outlook:          parsed.Outlook,
		DominantDecision: parsed.DominantDecision,
		Pattern:          parsed.Pattern,
		KeyThemes:        parsed.KeyThemes,
	}
	return memory.SaveWeekly(ticker, entry)
}