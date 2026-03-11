package indicators

import (
	"math"
)

// Indicators holds all computed technical indicators for a ticker.
type Indicators struct {
	RSI     *float64    `json:"rsi"`
	MACD    *MACDResult `json:"macd"`
	MA20    *float64    `json:"ma20"`
	MA50    *float64    `json:"ma50"`
	MA200   *float64    `json:"ma200"`
	AvgVol  *int64      `json:"avgVol"`
	CurVol  *float64    `json:"curVol"`
	Trend60d *float64   `json:"trend60d"`
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

func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
