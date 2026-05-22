package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	ordersTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "business_orders_total",
			Help: "Total number of orders created",
		},
	)

	orderAmount = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "business_order_amount",
			Help:    "Order amount distribution",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000},
		},
	)
)

func instrumentHandler(endpoint string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		handler(rec, r)
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(r.Method, endpoint, fmt.Sprintf("%d", rec.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(delay)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"hello, world!"}`)
}

func handleSlow(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(500+rand.Intn(2000)) * time.Millisecond
	time.Sleep(delay)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"slow response"}`)
}

func handleError(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(rand.Intn(50)) * time.Millisecond
	time.Sleep(delay)

	if rand.Float64() < 0.3 {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal server error"}`)
		return
	}
	if rand.Float64() < 0.2 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"bad request"}`)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"ok"}`)
}

func handleOrder(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(100+rand.Intn(300)) * time.Millisecond
	time.Sleep(delay)

	amount := rand.Float64() * 5000
	ordersTotal.Inc()
	orderAmount.Observe(amount)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"order_id":"%d","amount":%.2f}`, rand.Intn(100000), amount)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", instrumentHandler("/health", handleHealth))
	mux.HandleFunc("/api/hello", instrumentHandler("/api/hello", handleHello))
	mux.HandleFunc("/api/slow", instrumentHandler("/api/slow", handleSlow))
	mux.HandleFunc("/api/error", instrumentHandler("/api/error", handleError))
	mux.HandleFunc("/api/order", instrumentHandler("/api/order", handleOrder))
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	log.Printf("Server starting on %s", addr)
	log.Printf("Metrics available at %s/metrics", addr)
	log.Printf("Endpoints: /health, /api/hello, /api/slow, /api/error, /api/order")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
