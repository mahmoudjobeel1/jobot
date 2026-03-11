// Package api exposes the REST endpoints consumed by the mobile app.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"jobot/internal/memory"
	"jobot/internal/store"
)

// StartServer registers routes and starts the HTTP server in a goroutine.
func StartServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", withCORS(handleHealth))
	mux.HandleFunc("/portfolio", withCORS(handlePortfolio))
	mux.HandleFunc("/orders", withCORS(handleOrders))
	mux.HandleFunc("/stops/", withCORS(handleStops))

	fmt.Printf("  API server listening on %s\n", addr)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("  [API] server error: %v\n", err)
		}
	}()
}

func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ─── /health ─────────────────────────────────────────────────────────────────

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─── /portfolio ───────────────────────────────────────────────────────────────

type holdingResponse struct {
	store.Holding
	LatestPrediction *memory.Entry       `json:"latest_prediction,omitempty"`
	WeeklySummary    *memory.WeeklyEntry `json:"weekly_summary,omitempty"`
}

type portfolioResponse struct {
	Holdings []holdingResponse `json:"holdings"`
	Stops    []store.StopOrder `json:"stops"`
}

func handlePortfolio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	holdings := store.GetHoldings()
	resp := portfolioResponse{
		Holdings: make([]holdingResponse, len(holdings)),
		Stops:    store.GetStops(),
	}
	for i, h := range holdings {
		entries, _ := memory.LoadMemory(h.Ticker)
		var latest *memory.Entry
		if len(entries) > 0 {
			e := entries[len(entries)-1]
			latest = &e
		}
		weekly, _ := memory.LoadWeekly(h.Ticker)
		resp.Holdings[i] = holdingResponse{Holding: h, LatestPrediction: latest, WeeklySummary: weekly}
	}

	jsonResponse(w, http.StatusOK, resp)
}

// ─── /orders ─────────────────────────────────────────────────────────────────

type orderRequest struct {
	Type   string  `json:"type"`   // "buy" | "sell" | "stop_loss"
	Ticker string  `json:"ticker"`
	Qty    float64 `json:"qty"`
	Price  float64 `json:"price"` // execution price for buy/sell; stop price for stop_loss
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req orderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.Ticker = strings.ToUpper(strings.TrimSpace(req.Ticker))
	if req.Ticker == "" {
		http.Error(w, "ticker required", http.StatusBadRequest)
		return
	}
	if req.Qty <= 0 {
		http.Error(w, "qty must be > 0", http.StatusBadRequest)
		return
	}

	var err error
	switch strings.ToLower(req.Type) {
	case "buy":
		if req.Price <= 0 {
			http.Error(w, "price required for buy", http.StatusBadRequest)
			return
		}
		err = store.Buy(req.Ticker, req.Qty, req.Price)
	case "sell":
		err = store.Sell(req.Ticker, req.Qty)
	case "stop_loss":
		if req.Price <= 0 {
			http.Error(w, "price (stop price) required", http.StatusBadRequest)
			return
		}
		_, err = store.AddStop(req.Ticker, req.Qty, req.Price)
	default:
		http.Error(w, fmt.Sprintf("unknown order type %q", req.Type), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// ─── /stops/{id} and /stops/{id}/execute ─────────────────────────────────────

func handleStops(w http.ResponseWriter, r *http.Request) {
	// strip /stops/ prefix
	rest := strings.TrimPrefix(r.URL.Path, "/stops/")

	if strings.HasSuffix(rest, "/execute") {
		id := strings.TrimSuffix(rest, "/execute")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := store.ExecuteStop(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "executed"})
		return
	}

	// DELETE /stops/{id}
	if r.Method == http.MethodDelete {
		if rest == "" {
			http.Error(w, "stop id required", http.StatusBadRequest)
			return
		}
		if err := store.DeleteStop(rest); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
