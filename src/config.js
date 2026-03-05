const TICKERS = [
  "AAPL",
  "MSFT",
  "NVDA",
  "TSLA",
  "AMZN",
];

const HISTORY_DAYS          = 120;
const NEWS_LIMIT            = 8;
const MEMORY_LIMIT          = 40;
const MEMORY_CONTEXT_WINDOW = 8;
const NOTIFY_ON             = ["BUY", "SELL", "HOLD"];
const MIN_CONFIDENCE_TO_NOTIFY = "Medium";

module.exports = {
  TICKERS,
  HISTORY_DAYS,
  NEWS_LIMIT,
  MEMORY_LIMIT,
  MEMORY_CONTEXT_WINDOW,
  NOTIFY_ON,
  MIN_CONFIDENCE_TO_NOTIFY,
};