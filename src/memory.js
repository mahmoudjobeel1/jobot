const fs   = require("fs");
const path = require("path");
const { MEMORY_LIMIT } = require("./config");

const DATA_DIR = path.join(__dirname, "..", "data");

function ensureDataDir() {
  if (!fs.existsSync(DATA_DIR)) fs.mkdirSync(DATA_DIR, { recursive: true });
}

function filePath(ticker) {
  return path.join(DATA_DIR, `${ticker.toUpperCase()}.json`);
}

function loadMemory(ticker) {
  ensureDataDir();
  const fp = filePath(ticker);
  if (!fs.existsSync(fp)) return [];
  try {
    return JSON.parse(fs.readFileSync(fp, "utf8"));
  } catch {
    return [];
  }
}

function appendMemory(ticker, entry) {
  ensureDataDir();
  const updated = [...loadMemory(ticker), entry].slice(-MEMORY_LIMIT);
  fs.writeFileSync(filePath(ticker), JSON.stringify(updated, null, 2), "utf8");
}

function buildMemoryContext(ticker, limit) {
  const entries = loadMemory(ticker).slice(-limit);
  if (entries.length === 0) return "No prior analysis history for this ticker.";
  return entries.map(e =>
    `[${e.date}] Decision: ${e.decision} (${e.confidence}) @ $${e.price}` +
    ` | RSI: ${e.rsi ?? "N/A"} | MACD hist: ${e.macdHistogram ?? "N/A"}` +
    ` | Summary: ${e.summary}`
  ).join("\n");
}

module.exports = { loadMemory, appendMemory, buildMemoryContext };