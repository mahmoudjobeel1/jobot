const { NOTIFY_ON, MIN_CONFIDENCE_TO_NOTIFY } = require("./config");

const CONFIDENCE_RANK = { Low: 1, Medium: 2, High: 3 };
const DECISION_EMOJI  = { BUY: "🟢", SELL: "🔴", HOLD: "🟡" };

function shouldNotify(result) {
  if (!NOTIFY_ON.includes(result.decision)) return false;
  return CONFIDENCE_RANK[result.confidence] >= CONFIDENCE_RANK[MIN_CONFIDENCE_TO_NOTIFY];
}

function formatConsole(result) {
  const { ticker, price, decision, confidence, reasoning, key_risk, price_target, stop_loss, indicators } = result;
  const emoji = DECISION_EMOJI[decision];
  return [
    ``,
    `  ┌─ ${emoji} ${ticker} — ${decision} (${confidence} confidence)`,
    `  │  Price:      $${price}`,
    `  │  RSI:        ${indicators.rsi ?? "N/A"}`,
    `  │  MACD hist:  ${indicators.macdHistogram ?? "N/A"}`,
    `  │  Trend 60d:  ${indicators.trend60d !== null ? `${indicators.trend60d > 0 ? "+" : ""}${indicators.trend60d}%` : "N/A"}`,
    price_target ? `  │  Target:     ${price_target}` : null,
    stop_loss    ? `  │  Stop-loss:  ${stop_loss}`    : null,
    `  │  Risk:       ${key_risk}`,
    `  │  Reasoning:  ${reasoning}`,
    `  └─────────────────────────────────────────`,
  ].filter(Boolean).join("\n");
}

function formatDiscord(result) {
  const { ticker, price, decision, confidence, reasoning, key_risk, price_target, stop_loss, indicators, timestamp } = result;
  const emoji = DECISION_EMOJI[decision];
  return {
    content: [
      `## ${emoji} **${ticker}** — \`${decision}\` _(${confidence} confidence)_`,
      `**Price:** $${price}`,
      `**RSI:** ${indicators.rsi ?? "N/A"} | **MACD Hist:** ${indicators.macdHistogram ?? "N/A"} | **60d:** ${indicators.trend60d !== null ? `${indicators.trend60d > 0 ? "+" : ""}${indicators.trend60d}%` : "N/A"}`,
      price_target ? `**Target:** ${price_target}` : null,
      stop_loss    ? `**Stop-Loss:** ${stop_loss}`  : null,
      `**⚠️ Risk:** ${key_risk}`,
      `> ${reasoning}`,
      `_${new Date(timestamp).toUTCString()}_`,
    ].filter(Boolean).join("\n"),
  };
}

function formatTelegram(result) {
  const { ticker, price, decision, confidence, reasoning, key_risk, price_target, stop_loss, indicators, timestamp } = result;
  const emoji = DECISION_EMOJI[decision];
  return [
    `${emoji} <b>${ticker}</b> — <code>${decision}</code> (${confidence})`,
    `💵 <b>Price:</b> $${price}`,
    `📈 RSI: ${indicators.rsi ?? "N/A"} | MACD: ${indicators.macdHistogram ?? "N/A"} | 60d: ${indicators.trend60d !== null ? `${indicators.trend60d > 0 ? "+" : ""}${indicators.trend60d}%` : "N/A"}`,
    price_target ? `🎯 <b>Target:</b> ${price_target}` : null,
    stop_loss    ? `🛑 <b>Stop-Loss:</b> ${stop_loss}` : null,
    `⚠️ <b>Risk:</b> ${key_risk}`,
    reasoning,
    `<i>${new Date(timestamp).toUTCString()}</i>`,
  ].filter(Boolean).join("\n");
}

async function sendDiscord(result) {
  const webhookUrl = process.env.DISCORD_WEBHOOK_URL;
  if (!webhookUrl) return;
  try {
    const res = await fetch(webhookUrl, {
      method:  "POST",
      headers: { "Content-Type": "application/json" },
      body:    JSON.stringify(formatDiscord(result)),
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    console.log(`  [Discord] Sent ${result.ticker} ✓`);
  } catch (err) {
    console.error(`  [Discord] Failed:`, err.message);
  }
}

async function sendTelegram(result) {
  const botToken = process.env.TELEGRAM_BOT_TOKEN;
  const chatId   = process.env.TELEGRAM_CHAT_ID;
  if (!botToken || !chatId) return;
  try {
    const res = await fetch(`https://api.telegram.org/bot${botToken}/sendMessage`, {
      method:  "POST",
      headers: { "Content-Type": "application/json" },
      body:    JSON.stringify({ chat_id: chatId, text: formatTelegram(result), parse_mode: "HTML" }),
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    console.log(`  [Telegram] Sent ${result.ticker} ✓`);
  } catch (err) {
    console.error(`  [Telegram] Failed:`, err.message);
  }
}

async function notify(result) {
  console.log(formatConsole(result));
  if (!shouldNotify(result)) {
    console.log(`  [Notifier] Skipped — ${result.decision} / ${result.confidence} below threshold`);
    return;
  }
  await Promise.allSettled([
    sendDiscord(result),
    sendTelegram(result),
    // sendSlack(result),
  ]);
}

module.exports = { notify };