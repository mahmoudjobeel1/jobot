package finnhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"jobot/internal/config"
	"jobot/internal/indicators"
)

// Quote holds data from Finnhub's /quote endpoint.
type Quote struct {
	C  float64 `json:"c"`  // current price
	D  float64 `json:"d"`  // change
	Dp float64 `json:"dp"` // change percent
	H  float64 `json:"h"`  // high
	L  float64 `json:"l"`  // low
	O  float64 `json:"o"`  // open
	Pc float64 `json:"pc"` // previous close
	T  int64   `json:"t"`  // timestamp
}

// NewsItem holds a single news article from Finnhub.
type NewsItem struct {
	Headline string `json:"headline"`
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Date     string `json:"date"`
	URL      string `json:"url"`
}

// TickerData is the aggregate result for one ticker.
type TickerData struct {
	Ticker  string
	Quote   Quote
	Candles indicators.Candles
	News    []NewsItem
}

// yahoo Finance response shape
type yahooResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

type finnhubNewsRaw struct {
	Headline string `json:"headline"`
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Datetime int64  `json:"datetime"`
	URL      string `json:"url"`
}

const (
	finnhubBase = "https://finnhub.io/api/v1"
	delayMS     = 500 * time.Millisecond
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

func getKey() (string, error) {
	key := os.Getenv("FINNHUB_API_KEY")
	if key == "" {
		return "", fmt.Errorf("FINNHUB_API_KEY is not set in .env")
	}
	return key, nil
}

func fhGet(path string, params map[string]string) ([]byte, error) {
	key, err := getKey()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(finnhubBase + path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("token", key)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub HTTP %d on %s", resp.StatusCode, path)
	}
	return io.ReadAll(resp.Body)
}


// FetchQuote fetches the real-time quote for a ticker from Finnhub.
func FetchQuote(ticker string) (Quote, error) {
	body, err := fhGet("/quote", map[string]string{"symbol": ticker})
	if err != nil {
		return Quote{}, err
	}
	var q Quote
	if err := json.Unmarshal(body, &q); err != nil {
		return Quote{}, err
	}
	if q.C == 0 {
		return Quote{}, fmt.Errorf("no quote data for %s", ticker)
	}
	return q, nil
}

// FetchCandles fetches OHLCV candle data from Yahoo Finance.
func FetchCandles(ticker string) (indicators.Candles, error) {
	months := (config.HistoryDays + 29) / 30 // ceil division
	rawURL := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=%dmo",
		ticker, months,
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return indicators.Candles{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return indicators.Candles{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return indicators.Candles{}, fmt.Errorf("Yahoo Finance HTTP %d for %s", resp.StatusCode, ticker)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return indicators.Candles{}, err
	}

	var yr yahooResponse
	if err := json.Unmarshal(body, &yr); err != nil {
		return indicators.Candles{}, err
	}
	if len(yr.Chart.Result) == 0 {
		return indicators.Candles{}, fmt.Errorf("no candle data from Yahoo Finance for %s", ticker)
	}

	r := yr.Chart.Result[0]
	ohlcv := struct {
		Open   []float64
		High   []float64
		Low    []float64
		Close  []float64
		Volume []float64
	}{}
	if len(r.Indicators.Quote) > 0 {
		q := r.Indicators.Quote[0]
		ohlcv.Open = q.Open
		ohlcv.High = q.High
		ohlcv.Low = q.Low
		ohlcv.Close = q.Close
		ohlcv.Volume = q.Volume
	}

	return indicators.Candles{
		T: r.Timestamp,
		O: ohlcv.Open,
		H: ohlcv.High,
		L: ohlcv.Low,
		C: ohlcv.Close,
		V: ohlcv.Volume,
		S: "ok",
	}, nil
}

// FetchNews fetches recent company news from Finnhub.
func FetchNews(ticker string) ([]NewsItem, error) {
	today := time.Now()
	weekAgo := today.AddDate(0, 0, -7)
	fmtDate := func(t time.Time) string { return t.Format("2006-01-02") }

	body, err := fhGet("/company-news", map[string]string{
		"symbol": ticker,
		"from":   fmtDate(weekAgo),
		"to":     fmtDate(today),
	})
	if err != nil {
		return nil, err
	}

	var raw []finnhubNewsRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	limit := config.NewsLimit
	if len(raw) < limit {
		limit = len(raw)
	}
	raw = raw[:limit]

	items := make([]NewsItem, 0, len(raw))
	for _, n := range raw {
		summary := n.Summary
		if len(summary) > 200 {
			summary = summary[:200]
		}
		items = append(items, NewsItem{
			Headline: n.Headline,
			Summary:  summary,
			Source:   n.Source,
			Date:     time.Unix(n.Datetime, 0).Format("1/2/2006"),
			URL:      n.URL,
		})
	}
	return items, nil
}

// FetchTickerData fetches quote, candles, and news for a single ticker.
func FetchTickerData(ticker string) (TickerData, error) {
	fmt.Printf("  [Finnhub] Fetching %s...\n", ticker)

	quote, err := FetchQuote(ticker)
	if err != nil {
		return TickerData{}, fmt.Errorf("quote: %w", err)
	}
	time.Sleep(delayMS)

	candles, err := FetchCandles(ticker)
	if err != nil {
		return TickerData{}, fmt.Errorf("candles: %w", err)
	}
	time.Sleep(delayMS)

	var news []NewsItem
	news, _ = FetchNews(ticker) // best-effort; ignore error
	time.Sleep(delayMS)

	return TickerData{
		Ticker:  ticker,
		Quote:   quote,
		Candles: candles,
		News:    news,
	}, nil
}
