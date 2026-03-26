package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	coinID    string
	basePrice float64
	marketCap float64
	volume    float64
)

func main() {
	port := envStr("MOCK_PORT", "4050")
	coinID = envStr("MOCK_COIN_ID", "labeth")
	basePrice = envFloat("MOCK_PRICE_USD", 2.50)
	marketCap = envFloat("MOCK_MARKET_CAP", 25_000_000_000)
	volume = envFloat("MOCK_24H_VOLUME", 1_000_000)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Mock CoinGecko API starting on %s (coin=%s, price=$%.2f)", addr, coinID, basePrice)
	log.Fatal(http.ListenAndServe(addr, http.HandlerFunc(router)))
}

func router(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Printf("[req] %s %s", r.Method, r.URL.String())

	w.Header().Set("Content-Type", "application/json")

	switch {
	case path == "/health":
		w.Write([]byte(`{"status":"ok"}`))

	case strings.Contains(path, "/simple/price"):
		handleSimplePrice(w)

	case strings.Contains(path, "/market_chart"):
		handleMarketChart(w, r)

	case strings.Contains(path, "/coins/"):
		handleCoinData(w)

	default:
		log.Printf("[catch-all] %s %s", r.Method, r.URL.String())
		w.Write([]byte(`{}`))
	}
}

func handleSimplePrice(w http.ResponseWriter) {
	drift := 1.0 + (rand.Float64()-0.5)*0.04
	price := basePrice * drift
	changePct := (drift - 1.0) * 100

	resp := map[string]any{
		coinID: map[string]any{
			"usd":            math.Round(price*100) / 100,
			"usd_market_cap": marketCap,
			"usd_24h_vol":    volume,
			"usd_24h_change": math.Round(changePct*100) / 100,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func handleCoinData(w http.ResponseWriter) {
	drift := 1.0 + (rand.Float64()-0.5)*0.04
	price := basePrice * drift
	changePct := (drift - 1.0) * 100

	resp := map[string]any{
		"id":     coinID,
		"symbol": coinID,
		"name":   "labETH",
		"market_data": map[string]any{
			"current_price":               map[string]any{"usd": math.Round(price*100) / 100},
			"market_cap":                  map[string]any{"usd": marketCap},
			"total_volume":                map[string]any{"usd": volume},
			"price_change_percentage_24h": math.Round(changePct*100) / 100,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func handleMarketChart(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 365
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
		days = d
	}

	now := time.Now()
	prices := make([][2]float64, days)
	caps := make([][2]float64, days)
	vols := make([][2]float64, days)

	price := basePrice
	for i := 0; i < days; i++ {
		ts := float64(now.AddDate(0, 0, -(days-1-i)).UnixMilli())
		price *= 1.0 + (rand.Float64()-0.5)*0.02
		if price < basePrice*0.5 {
			price = basePrice * 0.5
		}
		if price > basePrice*1.5 {
			price = basePrice * 1.5
		}
		prices[i] = [2]float64{ts, math.Round(price*100) / 100}
		caps[i] = [2]float64{ts, marketCap}
		vols[i] = [2]float64{ts, volume * (0.5 + rand.Float64())}
	}

	resp := map[string]any{
		"prices":        prices,
		"market_caps":   caps,
		"total_volumes": vols,
	}
	json.NewEncoder(w).Encode(resp)
}
