package indicators

import (
	"math"
)

// Indicators holds all computed technical indicators for a ticker.
type Indicators struct {
	RSI      *float64    `json:"rsi"`
	MACD     *MACDResult `json:"macd"`
	MA20     *float64    `json:"ma20"`
	MA50     *float64    `json:"ma50"`
	MA200    *float64    `json:"ma200"`
	AvgVol   *int64      `json:"avgVol"`
	CurVol   *float64    `json:"curVol"`
	Trend60d *float64    `json:"trend60d"`
	ATR14    *float64    `json:"atr14"`
	ADX14    *float64    `json:"adx14"` // Average Directional Index (trend strength 0–100; >25 = trending)
}

// Candles holds the OHLCV data from Yahoo Finance.
type Candles struct {
	T []int64   `json:"t"`
	O []float64 `json:"o"`
	H []float64 `json:"h"`
	L []float64 `json:"l"`
	C []float64 `json:"c"`
	V []float64 `json:"v"`
	S string    `json:"s"`
}

// CalcRSI computes the Wilder RSI for the given period (default 14).
// Returns nil when there is insufficient data.
func CalcRSI(closes []float64, period int) *float64 {
	if len(closes) < period+1 {
		return nil
	}
	var gains, losses float64
	for i := 1; i <= period; i++ {
		delta := closes[i] - closes[i-1]
		if delta > 0 {
			gains += delta
		} else {
			losses -= delta
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	for i := period + 1; i < len(closes); i++ {
		delta := closes[i] - closes[i-1]
		avgGain = (avgGain*float64(period-1) + math.Max(delta, 0)) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + math.Max(-delta, 0)) / float64(period)
	}
	var rsi float64
	if avgLoss == 0 {
		rsi = 100
	} else {
		rsi = 100 - 100/(1+avgGain/avgLoss)
	}
	rsi = roundTo(rsi, 2)
	return &rsi
}

// CalcEMA computes the exponential moving average.
// Returns nil when there is insufficient data.
func CalcEMA(data []float64, period int) *float64 {
	if len(data) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	var sum float64
	for _, v := range data[:period] {
		sum += v
	}
	ema := sum / float64(period)
	for i := period; i < len(data); i++ {
		ema = data[i]*k + ema*(1-k)
	}
	result := roundTo(ema, 4)
	return &result
}

// MACDResult holds the MACD line, signal, and histogram values.
type MACDResult struct {
	MACD      float64  `json:"macd"`
	Signal    *float64 `json:"signal"`
	Histogram *float64 `json:"histogram"`
}

// CalcMACD computes the MACD indicator (12/26/9).
// Returns nil when there is insufficient data (fewer than 35 closes).
func CalcMACD(closes []float64) *MACDResult {
	if len(closes) < 35 {
		return nil
	}
	var macdSeries []float64
	for i := 26; i <= len(closes); i++ {
		slice := closes[:i]
		ema12 := CalcEMA(slice, 12)
		ema26 := CalcEMA(slice, 26)
		if ema12 != nil && ema26 != nil {
			macdSeries = append(macdSeries, *ema12-*ema26)
		}
	}
	macd := roundTo(macdSeries[len(macdSeries)-1], 4)
	signal := CalcEMA(macdSeries, 9)
	var histogram *float64
	if signal != nil {
		h := roundTo(macd-*signal, 4)
		histogram = &h
	}
	return &MACDResult{
		MACD:      macd,
		Signal:    signal,
		Histogram: histogram,
	}
}

// CalcSMA computes the simple moving average over the last `period` values.
// Returns nil when there is insufficient data.
func CalcSMA(closes []float64, period int) *float64 {
	if len(closes) < period {
		return nil
	}
	slice := closes[len(closes)-period:]
	var sum float64
	for _, v := range slice {
		sum += v
	}
	result := roundTo(sum/float64(period), 2)
	return &result
}

// CalcAvgVolume computes the average volume over the last `period` bars.
// Returns nil when there is insufficient data.
func CalcAvgVolume(volumes []float64, period int) *int64 {
	if len(volumes) < period {
		return nil
	}
	slice := volumes[len(volumes)-period:]
	var sum float64
	for _, v := range slice {
		sum += v
	}
	result := int64(math.Round(sum / float64(period)))
	return &result
}

// CalcATR computes the Average True Range using Wilder's smoothing method.
// Returns nil when there is insufficient data.
func CalcATR(highs, lows, closes []float64, period int) *float64 {
	n := len(closes)
	if n < period+1 || len(highs) != n || len(lows) != n {
		return nil
	}
	trs := make([]float64, n-1)
	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hpc := math.Abs(highs[i] - closes[i-1])
		lpc := math.Abs(lows[i] - closes[i-1])
		trs[i-1] = math.Max(hl, math.Max(hpc, lpc))
	}
	if len(trs) < period {
		return nil
	}
	var seed float64
	for i := 0; i < period; i++ {
		seed += trs[i]
	}
	atr := seed / float64(period)
	for i := period; i < len(trs); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}
	result := roundTo(atr, 4)
	return &result
}

// CalcADX computes the Average Directional Index (trend strength) using Wilder's smoothing.
// ADX > 25 indicates a trending market; ADX < 20 indicates a ranging market.
// Returns nil when there is insufficient data.
func CalcADX(candles Candles, period int) *float64 {
	n := len(candles.C)
	if n < period*2+2 || len(candles.H) != n || len(candles.L) != n {
		return nil
	}

	tr := make([]float64, n)
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)
	for i := 1; i < n; i++ {
		hl := candles.H[i] - candles.L[i]
		hpc := math.Abs(candles.H[i] - candles.C[i-1])
		lpc := math.Abs(candles.L[i] - candles.C[i-1])
		tr[i] = math.Max(hl, math.Max(hpc, lpc))
		up := candles.H[i] - candles.H[i-1]
		down := candles.L[i-1] - candles.L[i]
		if up > down && up > 0 {
			plusDM[i] = up
		}
		if down > up && down > 0 {
			minusDM[i] = down
		}
	}

	// Seed Wilder smoothed TR/DM using first `period` bars
	aTR, aPDM, aNDM := 0.0, 0.0, 0.0
	for i := 1; i <= period; i++ {
		aTR += tr[i]
		aPDM += plusDM[i]
		aNDM += minusDM[i]
	}

	// Compute DX for each bar and smooth into ADX
	dx := make([]float64, n)
	calcDX := func(tr, pdm, ndm float64) float64 {
		if tr == 0 {
			return 0
		}
		pdi, ndi := pdm/tr*100, ndm/tr*100
		if pdi+ndi == 0 {
			return 0
		}
		return math.Abs(pdi-ndi) / (pdi + ndi) * 100
	}
	dx[period] = calcDX(aTR, aPDM, aNDM)
	for i := period + 1; i < n; i++ {
		aTR = aTR - aTR/float64(period) + tr[i]
		aPDM = aPDM - aPDM/float64(period) + plusDM[i]
		aNDM = aNDM - aNDM/float64(period) + minusDM[i]
		dx[i] = calcDX(aTR, aPDM, aNDM)
	}

	// Seed ADX from the first `period` DX values
	adxSeed := 0.0
	for i := period; i < period*2 && i < n; i++ {
		adxSeed += dx[i]
	}
	adx := adxSeed / float64(period)
	for i := period * 2; i < n; i++ {
		adx = (adx*float64(period-1) + dx[i]) / float64(period)
	}
	result := roundTo(adx, 2)
	return &result
}

// ComputeAll computes all technical indicators from the candle data.
func ComputeAll(candles Candles) Indicators {
	closes := candles.C
	volumes := candles.V

	ind := Indicators{
		RSI:    CalcRSI(closes, 14),
		MACD:   CalcMACD(closes),
		MA20:   CalcSMA(closes, 20),
		MA50:   CalcSMA(closes, 50),
		MA200:  CalcSMA(closes, 200),
		AvgVol: CalcAvgVolume(volumes, 20),
		ATR14:  CalcATR(candles.H, candles.L, closes, 14),
		ADX14:  CalcADX(candles, 14),
	}

	if len(volumes) > 0 {
		v := volumes[len(volumes)-1]
		ind.CurVol = &v
	}

	if len(closes) >= 60 {
		last := closes[len(closes)-1]
		prev := closes[len(closes)-60]
		trend := roundTo((last-prev)/prev*100, 2)
		ind.Trend60d = &trend
	}

	return ind
}

// Regime classifies the current market regime for a ticker.
type Regime string

const (
	RegimeTrending Regime = "TRENDING"  // ADX > 25 and price > MA200
	RegimeBearish  Regime = "BEARISH"   // price < MA200 with declining MA50
	RegimeSideways Regime = "SIDEWAYS"  // ADX < 20 or no clear trend
)

// ClassifyRegime returns the current regime based on ADX14 and MA200.
func ClassifyRegime(ind Indicators, currentPrice float64) Regime {
	if ind.ADX14 == nil || ind.MA200 == nil {
		return RegimeSideways
	}
	aboveMA200 := currentPrice > *ind.MA200
	trending := *ind.ADX14 > 25

	if trending && aboveMA200 {
		return RegimeTrending
	}
	if !aboveMA200 {
		if ind.MA50 != nil && *ind.MA50 < *ind.MA200 {
			return RegimeBearish
		}
	}
	return RegimeSideways
}

func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
