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
	portfolioCtx := portfolio.BuildPortfolioContext(ticker, quote.C)

	return fmt.Sprintf(`You are a professional quantitative stock analyst advising a retail investor on their EXISTING portfolio. Analyze %s using all the data below and return a clear trading decision.

Your recommendation must account for the investor's current position — their cost basis, unrealized P&L, and position size. For example:
- If the stock is well above cost basis, consider whether it's time to take profits.
- If the stock is underwater, consider whether the thesis still holds or if it's better to cut losses.
- A "BUY" means add to the existing position; "SELL" means reduce or exit; "HOLD" means keep as-is.

═══ LIVE MARKET DATA — %s ═══
Ticker:         %s
Current Price:  $%g
Open/High/Low:  $%g / $%g / $%g
Prev Close:     $%g
Daily Change:   %s%% ($%s)
60-day Trend:   %s

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
Volume:         %s

═══ RECENT NEWS & SENTIMENT ═══
%s

═══ YOUR ANALYSIS HISTORY (accumulated memory) ═══
%s

Respond with ONLY a valid JSON object — no explanation, no markdown fences:
{
  "decision": "BUY" | "SELL" | "HOLD",
  "confidence": "Low" | "Medium" | "High",
  "reasoning": "2–4 sentences integrating technicals + news + portfolio P&L + memory context",
  "key_risk": "The single most important risk factor right now",
  "price_target": "$XX.XX or null",
  "stop_loss": "$XX.XX or null",
  "summary": "One concise sentence for memory storage"
}`,
		ticker,
		utcTime,
		ticker,
		quote.C,
		quote.O, quote.H, quote.L,
		quote.Pc,
		dpStr, dStr,
		trend60dStr,
		portfolioCtx,
		rsiStr, rsiLabel,
		macdLine,
		macdSignal,
		macdHist,
		priceVsMA(ind.MA20, "MA20 "),
		priceVsMA(ind.MA50, "MA50 "),
		priceVsMA(ind.MA200, "MA200"),
		volumeNote,
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
		Date:        now.Local().Format("1/2/2006, 3:04:05 PM"),
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