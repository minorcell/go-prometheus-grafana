# 第四章：Grafana 使用指南

Grafana 将 PromQL 查询结果转化为人类可以快速理解的图表。本章从 Dashboard 的整体结构讲起，逐步带你把「数据」变成「故事」。

---

## 4.1 登录与初始界面

启动 Docker Compose 后，打开浏览器访问：

```
http://localhost:3000
```

| 字段 | 值 | 说明 |
|------|-----|------|
| 用户名 | `admin` | 默认管理员账号 |
| 密码 | `admin` | 学习环境用默认值即可 |
| 首次登录 | 可跳过密码修改 | 点击「Skip」直接进入 |

进入后你会看到左侧有一个 **Go Server Metrics** Dashboard —— 这是我们预置的监控面板，点击进入。

---

## 4.2 预置 Dashboard 详解

打开 Dashboard 后，你会看到多个 Panel（面板）。每个 Panel 在告诉你一个故事：

```
┌──────────────────────────────┬──────────────────────────────┐
│  Request Rate (QPS)         │  Error Rate                  │
│  "各接口的流量有多大？"       │  "错误从哪里来？"              │
│  sum(rate(...)) by endpoint │  sum(rate(5xx/4xx)) by endpoint│
├──────────────────────────────┼──────────────────────────────┤
│  Request Duration (P50/90/99)│  In-Flight Requests          │
│  "用户的体验有多快？"         │  "系统正在处理多少个请求？"     │
│  histogram_quantile(...)    │  http_requests_in_flight       │
├──────────────┬───────────────┼──────────────┬───────────────┤
│ Orders Total │ Order Rate   │ Go Routines │ Memory         │
│ "订单累计数" │ "订单创建速率"│ "协程数量"   │ "内存使用"      │
└──────────────┴───────────────┴──────────────┴───────────────┘
```

点击任何 Panel 的标题 → **Edit**，你就能看到背后的 PromQL 查询和可视化配置。

---

## 4.3 创建你的第一个自定义 Panel

让我们从零开始创建一个「订单金额中位数」面板，并随着练习逐步优化。

### Step 1：添加新 Panel

1. 点击 Dashboard 右上角 **Add** → **Visualization**
2. 这会打开一个新的 Panel 编辑页面

### Step 2：编写 PromQL 查询

在底部的 Query 编辑器中输入：

```promql
histogram_quantile(0.50, sum(rate(business_order_amount_bucket[5m])) by (le))
```

**解读**：这条查询的意思是从 `business_order_amount` 直方图中计算「过去 5 分钟订单金额的中位数（P50）」。

### Step 3：选择可视化类型

Panel 右上角的下拉框选择可视化类型：

| 查询想回答的问题 | 推荐 Panel 类型 |
|-----------------|----------------|
| 「它在怎么变化？」 | **Time series**（折线图） |
| 「现在是多少？」 | **Stat**（大数字） |
| 「分布是怎样的？」 | **Bar chart** / **Histogram** |
| 「数值到极限了吗？」 | **Gauge**（仪表盘） |

对于「订单金额中位数」，选 **Time series**。

### Step 4：配置显示格式

右侧 **Panel options** 中：

| 设置 | 值 |
|------|-----|
| Title | `订单金额中位数（P50）` |
| Description | `过去 5 分钟订单金额的中位数，反映了客单价的核心趋势` |

右侧 **Standard options** 中：

| 设置 | 值 |
|------|-----|
| Unit | `Currency` → `Dollars ($)` |

### Step 5：保存

点击右上角 **Save**，输入 Panel 标题，然后 **Apply**。

---

## 4.4 面板类型选择指南

### Time Series（时序图）—— 最常用的面板

**适用场景**：展示指标随时间的变化趋势。

```promql
# 典型用法：各接口的 QPS 趋势
sum(rate(http_requests_total[1m])) by (endpoint)
```

**配置技巧**：
- **Legend** 设置为 `{{endpoint}}`，图例显示接口名
- **Gradient mode** 设为 `Opacity`，让线条更美观
- **Stack series** 切换堆叠/独立显示

### Stat（大数字）—— 一眼看到关键数值

**适用场景**：展示当前值的单数字面板。

```promql
# 典型用法：当前正在处理的请求数
http_requests_in_flight
```

**配置技巧**：
- **Value options** → Show 选择 `Calculate`，计算模式选 `Last`
- **Thresholds** 添加阈值颜色，比如 > 50 变红
- **Graph mode** 设为 `Area`，在数字右侧显示迷你趋势图

### Gauge（仪表盘）—— 展示利用率

**适用场景**：展示百分比类的容量指标。

```promql
# 假设你要监控队列满度，需要自己计算百分比
http_requests_in_flight / 100 * 100
```

### Bar Chart（柱状图）—— 对比不同维度

**适用场景**：横向比较不同服务/接口/错误码的数值。

```promql
# 典型用法：各接口的 5xx 错误数对比
sum(increase(http_requests_total{status=~"5.."}[1h])) by (endpoint)
```

### Table（表格）—— 展示明细数据

**适用场景**：需要同时看到精确数字和标签维度。

```promql
# 典型用法：按状态码和接口查看请求量
sum(rate(http_requests_total[1m])) by (endpoint, status)
```

---

## 4.5 进阶：模板变量

模板变量让一个 Dashboard 可以动态切换监控目标，避免为每个环境/服务创建重复的面板。

### 创建变量

1. **Dashboard Settings** (齿轮图标) → **Variables** (左侧标签) → **New variable**
2. 配置：

| 设置项 | 值 | 说明 |
|--------|-----|------|
| Name | `endpoint` | 在查询中引用时的变量名 |
| Label | `接口` | 在 Dashboard 顶部显示的名称 |
| Type | `Query` | 从数据源动态获取选项 |
| Query | `label_values(http_requests_total, endpoint)` | 查询所有 endpoint 标签的值 |

3. 点击 **Update** → **Save dashboard**

### 在查询中使用变量

```promql
# 用 $endpoint 引用变量
sum(rate(http_requests_total{endpoint=~"$endpoint"}[1m]))
```

现在 Dashboard 顶部会出现一个下拉框，选择不同的接口时，所有引用 `$endpoint` 的 Panel 都会自动刷新。

### 常用变量查询

```promql
# 获取某个指标的所有标签值
label_values(http_requests_total, endpoint)
label_values(http_requests_total, method)

# 按条件过滤后的标签值
label_values(http_requests_total{status=~"5.."}, endpoint)

# 自定义选项列表
custom: all, /health, /api/hello, /api/slow
```

---

## 4.6 告警（Alerting）

Grafana 也支持在 Panel 上配置告警规则。

### 创建告警

1. 打开一个现有 Panel 的编辑模式（或新建一个「5xx 错误率」的 Time Series Panel）
2. 切换到 **Alert** 标签
3. 点击 **Create alert rule from this panel**

### 告警规则示例

| 配置项 | 值 | 说明 |
|--------|-----|------|
| Rule name | `高错误率告警` | |
| Query | `sum(rate(http_requests_total{status=~"5.."}[5m]))` | 5xx 每秒数量 |
| Reduce | `Function: Last` | 取最近一个数据点 |
| Threshold | `IS ABOVE 1` | 5xx 速率超过 1 req/s 时触发 |

这套规则的含义是：「如果过去 5 分钟的平均 5xx 速率超过 1 次/秒，就触发告警」。

### 告警状态

| 状态 | 含义 |
|------|------|
| **Normal** | 一切正常 |
| **Pending** | 条件满足但未达到持续时间阈值 |
| **Alerting** | 持续满足条件，正在告警中 |
| **No Data** | 查询无返回数据（目标可能已挂） |

> **提示**：Prometheus 自带的 Alertmanager 更强大（支持分组、静默、路由等），Grafana Alerting 适合快速实验和学习。

---

## 4.7 Dashboard 的导入与导出

### 导出 Dashboard JSON

1. Dashboard → **Share** (右上角分享图标) → **Export**
2. 点击 **Save to file**
3. JSON 文件可以提交到 Git 仓库（就像本项目的 `grafana/dashboards/go-server.json`）

### 导入 Dashboard

1. 点击左上角 **+** → **Import**
2. 上传 JSON 文件或粘贴内容
3. 选择数据源 → **Import**

这就是我们在 `docker-compose.yml` 中挂载 Dashboard JSON 的自动化版本 —— 启动时就自动导入了。

---

## 4.8 动手练习：重新创建整个 Dashboard

学习的最好方式是亲手做一遍。试试看：

### 练习目标

从空白 Dashboard 开始，重新创建以下 Panel：

1. **全局 QPS 时序图**（Time series）
   - PromQL：`sum(rate(http_requests_total[1m]))`

2. **各接口 QPS 对比图**（Time series，多条线）
   - PromQL：`sum(rate(http_requests_total[1m])) by (endpoint)`
   - 图例：`{{endpoint}}`

3. **5xx 错误数**（Stat 大数字 + 迷你趋势图）
   - PromQL：`sum(rate(http_requests_total{status=~"5.."}[5m]))`
   - 添加阈值：> 0.5 变黄，> 1 变红

4. **P50 / P90 / P99 延迟**（Time series，3 条线）
   - PromQL: `histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`
   - 同样的写法: 0.50 → 0.90 → 0.99
   - 图例分别设为 P50、P90、P99

5. **订单累计总量**（Stat）
   - PromQL：`business_orders_total`

6. **Goroutine 数量**（Time series）
   - PromQL：`go_goroutines`

7. **内存使用**（Time series，单位 Byte → 自动转换）
   - PromQL: `go_memstats_alloc_bytes`
   - Standard options → Unit → Data → bytes(IEC)

### 完成后

保存 Dashboard，然后打开我们预置的 `Go Server Metrics`，对比一下你的实现和模板的差异。这种「逆向练习」是学习 Grafana 配置最快的方式。

---

## 本章小结

- **Panel** = 查询 + 可视化配置 = Grafana 的基本单元
- **Dashboard** = Panel 的集合 = 一个完整的监控视角
- **模板变量** 让 Dashboard 可以动态切换维度
- PromQL 决定你能看到什么数据，Panel 配置决定数据以什么形式呈现
