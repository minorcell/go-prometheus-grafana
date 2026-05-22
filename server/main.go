// Package main 实现了一个带 Prometheus 指标暴露的 HTTP 服务器。
//
// 本服务演示了在 Go 应用中集成 Prometheus 监控的最佳实践：
//  1. 定义指标（Counter / Gauge / Histogram）
//  2. 在请求生命周期中采集指标
//  3. 通过 /metrics 端点暴露指标给 Prometheus 抓取
//
// 启动后访问 http://localhost:8080/metrics 可查看原始指标数据。
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

// ============================================================
// 指标定义
// 使用 promauto 包自动将指标注册到默认 Registry，
// 省去手动调用 prometheus.MustRegister() 的步骤。
// ============================================================

var (
	// httpRequestsTotal 是一个 CounterVec 类型的计数器。
	// Counter 只增不减，适合统计「请求总数」「错误总数」等累计量。
	//
	// CounterVec 带标签（label），可以按维度切分数据。
	// 例如按 method/endpoint/status 分组统计：
	//   http_requests_total{method="GET",endpoint="/api/hello",status="200"}
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP 请求总数，按 HTTP 方法、接口路径和响应状态码分组",
		},
		[]string{"method", "endpoint", "status"},
	)

	// httpRequestDuration 是一个 HistogramVec 类型的直方图。
	// Histogram 将观测值放入预定义的桶（bucket）中，可以在 PromQL 中
	// 通过 histogram_quantile() 函数计算 P50/P90/P99 等分位数。
	//
	// 使用 prometheus.DefBuckets（默认桶）：
	//   .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10（秒）
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP 请求耗时分布（秒），按 HTTP 方法和接口路径分组",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// httpRequestsInFlight 是一个 Gauge 类型的瞬时值。
	// Gauge 可增可减，适合统计「当前并发数」「内存使用量」等瞬时状态。
	//
	// 注意：这里没有使用 Vec 变体，因为它不需要按标签切分。
	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "当前正在处理中的 HTTP 请求数",
		},
	)

	// ordersTotal 演示了业务指标的用法。
	// 除了 HTTP 层面的技术指标，Prometheus 也可以统计业务数据。
	// 这里统计「创建订单的总数」，帮助了解业务吞吐量。
	ordersTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "business_orders_total",
			Help: "业务指标 —— 创建的订单总数",
		},
	)

	// orderAmount 演示了自定义桶的用法。
	// 自定义桶比默认桶更能反映业务场景的金额分布：
	//   小额(<10), 普通(10-50), 中等(50-100), 大额(100-500), ...
	orderAmount = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "business_order_amount",
			Help:    "业务指标 —— 订单金额分布",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000},
		},
	)
)

// ============================================================
// 中间件：自动采集 HTTP 指标
// ============================================================

// instrumentHandler 是一个高阶函数（返回 http.HandlerFunc 的中间件）。
// 它包裹原始 handler，在请求前后自动采集以下指标：
//   1. 请求开始：http_requests_in_flight +1
//   2. 请求结束：http_requests_in_flight -1
//   3. 记录耗时：http_request_duration_seconds 直方图
//   4. 记录计数：http_requests_total 计数器（带 method/endpoint/status 标签）
//
// 使用 statusRecorder 包装 ResponseWriter 来捕获真实的 HTTP 状态码，
// 因为 Go 标准库的 ResponseWriter 不直接暴露状态码。
func instrumentHandler(endpoint string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 请求进入时：并发数 +1
		httpRequestsInFlight.Inc()
		// 请求离开时（无论成功或 panic）：并发数 -1
		defer httpRequestsInFlight.Dec()

		// 记录开始时间，用于计算耗时
		start := time.Now()

		// 用 statusRecorder 包装原始的 ResponseWriter，
		// 以便在执行 handler 后获取真实返回的 HTTP 状态码
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		handler(rec, r)

		// 计算请求耗时（秒）
		duration := time.Since(start).Seconds()

		// 记录请求总数（带 method / endpoint / status 标签）
		httpRequestsTotal.WithLabelValues(
			r.Method, endpoint, fmt.Sprintf("%d", rec.statusCode),
		).Inc()

		// 将耗时观测值记录到直方图中
		httpRequestDuration.WithLabelValues(
			r.Method, endpoint,
		).Observe(duration)
	}
}

// statusRecorder 包装 http.ResponseWriter，拦截 WriteHeader 调用，
// 从而记录真实返回的 HTTP 状态码。
//
// Go 标准库的 http.ResponseWriter 接口在 WriteHeader 被调用前不透露状态码，
// 因此需要用这个包装器来捕获 status code 用于指标采集。
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 覆盖嵌入的 ResponseWriter.WriteHeader，
// 先记录状态码，再委托给原始 ResponseWriter。
func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// ============================================================
// 业务 Handler
// ============================================================

// handleHealth 健康检查接口，始终返回 200。
// 可用于 Kubernetes 的 liveness/readiness probe。
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// handleHello 普通接口，模拟随机 0-100ms 的处理延迟。
// 代表正常业务接口的典型行为。
func handleHello(w http.ResponseWriter, r *http.Request) {
	// 模拟 0-100ms 的随机业务处理延迟
	delay := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(delay)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"hello, world!"}`)
}

// handleSlow 慢接口，模拟 500-2500ms 的处理延迟。
// 用于观察慢请求对整体延迟分布的影响，以及如何通过 PromQL 定位问题。
func handleSlow(w http.ResponseWriter, r *http.Request) {
	// 模拟 500ms ~ 2.5s 的长耗时处理
	delay := time.Duration(500+rand.Intn(2000)) * time.Millisecond
	time.Sleep(delay)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"slow response"}`)
}

// handleError 模拟不稳定的接口，按概率返回不同状态码：
//   - 30% 概率返回 500 Internal Server Error
//   - 20% 概率返回 400 Bad Request
//   - 50% 概率返回 200 OK
//
// 这个接口用来观察错误率指标的波动和 Grafana 告警的触发效果。
func handleError(w http.ResponseWriter, r *http.Request) {
	// 短延迟模拟处理
	delay := time.Duration(rand.Intn(50)) * time.Millisecond
	time.Sleep(delay)

	// 30% 概率返回 500
	if rand.Float64() < 0.3 {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal server error"}`)
		return
	}

	// 20% 概率返回 400（在前一个 if 不命中的前提下）
	if rand.Float64() < 0.2 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"bad request"}`)
		return
	}

	// 其余 50% 正常返回
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message":"ok"}`)
}

// handleOrder 模拟创建订单的接口，演示业务指标的采集：
//   - 随机生成 0-5000 的订单金额
//   - 递增 ordersTotal 计数器
//   - 将金额记录到 orderAmount 直方图
//
// 这是业务指标埋点的典型模式：在业务逻辑的关键节点调用 .Inc() 或 .Observe()。
func handleOrder(w http.ResponseWriter, r *http.Request) {
	// 模拟 100-400ms 的业务处理延迟
	delay := time.Duration(100+rand.Intn(300)) * time.Millisecond
	time.Sleep(delay)

	// 随机生成订单金额（0 ~ 5000）
	amount := rand.Float64() * 5000

	// 业务指标埋点：订单计数 +1，金额入桶
	ordersTotal.Inc()
	orderAmount.Observe(amount)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"order_id":"%d","amount":%.2f}`, rand.Intn(100000), amount)
}

// ============================================================
// 主函数：启动服务器
// ============================================================

func main() {
	// 使用 Go 标准库的 ServeMux 作为路由
	mux := http.NewServeMux()

	// 注册业务路由（均通过 instrumentHandler 中间件包裹，自动采集指标）
	mux.HandleFunc("/health", instrumentHandler("/health", handleHealth))
	mux.HandleFunc("/api/hello", instrumentHandler("/api/hello", handleHello))
	mux.HandleFunc("/api/slow", instrumentHandler("/api/slow", handleSlow))
	mux.HandleFunc("/api/error", instrumentHandler("/api/error", handleError))
	mux.HandleFunc("/api/order", instrumentHandler("/api/order", handleOrder))

	// 注册 Prometheus 指标暴露端点
	// promhttp.Handler() 自动从默认 Registry 中读取所有已注册的指标，
	// 并以 Prometheus 文本格式输出
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	log.Printf("========================================")
	log.Printf("  Server starting on %s", addr)
	log.Printf("  Metrics:  http://localhost%s/metrics", addr)
	log.Printf("  Endpoints: /health /api/hello /api/slow /api/error /api/order")
	log.Printf("========================================")

	// http.ListenAndServe 会阻塞直到进程被终止
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
