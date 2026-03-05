/**
 * Relative Strength Index (RSI)
 * @param {number[]} closes - Array of closing prices (oldest → newest)
 * @param {number}   period - Lookback period (default 14)
 * @returns {number|null}
 */
export function calcRSI(closes, period = 14) {
  if (closes.length < period + 1) return null;

  let gains = 0, losses = 0;
  for (let i = 1; i <= period; i++) {
    const delta = closes[i] - closes[i - 1];
    if (delta > 0) gains += delta;
    else losses -= delta;
  }

  let avgGain = gains / period;
  let avgLoss = losses / period;

  for (let i = period + 1; i < closes.length; i++) {
    const delta = closes[i] - closes[i - 1];
    avgGain = (avgGain * (period - 1) + Math.max(delta, 0)) / period;
    avgLoss = (avgLoss * (period - 1) + Math.max(-delta, 0)) / period;
  }

  if (avgLoss === 0) return 100;
  return parseFloat((100 - 100 / (1 + avgGain / avgLoss)).toFixed(2));
}

/**
 * Exponential Moving Average (EMA)
 * @param {number[]} data
 * @param {number}   period
 * @returns {number|null}
 */
export function calcEMA(data, period) {
  if (data.length < period) return null;
  const k = 2 / (period + 1);
  let ema = data.slice(0, period).reduce((a, b) => a + b, 0) / period;
  for (let i = period; i < data.length; i++) {
    ema = data[i] * k + ema * (1 - k);
  }
  return parseFloat(ema.toFixed(4));
}

/**
 * MACD — returns { macd, signal, histogram }
 * Uses 12/26 EMA for MACD line and 9 EMA for signal
 * @param {number[]} closes
 * @returns {{ macd: number, signal: number, histogram: number }|null}
 */
export function calcMACD(closes) {
  if (closes.length < 35) return null;

  // Build the MACD series across all available data points
  const macdSeries = [];
  for (let i = 26; i <= closes.length; i++) {
    const slice = closes.slice(0, i);
    const ema12 = calcEMA(slice, 12);
    const ema26 = calcEMA(slice, 26);
    if (ema12 !== null && ema26 !== null) {
      macdSeries.push(ema12 - ema26);
    }
  }

  const macd = macdSeries[macdSeries.length - 1];
  const signal = calcEMA(macdSeries, 9);
  const histogram = signal !== null ? parseFloat((macd - signal).toFixed(4)) : null;

  return {
    macd: parseFloat(macd.toFixed(4)),
    signal,
    histogram,
  };
}

/**
 * Simple Moving Average (SMA)
 * @param {number[]} closes
 * @param {number}   period
 * @returns {number|null}
 */
export function calcSMA(closes, period) {
  if (closes.length < period) return null;
  const slice = closes.slice(-period);
  return parseFloat((slice.reduce((a, b) => a + b, 0) / period).toFixed(2));
}

/**
 * Average volume over a period
 * @param {number[]} volumes
 * @param {number}   period
 * @returns {number|null}
 */
export function calcAvgVolume(volumes, period = 20) {
  if (volumes.length < period) return null;
  const slice = volumes.slice(-period);
  return Math.round(slice.reduce((a, b) => a + b, 0) / period);
}

/**
 * Compute all indicators for a given candle dataset
 * @param {{ c: number[], v: number[] }} candles
 * @returns {object} All computed indicators
 */
export function computeAll(candles) {
  const closes  = candles.c ?? [];
  const volumes = candles.v ?? [];

  return {
    rsi:    calcRSI(closes),
    macd:   calcMACD(closes),
    ma20:   calcSMA(closes, 20),
    ma50:   calcSMA(closes, 50),
    ma200:  calcSMA(closes, 200),
    avgVol: calcAvgVolume(volumes, 20),
    curVol: volumes.at(-1) ?? null,
    // 60-day trend: percentage change from first to last close
    trend60d: closes.length >= 60
      ? parseFloat(((closes.at(-1) - closes.at(-60)) / closes.at(-60) * 100).toFixed(2))
      : null,
  };
}