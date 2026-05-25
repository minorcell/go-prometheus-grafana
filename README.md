# Prometheus + Grafana 学习项目

一个完整的可观测性学习环境：Go HTTP Server → Prometheus → Grafana。

> 交互教学：[可观测性实战：Prometheus + Grafana 全栈监控](https://mcell.top/tutorials/observability-prometheus-grafana)

## 架构

```
┌──────────┐         ┌────────────┐  scrape   ┌────────────┐  query   ┌─────────┐
│  Client  │────────▶│ Go Server  │◀──────────│ Prometheus │◀──────── │ Grafana │
│ (压测)    │  HTTP   │   :8080    │  /metrics │   :9090    │          │  :3000  │
└──────────┘         └────────────┘           └────────────┘          └─────────┘
```

## 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.21+（仅本地开发需要）

### 一键启动

```bash
docker compose up --build
```

### 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| Go Server | http://localhost:8080 | 业务服务 |
| Prometheus | http://localhost:9090 | 指标查询 |
| Grafana | http://localhost:3000 | 可视化面板 (admin/admin) |

### 生成流量

```bash
# 使用 client 生成负载
cd client && go run main.go -url http://localhost:8080 -concurrency 10

# 或用 curl 手动测试
curl http://localhost:8080/api/hello
curl http://localhost:8080/api/slow
curl http://localhost:8080/api/error
curl http://localhost:8080/api/order
curl http://localhost:8080/metrics
```

## 项目结构

```
.
├── server/              # Go HTTP 服务（带 Prometheus 埋点）
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── client/              # 压测客户端
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── prometheus/          # Prometheus 配置
│   └── prometheus.yml
├── grafana/             # Grafana 配置
│   ├── provisioning/   # 自动化配置（数据源 + Dashboard）
│   └── dashboards/     # Dashboard JSON
├── docs/               # 学习文档
│   ├── 01-concepts.md      # 核心概念
│   ├── 02-metrics-types.md # 指标类型详解
│   ├── 03-promql.md        # PromQL 查询语法
│   └── 04-grafana-guide.md # Grafana 使用指南
├── docker-compose.yml
└── README.md
```

## 暴露的指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `http_requests_total` | Counter | 请求总数（按 method/endpoint/status 分组） |
| `http_request_duration_seconds` | Histogram | 请求延迟分布 |
| `http_requests_in_flight` | Gauge | 当前处理中的请求 |
| `business_orders_total` | Counter | 业务订单数 |
| `business_order_amount` | Histogram | 订单金额分布 |

## 学习路径

1. **启动环境** → `docker compose up --build`
2. **查看原始指标** → 访问 http://localhost:8080/metrics
3. **Prometheus 查询** → 在 http://localhost:9090 练习 PromQL
4. **Grafana 面板** → 在 http://localhost:3000 查看预置 Dashboard
5. **阅读文档** → `docs/` 目录下的学习资料
6. **动手实验** → 修改 server 代码添加新指标，创建新的 Dashboard

## 停止服务

```bash
docker compose down

# 清除数据卷
docker compose down -v
```
