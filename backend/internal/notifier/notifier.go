package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"jobot/internal/analyst"
	"jobot/internal/config"
)

var confidenceRank = map[string]int{
	"Low":    1,
	"Medium": 2,
	"High":   3,
}

var decisionEmoji = map[string]string{
	"BUY":  "🟢",
	"SELL": "🔴",
	"HOLD": "🟡",
}

func shouldNotify(result analyst.AnalysisResult) bool {
	for _, d := range config.NotifyOn {
		if d == result.Decision {
			return confidenceRank[result.Confidence] >= confidenceRank[config.MinConfidenceToNotify]
		}
	}
	return false
}

func nullableStr(s *string, prefix string) string {
	if s == nil || *s == "" || *s == "null" {
		return ""
	}
	return prefix + *s
}

func trend60dStr(ind analyst.IndicatorSnapshot) string {
	if ind.Trend60d == nil {
		return "N/A"
	}
	sign := ""
	if *ind.Trend60d > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%g%%", sign, *ind.Trend60d)
}

func rsiStr(ind analyst.IndicatorSnapshot) string {
	if ind.RSI == nil {
		return "N/A"
	}
	return fmt.Sprintf("%g", *ind.RSI)
}

func macdHistStr(ind analyst.IndicatorSnapshot) string {
	if ind.MACDHistogram == nil {
		return "N/A"
	}
	return fmt.Sprintf("%g", *ind.MACDHistogram)
}

func plStr(result analyst.AnalysisResult) string {
	if result.Qty == 0 {
		return "No position"
	}
	sign := "+"
	if result.UnrealizedPL < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s$%.2f (%s%.2f%%)",
		sign, math.Abs(result.UnrealizedPL),
		sign, math.Abs(result.UnrealizedPLPct))
}

func positionStr(result analyst.AnalysisResult) string {
	if result.Qty == 0 {
		return "No position"
	}
	return fmt.Sprintf("%.4g shares @ $%.2f", result.Qty, result.AvgCost)
}

// FormatConsole formats the result as a human-readable console block.
func FormatConsole(result analyst.AnalysisResult) string {
	emoji := decisionEmoji[result.Decision]
	var lines []string
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  ┌─ %s %s — %s (%s confidence)", emoji, result.Ticker, result.Decision, result.Confidence))
	lines = append(lines, fmt.Sprintf("  │  Price:      $%g", result.Price))
	lines = append(lines, fmt.Sprintf("  │  Position:   %s", positionStr(result)))
	lines = append(lines, fmt.Sprintf("  │  P&L:        %s", plStr(result)))
	lines = append(lines, fmt.Sprintf("  │  RSI:        %s", rsiStr(result.Indicators)))
	lines = append(lines, fmt.Sprintf("  │  MACD hist:  %s", macdHistStr(result.Indicators)))
	lines = append(lines, fmt.Sprintf("  │  Trend 60d:  %s", trend60dStr(result.Indicators)))
	if t := nullableStr(result.PriceTarget, ""); t != "" {
		lines = append(lines, fmt.Sprintf("  │  Target:     %s", t))
	}
	if s := nullableStr(result.StopLoss, ""); s != "" {
		lines = append(lines, fmt.Sprintf("  │  Stop-loss:  %s", s))
	}
	lines = append(lines, fmt.Sprintf("  │  Risk:       %s", result.KeyRisk))
	lines = append(lines, fmt.Sprintf("  │  Reasoning:  %s", result.Reasoning))
	lines = append(lines, "  └─────────────────────────────────────────")
	return strings.Join(lines, "\n")
}

type discordPayload struct {
	Content string `json:"content"`
}

func formatDiscord(result analyst.AnalysisResult) discordPayload {
	emoji := decisionEmoji[result.Decision]
	ts, _ := time.Parse(time.RFC3339, result.Timestamp)
	var parts []string
	parts = append(parts, fmt.Sprintf("## %s **%s** — `%s` _(%s confidence)_", emoji, result.Ticker, result.Decision, result.Confidence))
	parts = append(parts, fmt.Sprintf("**Price:** $%g", result.Price))
	if result.Qty > 0 {
		parts = append(parts, fmt.Sprintf("**Position:** %s | **P&L:** %s", positionStr(result), plStr(result)))
	}
	parts = append(parts, fmt.Sprintf("**RSI:** %s | **MACD Hist:** %s | **60d:** %s",
		rsiStr(result.Indicators), macdHistStr(result.Indicators), trend60dStr(result.Indicators)))
	if t := nullableStr(result.PriceTarget, ""); t != "" {
		parts = append(parts, fmt.Sprintf("**Target:** %s", t))
	}
	if s := nullableStr(result.StopLoss, ""); s != "" {
		parts = append(parts, fmt.Sprintf("**Stop-Loss:** %s", s))
	}
	parts = append(parts, fmt.Sprintf("**⚠️ Risk:** %s", result.KeyRisk))
	parts = append(parts, fmt.Sprintf("> %s", result.Reasoning))
	parts = append(parts, fmt.Sprintf("_%s_", ts.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")))
	return discordPayload{Content: strings.Join(parts, "\n")}
}

func formatTelegram(result analyst.AnalysisResult) string {
	emoji := decisionEmoji[result.Decision]
	ts, _ := time.Parse(time.RFC3339, result.Timestamp)
	var parts []string
	parts = append(parts, fmt.Sprintf("%s <b>%s</b> — <code>%s</code> (%s)", emoji, result.Ticker, result.Decision, result.Confidence))
	parts = append(parts, fmt.Sprintf("💵 <b>Price:</b> $%g", result.Price))
	if result.Qty > 0 {
		parts = append(parts, fmt.Sprintf("📊 <b>Position:</b> %s | <b>P&L:</b> %s", positionStr(result), plStr(result)))
	}
	parts = append(parts, fmt.Sprintf("📈 RSI: %s | MACD: %s | 60d: %s",
		rsiStr(result.Indicators), macdHistStr(result.Indicators), trend60dStr(result.Indicators)))
	if t := nullableStr(result.PriceTarget, ""); t != "" {
		parts = append(parts, fmt.Sprintf("🎯 <b>Target:</b> %s", t))
	}
	if s := nullableStr(result.StopLoss, ""); s != "" {
		parts = append(parts, fmt.Sprintf("🛑 <b>Stop-Loss:</b> %s", s))
	}
	parts = append(parts, fmt.Sprintf("⚠️ <b>Risk:</b> %s", result.KeyRisk))
	parts = append(parts, result.Reasoning)
	parts = append(parts, fmt.Sprintf("<i>%s</i>", ts.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")))
	return strings.Join(parts, "\n")
}

func sendDiscord(result analyst.AnalysisResult) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}
	payload, err := json.Marshal(formatDiscord(result))
	if err != nil {
		fmt.Printf("  [Discord] Marshal error: %v\n", err)
		return
	}
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("  [Discord] Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("  [Discord] HTTP %d\n", resp.StatusCode)
		return
	}
	fmt.Printf("  [Discord] Sent %s\n", result.Ticker)
}

func sendTelegram(result analyst.AnalysisResult) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if botToken == "" || chatID == "" {
		return
	}
	payload, err := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       formatTelegram(result),
		"parse_mode": "HTML",
	})
	if err != nil {
		fmt.Printf("  [Telegram] Marshal error: %v\n", err)
		return
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Printf("  [Telegram] Failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("  [Telegram] HTTP %d\n", resp.StatusCode)
		return
	}
	fmt.Printf("  [Telegram] Sent %s\n", result.Ticker)
}

// Notify prints the result to the console and, if eligible, sends to Discord and Telegram.
func Notify(result analyst.AnalysisResult) {
	fmt.Println(FormatConsole(result))
	if !shouldNotify(result) {
		fmt.Printf("  [Notifier] Skipped — %s / %s below threshold\n", result.Decision, result.Confidence)
		return
	}
	// Send concurrently, best-effort
	done := make(chan struct{}, 2)
	go func() { sendDiscord(result); done <- struct{}{} }()
	go func() { sendTelegram(result); done <- struct{}{} }()
	<-done
	<-done
}