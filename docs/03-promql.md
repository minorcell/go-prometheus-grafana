# 第三章：PromQL 实战手册

PromQL 是 Prometheus 的查询语言，也是 Grafana 面板背后的「发动机」。**会写 PromQL，你就能回答关于系统的任何问题。**

本章从基础到高级，循序渐进地讲解 PromQL。

---

## 3.1 先理解：时间序列

在学语法之前，先理解 Prometheus 存储数据的模型：

```
时间序列 = 指标名 + 标签集合 + 时间戳 + 值

例如：
http_requests_total{method="GET", endpoint="/health", status="200"}
  @1716379200  42
  @1716379205  42
  @1716379210  43
  @1716379215  43
  ...
```

同一个指标名 + 不同标签 = 不同的时间序列。每个时间序列是一条随时间变化的数据线。

---

## 3.2 即时查询 vs 范围查询

在 Prometheus UI 中执行查询时有**两种模式**：

| 模式 | 返回什么 | 用途 |
|------|---------|------|
| **Instant Query** (Table) | 最近一个抓取周期的值 | 查看当前状态（当前内存、当前并发数） |
| **Range Query** (Graph) | 一段时间内所有抓取点的值 | 查看趋势变化（折线图、时域分析） |

```promql
# 即时查询：返回一个当前值
go_goroutines
# → go_goroutines{} 12

# 范围查询（在 Graph 模式下 [5m] 自动生效）：返回一系列时间点
go_goroutines
# → [{time: 14:00, value: 10}, {time: 14:05, value: 12}, ...]
```

---

## 3.3 选择器（Selector）

### 精确匹配

```promql
# 选择指定标签值的时间序列
http_requests_total{method="GET"}

# 多个标签条件（AND 逻辑）
http_requests_total{method="GET", endpoint="/health"}
```

### 正则匹配

```promql
# =~ 正则匹配
http_requests_total{endpoint=~"/api/.*"}       # 匹配所有 /api/ 开头的接口
http_requests_total{status=~"5.."}              # 匹配 500-599
http_requests_total{method=~"GET|POST"}         # 匹配 GET 或 POST

# !~ 反向正则匹配
http_requests_total{endpoint!~"/health|/metrics"} # 排除 /health 和 /metrics
```

### 不等匹配

```promql
# != 不等于
http_requests_total{status!="200"}

# !~ 不匹配正则
http_requests_total{endpoint!=".*internal.*"}
```

---

## 3.4 核心函数

PromQL 的内置函数是它的灵魂。我们按使用频率从高到低讲解。

### rate() —— 计算每秒增长率

**最重要的函数，没有之一。** 90% 的 Counter 查询都应该包一层 rate()。

```promql
# 语法：rate(指标[时间窗口])
# 返回：每秒的增长速率（float64）

# 过去 1 分钟内的每秒请求速率
rate(http_requests_total[1m])

# 过去 5 分钟内的每秒错误速率
rate(http_requests_total{status=~"5.."}[5m])

# 按接口分组的 QPS
sum(rate(http_requests_total[1m])) by (endpoint)
```

**时间窗口选择指南**：

| 窗口 | 适用场景 |
|------|---------|
| `[1m]` | 实时监控、快速检测突变（本项目默认窗口） |
| `[5m]` | 平衡灵敏度和稳定性（生产环境常用） |
| `[15m]` | 趋势分析、消除短期抖动 |

> **关键理解**：时间窗口越长，曲线越平滑；窗口越短，对突变的响应越快。没有「最好的窗口」，只有「适合当前场景的窗口」。

`rate()` 的底层原理：

```
rate(metric[1m]) = (最后一个值 - 第一个值) / 时间跨度

如果指标在 1 分钟内从 100 涨到 160：
rate() = (160 - 100) / 60s = 1.0 req/s
```

**rate() 会自动处理 Counter 重置**（进程重启时 Counter 归零），不会因为重置产生错误的数据尖刺。

### increase() —— 计算时间窗口内的总增量

```promql
# 语法：increase(指标[时间窗口])
# 返回：窗口内增加的总量

# 过去 1 小时的总请求数
increase(http_requests_total[1h])

# 过去 5 分钟的订单增量
increase(business_orders_total[5m])

# 过去 1 分钟的 5xx 错误总数
increase(http_requests_total{status=~"5.."}[1m])
```

**rate() vs increase() 的区别只有一个：**

```promql
rate(http_requests_total[5m])          # 单位：req/second
increase(http_requests_total[5m])      # 单位：req（绝对值）

# 两者关系：
increase(metric[5m]) ≈ rate(metric[5m]) * 300  # 300 = 5m = 300s
```

用哪个取决于你想要什么：
- 「接口的 QPS 是多少？」→ `rate()`
- 「过去一小时发了多少请求？」→ `increase()`

### histogram_quantile() —— 从直方图计算分位数

```promql
# 语法：histogram_quantile(分位数, 对 bucket 的函数)

# P99 延迟
histogram_quantile(0.99,
  sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 按接口分组的 P90
histogram_quantile(0.90,
  sum(rate(http_request_duration_seconds_bucket[5m])) by (le, endpoint))

# P50（中位数）
histogram_quantile(0.50,
  sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
```

**注意 `by (le)` 是必须的** —— `le` 标签标记了桶的边界，`histogram_quantile()` 需要靠它识别各桶之间的分布。如果忘记 `by (le)`，你会得到不可预测的结果。

### sum / avg / max / min —— 聚合操作符

```promql
# sum：所有维度相加
sum(http_requests_total)
# → 全局总请求数

# sum + by：按维度分组求和
sum(http_requests_total) by (endpoint)
# → 每个接口的请求总数

# sum + without：去掉某些维度后求和
sum(http_requests_total) without (method, status)
# → 保留 endpoint 维度（去掉 method 和 status）

# avg：平均值
avg(go_goroutines{})

# max / min：最大值 / 最小值
max(http_requests_in_flight)
min(go_memstats_alloc_bytes)
```

### irate() —— 瞬时增长率

```promql
# irate 只看最近两个样本，对突变更敏感
irate(http_requests_total[1m])

# rate 看整个窗口的平均，更平滑
rate(http_requests_total[1m])
```

| 函数 | 敏感度 | 平滑度 | 场景 |
|------|--------|--------|------|
| `rate()` | 中 | 高 | 常规监控（推荐） |
| `irate()` | 高 | 低 | 快速探测突发流量 |

---

## 3.5 组合运用：回答真实问题

让我们用 PromQL 回答几个运维中常见的问题：

### Q1：「哪个接口的错误率最高？」

```promql
# Step 1：计算每个接口的 5xx 速率
sum(rate(http_requests_total{status=~"5.."}[5m])) by (endpoint)
```

在 Prometheus Graph 中观察 —— 曲线最高的那个就是错误率最高的接口。

### Q2：「/api/error 的错误率占总请求的百分比？」

```promql
# 错误率百分比
sum(rate(http_requests_total{endpoint="/api/error", status=~"5.."}[5m]))
  /
sum(rate(http_requests_total{endpoint="/api/error"}[5m]))
  * 100
```

### Q3：「平均请求延迟是多少？」

```promql
# 平均延迟 = 总耗时 / 总次数
rate(http_request_duration_seconds_sum[5m])
  /
rate(http_request_duration_seconds_count[5m])
```

### Q4：「哪个接口最慢？（按 P99）」

```promql
histogram_quantile(0.99,
  sum(rate(http_request_duration_seconds_bucket[5m])) by (le, endpoint))
```

### Q5：「5xx 错误正在加速还是减速？」

```promql
# 在 Graph 中查看 5xx 的 rate 曲线
# 曲线向上 → 加速（恶化）
# 曲线向下 → 减速（恢复）
sum(rate(http_requests_total{status=~"5.."}[5m]))
```

### Q6：「过去 6 小时处理了多少订单？」

```promql
increase(business_orders_total[6h])
```

---

## 3.6 实战练习：在 Prometheus UI 中动手

打开 http://localhost:9090 ，进入 **Graph** 标签页，依次执行以下查询：

### 练习一：基础选择器

```promql
# 1. 查看所有 http_requests_total 指标
http_requests_total

# 2. 只看 GET 请求
http_requests_total{method="GET"}

# 3. 只看 5xx 错误
http_requests_total{status=~"5.."}

# 4. 排除 /health 端点
http_requests_total{endpoint!="/health"}
```

### 练习二：rate 和 increase

```promql
# 5. 全局 QPS
sum(rate(http_requests_total[1m]))

# 6. 每个接口的 QPS（在 Graph 中区分颜色）
sum(rate(http_requests_total[1m])) by (endpoint)

# 7. 过去 1 分钟的请求增量
sum(increase(http_requests_total[1m]))
```

### 练习三：Histogram 分位数

```promql
# 8. P99 延迟
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 9. 按接口的 P99（比较不同接口的延迟差异）
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, endpoint))
```

### 练习四：实时指标

```promql
# 10. 当前并发请求数
http_requests_in_flight

# 11. Goroutine 数量
go_goroutines

# 12. 内存分配字节
go_memstats_alloc_bytes
```

> **建议**：每个查询都切换到 **Graph** 视图看看折线图，再切回 **Table** 看看即时值。理解两种模式的区别是掌握 PromQL 的重要一步。
