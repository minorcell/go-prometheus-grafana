# PromQL 常用查询

## 基础语法

### 选择器

```promql
# 精确匹配
http_requests_total{method="GET"}

# 正则匹配
http_requests_total{status=~"5.."}

# 反向匹配
http_requests_total{endpoint!="/health"}
```

### 时间范围

```promql
# 最近5分钟的样本
http_requests_total[5m]

# 5分钟前的瞬时值
http_requests_total offset 5m
```

## 常用函数

### rate() - 计算速率

```promql
# 每秒请求速率（基于1分钟窗口）
rate(http_requests_total[1m])

# 按 endpoint 分组的 QPS
sum(rate(http_requests_total[1m])) by (endpoint)
```

### increase() - 计算增量

```promql
# 过去1小时的请求增量
increase(http_requests_total[1h])
```

### histogram_quantile() - 分位数

```promql
# P50
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# P90
histogram_quantile(0.90, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# P99
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 按 endpoint 分组的 P99
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, endpoint))
```

### 聚合操作

```promql
# 总和
sum(rate(http_requests_total[1m]))

# 按标签分组
sum(rate(http_requests_total[1m])) by (endpoint)

# 平均值
avg(http_request_duration_seconds_sum / http_request_duration_seconds_count)

# 最大值
max(http_requests_in_flight)
```

## 实用查询示例

### 错误率

```promql
# 5xx 错误率占总请求的比例
sum(rate(http_requests_total{status=~"5.."}[5m])) 
/ 
sum(rate(http_requests_total[5m]))
```

### Apdex Score

```promql
# 假设 T=0.3s
(
  sum(rate(http_request_duration_seconds_bucket{le="0.3"}[5m]))
  +
  sum(rate(http_request_duration_seconds_bucket{le="1.2"}[5m]))
)
/ 2
/ sum(rate(http_request_duration_seconds_count[5m]))
```

### 平均请求延迟

```promql
rate(http_request_duration_seconds_sum[5m])
/
rate(http_request_duration_seconds_count[5m])
```

## 在 Prometheus UI 中练习

1. 打开 http://localhost:9090
2. 点击 "Graph" 标签
3. 输入上面的查询表达式
4. 点击 "Execute"
5. 切换 "Table" 和 "Graph" 查看不同展示方式
