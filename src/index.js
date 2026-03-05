require("dotenv").config();

const cron                        = require("node-cron");
const { TICKERS }                 = require("./config");
const { fetchTickerData }         = require("./finnhub");
const { analyzeStock }            = require("./analyst");
const { notify }                  = require("./notifier");


function isMarketOpen() {
  const et  = new Date(new Date().toLocaleString("en-US", { timeZone: "America/New_York" }));
  const day = et.getDay();
  if (day === 0 || day === 6) return false;
  const hm = et.getHours() * 100 + et.getMinutes();
  return hm >= 930 && hm < 1600;
}

async function processTicker(ticker) {
  try {
    const { quote, candles, news } = await fetchTickerData(ticker);
    const result = await analyzeStock(ticker, quote, candles, news);
    await notify(result);
    return { ticker, ok: true };
  } catch (err) {
    console.error(`  [ERROR] ${ticker}: ${err.message}`);
    return { ticker, ok: false, error: err.message };
  }
}

async function runCycle() {
  console.log(`\n${"═".repeat(60)}`);
  console.log(`  STOCK AI AGENT — ${new Date().toUTCString()}`);
  console.log(`  Tickers: ${TICKERS.join(", ")}`);
  console.log(`${"═".repeat(60)}`);

  if (process.env.MARKET_HOURS_ONLY === "true" && !isMarketOpen()) {
    console.log("  [Scheduler] Market is closed — skipping.\n");
    return;
  }

  const results = [];
  for (const ticker of TICKERS) {
    const r = await processTicker(ticker);
    results.push(r);
    if (TICKERS.indexOf(ticker) < TICKERS.length - 1) {
      await new Promise(res => setTimeout(res, 1200));
    }
  }

  const ok     = results.filter(r => r.ok).length;
  const failed = results.filter(r => !r.ok);
  console.log(`\n  ✓ Cycle complete — ${ok}/${TICKERS.length} succeeded`);
  if (failed.length) {
    console.log(`  ✗ Failed: ${failed.map(r => `${r.ticker} (${r.error})`).join(", ")}`);
  }
  console.log(`${"═".repeat(60)}\n`);
}

const schedule = process.env.CRON_SCHEDULE || "*/15 9-16 * * 1-5";

console.log("╔══════════════════════════════════════════════════════════╗");
console.log("║          STOCK AI AGENT — Starting up                   ║");
console.log("╚══════════════════════════════════════════════════════════╝");
console.log(`  Schedule:  ${schedule}`);
console.log(`  Tickers:   ${TICKERS.join(", ")}\n`);

if (!process.env.FINNHUB_API_KEY)   { console.error("  ✗ FINNHUB_API_KEY missing");   process.exit(1); }
if (!process.env.ANTHROPIC_API_KEY) { console.error("  ✗ ANTHROPIC_API_KEY missing"); process.exit(1); }

(async () => {
  if (process.env.RUN_ON_START !== "false") {
    console.log("  Running initial cycle on startup…");
    await runCycle();
  }

  cron.schedule(schedule, runCycle, { timezone: "America/New_York" });
  console.log(`  Agent is running. Cron: ${schedule}\n`);
})();