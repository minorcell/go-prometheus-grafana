# 核心概念

## 整体架构

```
┌─────────┐       ┌────────────┐       ┌─────────┐
│  Client │──────▶│  Go Server │◀──────│Prometheus│
│  (Load) │       │  :8080     │ scrape│  :9090   │
└─────────┘       └────────────┘       └────┬────┘
                                             │ query
                                        ┌────▼────┐
                                        │ Grafana │
                                        │  :3000  │
                                        └─────────┘
```

## Prometheus 是什么？

Prometheus 是一个开源的系统监控和告警工具包，主要特点：

1. **拉取模型 (Pull-based)**：Prometheus 主动从目标服务拉取指标，而不是服务推送指标
2. **时序数据库**：内置高效的时序数据存储
3. **PromQL**：强大的查询语言
4. **服务发现**：支持多种服务发现机制
5. **告警**：内置 Alertmanager 支持

## Grafana 是什么？

Grafana 是一个开源的可视化平台：

1. **多数据源**：支持 Prometheus、InfluxDB、Elasticsearch 等
2. **丰富的面板**：时序图、柱状图、仪表盘、表格等
3. **告警**：可配置告警规则
4. **模板变量**：动态过滤和切换
5. **分享**：Dashboard 可导出为 JSON

## 数据流

1. Go Server 在 `/metrics` 端点暴露指标
2. Prometheus 每 5 秒抓取一次 `/metrics`
3. 数据存储在 Prometheus 时序数据库中
4. Grafana 通过 PromQL 查询 Prometheus 获取数据
5. Grafana 将数据可视化为图表
