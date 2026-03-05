const Anthropic = require("@anthropic-ai/sdk");
const { computeAll } = require("./indicators");
const { buildMemoryContext, appendMemory } = require("./memory");
const { MEMORY_CONTEXT_WINDOW } = require("./config");

const anthropic = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });

function buildPrompt(ticker, quote, candles, news, memoryContext) {
  const ind    = computeAll(candles);
  const closes = candles.c ?? [];

  const priceVsMA = (ma, label) => {
    if (!ma) return `${label}: N/A`;
    const diff = ((quote.c - ma) / ma * 100).toFixed(2);
    return `${label}: $${ma} (price is ${diff >= 0 ? "+" : ""}${diff}% ${diff >= 0 ? "above" : "below"})`;
  };

  const newsBlock = news.length
    ? news.map(n => `  • [${n.date}] ${n.headline} (${n.source})`).join("\n")
    : "  No recent news found.";

  const volumeNote = ind.avgVol && ind.curVol
    ? `${ind.curVol.toLocaleString()} vs 20-day avg ${ind.avgVol.toLocaleString()} ` +
      `(${((ind.curVol / ind.avgVol - 1) * 100).toFixed(1)}% ${ind.curVol > ind.avgVol ? "above" : "below"} avg)`
    : "N/A";

  return `You are a professional quantitative stock analyst. Analyze ${ticker} using all the data below and return a clear trading decision.

═══ LIVE MARKET DATA — ${new Date().toUTCString()} ═══
Ticker:         ${ticker}
Current Price:  $${quote.c}
Open/High/Low:  $${quote.o} / $${quote.h} / $${quote.l}
Prev Close:     $${quote.pc}
Daily Change:   ${quote.dp?.toFixed(2)}% ($${quote.d?.toFixed(2)})
60-day Trend:   ${ind.trend60d !== null ? `${ind.trend60d > 0 ? "+" : ""}${ind.trend60d}%` : "N/A"}

═══ TECHNICAL INDICATORS ═══
RSI (14):       ${ind.rsi ?? "N/A"}${ind.rsi > 70 ? " ← OVERBOUGHT" : ind.rsi < 30 ? " ← OVERSOLD" : ""}
MACD Line:      ${ind.macd?.macd ?? "N/A"}
MACD Signal:    ${ind.macd?.signal ?? "N/A"}
MACD Histogram: ${ind.macd?.histogram ?? "N/A"}
${priceVsMA(ind.ma20,  "MA20 ")}
${priceVsMA(ind.ma50,  "MA50 ")}
${priceVsMA(ind.ma200, "MA200")}
Volume:         ${volumeNote}

═══ RECENT NEWS & SENTIMENT ═══
${newsBlock}

═══ YOUR ANALYSIS HISTORY (accumulated memory) ═══
${memoryContext}

Respond with ONLY a valid JSON object — no explanation, no markdown fences:
{
  "decision": "BUY" | "SELL" | "HOLD",
  "confidence": "Low" | "Medium" | "High",
  "reasoning": "2–4 sentences integrating technicals + news + memory context",
  "key_risk": "The single most important risk factor right now",
  "price_target": "$XX.XX or null",
  "stop_loss": "$XX.XX or null",
  "summary": "One concise sentence for memory storage"
}`;
}

async function analyzeStock(ticker, quote, candles, news) {
  const ind           = computeAll(candles);
  const memoryContext = buildMemoryContext(ticker, MEMORY_CONTEXT_WINDOW);
  const prompt        = buildPrompt(ticker, quote, candles, news, memoryContext);

  console.log(`  [Claude] Analyzing ${ticker}…`);

  const message = await anthropic.messages.create({
    model:      "claude-sonnet-4-20250514",
    max_tokens: 1024,
    messages:   [{ role: "user", content: prompt }],
  });

  const rawText = message.content
    .filter(b => b.type === "text")
    .map(b => b.text)
    .join("")
    .replace(/```json|```/g, "")
    .trim();

  let parsed;
  try {
    parsed = JSON.parse(rawText);
  } catch {
    throw new Error(`Claude returned non-JSON for ${ticker}: ${rawText.slice(0, 200)}`);
  }

  const result = {
    ticker,
    timestamp:    new Date().toISOString(),
    date:         new Date().toLocaleString(),
    price:        quote.c,
    decision:     parsed.decision,
    confidence:   parsed.confidence,
    reasoning:    parsed.reasoning,
    key_risk:     parsed.key_risk,
    price_target: parsed.price_target ?? null,
    stop_loss:    parsed.stop_loss ?? null,
    summary:      parsed.summary,
    indicators: {
      rsi:           ind.rsi,
      macdHistogram: ind.macd?.histogram ?? null,
      ma20:          ind.ma20,
      ma50:          ind.ma50,
      ma200:         ind.ma200,
      trend60d:      ind.trend60d,
    },
  };

  appendMemory(ticker, {
    date:          result.date,
    decision:      result.decision,
    confidence:    result.confidence,
    price:         result.price,
    rsi:           result.indicators.rsi,
    macdHistogram: result.indicators.macdHistogram,
    summary:       result.summary,
    priceTarget:   result.price_target,
    stopLoss:      result.stop_loss,
  });

  return result;
}

module.exports = { analyzeStock };