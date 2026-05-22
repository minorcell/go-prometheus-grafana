# Grafana 使用指南

## 登录

- 地址：http://localhost:3000
- 用户名：`admin`
- 密码：`admin`

## 预置 Dashboard

项目启动后会自动导入 "Go Server Metrics" Dashboard，包含：

1. **Request Rate (QPS)** - 各接口每秒请求数
2. **Error Rate** - 4xx/5xx 错误率
3. **Request Duration** - P50/P90/P99 延迟
4. **In-Flight Requests** - 当前并发请求数
5. **Orders Total** - 订单总数
6. **Order Rate** - 订单创建速率
7. **Go Runtime** - Goroutines 和内存使用

## 创建自定义 Panel

### 步骤

1. 点击 Dashboard 右上角 "Add" → "Visualization"
2. 选择数据源 "Prometheus"
3. 在 Query 编辑器中输入 PromQL
4. 配置面板样式
5. 保存

### 面板类型选择

| 场景 | 推荐面板 |
|------|----------|
| 趋势变化 | Time series |
| 当前值 | Stat / Gauge |
| 对比分布 | Bar chart |
| 热力图 | Heatmap |
| 日志 | Logs |
| 表格数据 | Table |

## 模板变量

可以在 Dashboard Settings → Variables 中添加变量：

```
# 查询所有 endpoint
label_values(http_requests_total, endpoint)

# 查询所有 method
label_values(http_requests_total, method)
```

然后在 Panel 查询中使用 `$endpoint` 引用变量。

## 告警规则

1. 进入 Panel 编辑模式
2. 切换到 "Alert" 标签
3. 配置告警条件，例如：
   - 当 error rate > 10% 持续 1 分钟
   - 当 P99 延迟 > 2s 持续 30 秒

## 导出与分享

- Dashboard → Share → Export：导出为 JSON
- Dashboard → Share → Snapshot：创建快照链接
