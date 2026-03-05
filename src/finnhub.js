// Quote + News  →  Finnhub free tier
// Candles (OHLCV) →  Yahoo Finance (free, no API key needed)

const { HISTORY_DAYS, NEWS_LIMIT } = require("./config");

const FINNHUB_BASE = "https://finnhub.io/api/v1";
const DELAY_MS     = 500;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function getKey() {
  const key = process.env.FINNHUB_API_KEY;
  if (!key) throw new Error("FINNHUB_API_KEY is not set in .env");
  return key;
}


async function fhGet(path, params = {}) {
  const url = new URL(`${FINNHUB_BASE}${path}`);
  url.searchParams.set("token", getKey());
  for (const [k, v] of Object.entries(params)) url.searchParams.set(k, String(v));

  const res = await fetch(url.toString(), { signal: AbortSignal.timeout(15_000) });
  if (!res.ok) throw new Error(`Finnhub HTTP ${res.status} on ${path}`);
  return res.json();
}

// ─── Yahoo Finance helper (candles) ──────────────────────────────────────────

async function fetchCandles(ticker) {
  // Yahoo Finance v8 chart endpoint — free, no auth required
  const url = `https://query1.finance.yahoo.com/v8/finance/chart/${ticker}` +
              `?interval=1d&range=${Math.ceil(HISTORY_DAYS / 30)}mo`;

  const res = await fetch(url, {
    headers: { "User-Agent": "Mozilla/5.0" },
    signal: AbortSignal.timeout(15_000),
  });

  if (!res.ok) throw new Error(`Yahoo Finance HTTP ${res.status} for ${ticker}`);

  const json   = await res.json();
  const result = json?.chart?.result?.[0];

  if (!result) throw new Error(`No candle data from Yahoo Finance for ${ticker}`);

  const timestamps = result.timestamp ?? [];
  const ohlcv      = result.indicators?.quote?.[0] ?? {};

  return {
    t: timestamps,
    o: ohlcv.open   ?? [],
    h: ohlcv.high   ?? [],
    l: ohlcv.low    ?? [],
    c: ohlcv.close  ?? [],
    v: ohlcv.volume ?? [],
    s: "ok",
  };
}

async function fetchQuote(ticker) {
  const data = await fhGet("/quote", { symbol: ticker });
  if (!data || data.c === 0) throw new Error(`No quote data for ${ticker}`);
  return data;
}

async function fetchNews(ticker) {
  const fmt     = (d) => d.toISOString().split("T")[0];
  const today   = new Date();
  const weekAgo = new Date(Date.now() - 7 * 86400_000);
  const data    = await fhGet("/company-news", { symbol: ticker, from: fmt(weekAgo), to: fmt(today) });

  return (data ?? [])
    .slice(0, NEWS_LIMIT)
    .map(({ headline, summary, source, datetime, url }) => ({
      headline,
      summary: summary?.slice(0, 200),
      source,
      date: new Date(datetime * 1000).toLocaleDateString(),
      url,
    }));
}

async function fetchTickerData(ticker) {
  console.log(`  [Finnhub] Fetching ${ticker}…`);

  const quote = await fetchQuote(ticker);
  await sleep(DELAY_MS);

  const candles = await fetchCandles(ticker);
  await sleep(DELAY_MS);

  let news = [];
  try {
    news = await fetchNews(ticker);
    await sleep(DELAY_MS);
  } catch {
    // news is non-critical
  }

  return { ticker, quote, candles, news };
}

module.exports = { fetchQuote, fetchCandles, fetchNews, fetchTickerData };