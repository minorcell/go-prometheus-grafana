# 第二章：Prometheus 指标类型

Prometheus 定义了四种核心指标类型。选择正确的类型是埋点的第一步 —— 类型选错了，后面的查询和可视化都会受限。

本章通过 **定义 → 代码示例 → PromQL 查询 → Grafana 面板** 的路径，逐一讲解每种类型。

---

## 2.1 Counter（计数器）

### 是什么？

Counter 是一个**只增不减**的累计值。它适合统计「发生了多少次」的场景。

```
# 典型特征：重置前始终上涨
http_requests_total  10 → 11 → 12 → 13 ...
```

### 什么时候用？

| 场景 | Counter 名称示例 |
|------|-----------------|
| HTTP 请求总数 | `http_requests_total` |
| 处理的总字节数 | `http_response_size_bytes_total` |
| 错误次数 | `errors_total` |
| 订单数量 | `business_orders_total` |
| 任务完成数 | `tasks_completed_total` |

> **经验法则**：如果一样东西可以「数数」，而且数只会越来越大，就用 Counter。

### 代码示例

```go
// 创建 Counter（使用 promauto 自动注册）
var httpRequestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "HTTP 请求总数",
    },
    []string{"method", "endpoint", "status"},
)

// 在业务代码中计数
httpRequestsTotal.WithLabelValues("GET", "/api/hello", "200").Inc()
```

**关键 API**：
- `.Inc()` —— 加 1
- `.Add(n)` —— 加 n
- `.WithLabelValues(...).Inc()` —— 为特定标签组合计数

### PromQL 查询

**直接查询 Counter 的值意义不大**（它只会越来越大），真正有用的是「速率」和「增量」：

```promql
# ❌ 错误用法：直接查询（值太大且单调增长，没有参考意义）
http_requests_total

# ✅ 正确用法 1：每秒请求速率（QPS）
rate(http_requests_total[1m])

# ✅ 正确用法 2：过去 5 分钟的请求增量
increase(http_requests_total[5m])

# ✅ 正确用法 3：按 endpoint 分组的每秒请求速率
sum(rate(http_requests_total[1m])) by (endpoint)

# ✅ 正确用法 4：某个接口的 5xx 错误数
increase(http_requests_total{endpoint="/api/error", status=~"5.."}[1m])
```

**为什么不用 Counter 原始值，要用 rate/increase？**

Counter 会一直增长，重启时归零。如果直接画 Counter 的原始值，你会看到一条永远上涨的线（中间可能在重启时断崖式下跌），没有分析价值。

`rate()` 和 `increase()` 帮你在时间窗口内计算「变化率」，这才是分析师关心的指标。

> **关键理解**：`rate()` 返回的是「每秒增长率」，`increase()` 返回的是「时间范围内的总增量」。`increase(foo[5m]) ≈ rate(foo[5m]) * 300`（5 分钟 = 300 秒）。

---

## 2.2 Gauge（仪表盘）

### 是什么？

Gauge 是一个**可增可减**的瞬时值。它代表「当前状态」而非「累计事件」。

```
# 典型特征：上升、下降、归零都有可能
in_flight_requests  3 → 5 → 2 → 0 → 4 ...
```

### 什么时候用？

| 场景 | Gauge 名称示例 |
|------|---------------|
| 当前并发请求数 | `http_requests_in_flight` |
| 内存使用量 | `go_memstats_alloc_bytes` |
| Goroutine 数量 | `go_goroutines` |
| 队列长度 | `task_queue_size` |
| CPU 温度 | `cpu_temperature_celsius` |

> **经验法则**：如果一样东西会「涨也会跌」，就用 Gauge。

### 代码示例

```go
// 创建 Gauge
var httpRequestsInFlight = promauto.NewGauge(
    prometheus.GaugeOpts{
        Name: "http_requests_in_flight",
        Help: "当前处理中的请求数",
    },
)

// 在请求处理前后操作 Gauge
func handler(w http.ResponseWriter, r *http.Request) {
    httpRequestsInFlight.Inc()   // 进入时 +1
    defer httpRequestsInFlight.Dec()  // 离开时 -1
    // ... 处理请求
}
```

**关键 API**：
- `.Inc()` —— 加 1
- `.Dec()` —— 减 1
- `.Add(n)` —— 加 n（也可以是负数）
- `.Set(n)` —— 设为指定值

### PromQL 查询

```promql
# ✅ 直接查询当前值
http_requests_in_flight

# ✅ 最近 5 分钟的变化量
delta(http_requests_in_flight[5m])

# ✅ 最近 5 分钟的平均值
avg_over_time(http_requests_in_flight[5m])

# ✅ Go 内存使用（字节）
go_memstats_alloc_bytes

# ✅ Goroutine 数量趋势
go_goroutines
```

### Counter vs Gauge：是否该混淆？

**永远不要**。两者的语义完全不同：

```go
// ❌ 错误：用 Counter 表示当前并发数（Counter 只增不减，不能反映真实变化）
requestsInFlight: 5 → Inc() 6 → Inc() 7 → 但实际只有 3 个请求在处理

// ✅ 正确：用 Gauge 表示当前并发数
requestsInFlight: 5 → Dec() 4 → Dec() 3  ← 精确反映真实状态
```

---

## 2.3 Histogram（直方图）

### 是什么？

Histogram 将观测值放入**预定义的桶（bucket）**中计数，同时记录**总和**和**总数**。它不存储每个样本的值，而是存储分布信息。

```
# 假设你测量请求耗时，桶定义为：0.1s, 0.5s, 1s, 5s

请求耗时 0.3s  → _bucket{le="0.1"}=0  (0.3 > 0.1，不入桶)
                  _bucket{le="0.5"}=1  (0.3 ≤ 0.5，入桶！)
                  _bucket{le="1"}=1
                  _bucket{le="5"}=1

请求耗时 2s    → _bucket{le="0.1"}=0
                  _bucket{le="0.5"}=1
                  _bucket{le="1"}=1
                  _bucket{le="5"}=2   (2 ≤ 5，入桶！)
```

每个 Histogram 实际上会创建 **3 条独立的时间序列**：

| 子序列 | 命名规则 | 含义 |
|--------|---------|------|
| `_bucket{le="..."}` | `foo_bucket{le="0.5"}` | 观测值 ≤ 阈值的累计计数 |
| `_sum` | `foo_sum` | 所有观测值的总和 |
| `_count` | `foo_count` | 观测次数 |

### 什么时候用？

| 场景 | Histogram 名称示例 |
|------|-------------------|
| 请求延迟 | `http_request_duration_seconds` |
| 响应大小 | `http_response_size_bytes` |
| 订单金额分布 | `business_order_amount` |
| 任务处理时间 | `task_processing_seconds` |

> **经验法则**：如果你想问「P99 是多少？」，用 Histogram。

### 代码示例

```go
// 创建 Histogram（使用默认桶）
var httpRequestDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "请求耗时分布",
        Buckets: prometheus.DefBuckets,  // .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
    },
    []string{"method", "endpoint"},
)

// 创建 Histogram（使用自定义桶，适合业务场景）
var orderAmount = promauto.NewHistogram(
    prometheus.HistogramOpts{
        Name:    "business_order_amount",
        Help:    "订单金额分布",
        Buckets: []float64{10, 50, 100, 500, 1000, 5000},
    },
)

// 记录观测值
httpRequestDuration.WithLabelValues("GET", "/api/hello").Observe(0.23) // 230ms
orderAmount.Observe(1250.0) // 1250 元
```

**桶的选择至关重要**：

```go
// ❌ 桶选太粗：P99 精度差
Buckets: []float64{0.1, 1, 10}

// ❌ 桶选太细：产生太多时间序列，占用内存
Buckets: []float64{0.001, 0.002, 0.003, ...} // 100 个桶

// ✅ 合理选择：覆盖常见范围，上密下疏
Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
```

### PromQL 查询

```promql
# ✅ P50（中位数）
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# ✅ P90
histogram_quantile(0.90, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# ✅ P99
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# ✅ 按接口分组的 P99
histogram_quantile(0.99,
  sum(rate(http_request_duration_seconds_bucket[5m])) by (le, endpoint))

# ✅ 平均延迟（不使用 histogram_quantile）
rate(http_request_duration_seconds_sum[5m])
  /
rate(http_request_duration_seconds_count[5m])

# ✅ Apdex 分数（用户满意度指标）
(sum(rate(http_request_duration_seconds_bucket{le="0.3"}[5m])) +
 sum(rate(http_request_duration_seconds_bucket{le="1.2"}[5m])) / 2)
 /
 sum(rate(http_request_duration_seconds_count[5m]))
```

**为什么用 `rate()` 包一层，再传给 `histogram_quantile()`？**

Histogram 的 bucket 是 Counter（累计计数，只增不减）。如果不先计算 rate，你得到的是「从程序启动到现在的请求分布」—— 这会被历史数据稀释，看不到最新的延迟变化。

`rate() + histogram_quantile()` 的组合确保你看到的是「最近时间窗口内的分布」。

---

## 2.4 Summary（摘要）

### 是什么？

Summary 与 Histogram 类似，都用于记录分布。但 **Summary 在客户端计算分位数**，然后直接暴露分位值。

```go
// 服务端暴露的就是计算好的 P50/P90/P99
request_duration_seconds{quantile="0.5"} 0.23
request_duration_seconds{quantile="0.9"} 0.85
request_duration_seconds{quantile="0.99"} 1.42
request_duration_seconds_sum 140.5
request_duration_seconds_count 1000
```

### Histogram vs Summary：该用哪个？

| 维度 | Histogram ✅ 推荐 | Summary |
|------|-------------------|---------|
| 分位数计算位置 | 服务端（PromQL） | 客户端（应用代码） |
| 可跨服务聚合 | ✅ 可以对多个实例相加 | ❌ 分位数不能相加 |
| 查询灵活性 | ✅ 可以查询 P50/P90/P99/etc. | ❌ 只预计算了指定的分位值 |
| 性能开销 | 低（只需计数器操作） | 高（客户端需要维护滑动窗口） |
| 精度 | 取决于桶配置 | 精确（在窗口内） |

> **结论**：90% 的场景用 **Histogram**。Summary 仅适用于不需要跨实例聚合、且对精确分位数有硬性要求的特殊场景。

### 代码示例（仅供参考）

```go
var requestDuration = promauto.NewSummary(
    prometheus.SummaryOpts{
        Name:       "request_duration_seconds",
        Help:       "请求耗时摘要",
        Objectives: map[float64]float64{
            0.5: 0.05,   // P50 误差在 ±5% 内
            0.9: 0.01,   // P90 误差在 ±1% 内
            0.99: 0.001, // P99 误差在 ±0.1% 内
        },
    },
)
```

---

## 2.5 本项目指标一览

回到我们的 Go Server，看看每种指标类型的实际应用：

| 指标名 | 类型 | 标签 | 问答 |
|--------|------|------|------|
| `http_requests_total` | Counter | method, endpoint, status | 「过去一分钟 /api/hello 每秒处理多少请求？」 |
| `http_request_duration_seconds` | Histogram | method, endpoint | 「/api/slow 的 P99 延迟是多少？」 |
| `http_requests_in_flight` | Gauge | — | 「当前有多少请求正在处理中？」 |
| `business_orders_total` | Counter | — | 「今天创建了多少订单？」 |
| `business_order_amount` | Histogram | — | 「订单金额的中位数是多少？」 |

访问 `http://localhost:8080/metrics` 自己看一眼这些指标吧 —— 理解数据格式是后面的基础。

---

## 2.6 动手练习

启动项目后，在 Prometheus UI (http://localhost:9090) 中尝试以下查询：

1. **查看所有端点每秒的请求速率**
   ```
   sum(rate(http_requests_total[1m])) by (endpoint)
   ```

2. **查看所有 5xx 错误每秒的数量**
   ```
   sum(rate(http_requests_total{status=~"5.."}[1m]))
   ```

3. **查看 /api/error 接口的各类状态码分布**
   ```
   sum(rate(http_requests_total{endpoint="/api/error"}[1m])) by (status)
   ```

4. **查看全局 P99 延迟**
   ```
   histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
   ```

5. **查看当前并发处理中的请求数**
   ```
   http_requests_in_flight
   ```

每个查询都试试看，感受一下 Counter 和 Histogram 在实操中的区别。
