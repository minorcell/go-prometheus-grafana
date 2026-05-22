# Prometheus 指标类型

## 四种基本类型

### 1. Counter（计数器）

只增不减的累积计数器。

```go
httpRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total number of HTTP requests",
    },
    []string{"method", "endpoint", "status"},
)
```

**使用场景**：请求总数、错误总数、处理的字节数

**常用 PromQL**：
```promql
# 每秒请求率
rate(http_requests_total[1m])

# 5分钟内的增量
increase(http_requests_total[5m])
```

### 2. Gauge（仪表盘）

可增可减的瞬时值。

```go
httpRequestsInFlight = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Name: "http_requests_in_flight",
        Help: "Number of HTTP requests currently being processed",
    },
)
```

**使用场景**：当前连接数、内存使用量、温度、队列长度

**常用 PromQL**：
```promql
# 直接查询当前值
http_requests_in_flight

# 最近5分钟的变化
delta(http_requests_in_flight[5m])
```

### 3. Histogram（直方图）

观测值分布在预定义桶中的计数。

```go
httpRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request duration in seconds",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "endpoint"},
)
```

**使用场景**：请求延迟、响应大小

**常用 PromQL**：
```promql
# P99 延迟
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 平均延迟
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

### 4. Summary（摘要）

类似 Histogram，但在客户端计算分位数。

```go
requestDuration = prometheus.NewSummaryVec(
    prometheus.SummaryOpts{
        Name:       "request_duration_seconds",
        Help:       "Request duration",
        Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
    },
    []string{"method"},
)
```

**使用场景**：当不需要聚合多个实例的分位数时

## Histogram vs Summary

| 特性 | Histogram | Summary |
|------|-----------|---------|
| 分位数计算 | 服务端 (PromQL) | 客户端 |
| 可聚合 | 是 | 否 |
| 精度 | 取决于桶配置 | 精确 |
| 性能开销 | 低 | 高 |
| 推荐 | ✅ 大多数场景 | 特殊场景 |

## 本项目中的指标

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `http_requests_total` | Counter | HTTP 请求总数 |
| `http_request_duration_seconds` | Histogram | 请求耗时分布 |
| `http_requests_in_flight` | Gauge | 当前处理中的请求数 |
| `business_orders_total` | Counter | 订单总数 |
| `business_order_amount` | Histogram | 订单金额分布 |
