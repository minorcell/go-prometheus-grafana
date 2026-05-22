// Package main 实现了一个压测客户端，用于向 server 持续发送 HTTP 请求。
//
// 本客户端的作用：
//   1. 持续生成流量，让 server 的指标「有数据可看」
//   2. 通过不同权重的端点模拟真实的流量分布
//   3. 让 Grafana 面板上的线条能持续波动
//
// 如果不运行客户端，server 的指标都是 0，Prometheus 和 Grafana 中就没有数据。
//
// 使用方式：
//   go run main.go -url http://localhost:8080 -concurrency 10 -interval 500ms
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ============================================================
// 命令行参数
// ============================================================

var (
	// baseURL 目标服务器的地址
	baseURL string
	// concurrency 并发的工作 goroutine 数量
	concurrency int
	// interval 每个 worker 两次请求之间的基础间隔
	interval time.Duration
)

func init() {
	flag.StringVar(&baseURL, "url", "http://localhost:8080", "目标服务器的 Base URL")
	flag.IntVar(&concurrency, "concurrency", 5, "并发 worker 数量（每个 worker 持续发请求）")
	flag.DurationVar(&interval, "interval", 500*time.Millisecond, "每个 worker 的请求基础间隔")
}

// ============================================================
// 端点配置：权重决定了不同接口的流量比例
// ============================================================

// endpoints 定义了可请求的服务器端点及其相对权重。
// 权重越大，该端点被选中的概率越高。
// 所有权重之和 = 1.0。
//
// 当前流量分布：
//   /api/hello  40% —— 模拟正常流量，0-100ms 延迟
//   /api/slow   10% —— 模拟慢请求，500-2500ms 延迟
//   /api/error  20% —— 模拟错误，30% 500 + 20% 400
//   /api/order  20% —— 模拟创建订单，带业务指标
//   /health     10% —— 健康检查
var endpoints = []struct {
	path   string
	weight float64
}{
	{"/api/hello", 0.4},
	{"/api/slow", 0.1},
	{"/api/error", 0.2},
	{"/api/order", 0.2},
	{"/health", 0.1},
}

// pickEndpoint 根据 weight 权重随机选择一个端点。
// 算法：生成 [0,1) 随机数，按权重累积直到命中。
//
// 例如 endpoints = [{A, 0.4}, {B, 0.3}, {C, 0.3}]
//   - 随机数 0.2 => A
//   - 随机数 0.5 => B
//   - 随机数 0.8 => C
func pickEndpoint() string {
	r := rand.Float64()
	cumulative := 0.0
	for _, ep := range endpoints {
		cumulative += ep.weight
		if r <= cumulative {
			return ep.path
		}
	}
	// 理论上不会走到这里（权重和 = 1.0），兜底返回第一个
	return endpoints[0].path
}

// ============================================================
// Worker：持续发送请求的 goroutine
// ============================================================

// worker 是一个持续运行的请求发生器。
// 它会：
//   1. 随机选择端点（按权重）
//   2. 发送 HTTP GET 请求
//   3. 记录响应状态和耗时
//   4. 等待间隔（带随机抖动）后继续
//
// 每个 worker 在独立的 goroutine 中运行，互不影响。
func worker(id int, wg *sync.WaitGroup) {
	// 确保在函数返回时通知 WaitGroup（实际上永不返回，这里是为了符合接口约定）
	defer wg.Done()

	// 创建一个 HTTP 客户端，设置 10 秒超时
	// 超时设置很重要：避免慢请求永久阻塞 worker
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		// 按权重随机选择端点
		endpoint := pickEndpoint()
		url := baseURL + endpoint

		// 记录请求开始时间
		start := time.Now()
		resp, err := client.Get(url)
		duration := time.Since(start)

		if err != nil {
			// 网络错误或超时 —— 记录错误日志
			log.Printf("[worker-%d] ERROR %s: %v", id, endpoint, err)
		} else {
			// 成功收到响应 —— 消费并丢弃 Body，防止连接泄漏
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			// 记录请求结果：端点、状态码、耗时
			log.Printf("[worker-%d] %s %d %s",
				id, endpoint, resp.StatusCode, duration.Round(time.Millisecond))
		}

		// 间隔等待：基础间隔的一半 + 随机抖动
		// 抖动（jitter）可以避免所有 worker 在同一时刻发请求，
		// 从而产生更自然的流量曲线（而非周期性的尖峰）
		jitter := time.Duration(rand.Int63n(int64(interval)))
		time.Sleep(interval/2 + jitter)
	}
}

// ============================================================
// 主函数
// ============================================================

func main() {
	// 解析命令行参数
	flag.Parse()

	// 打印启动信息
	fmt.Println("====================================")
	fmt.Println("  Load Generator (压测客户端)")
	fmt.Printf("  目标地址: %s\n", baseURL)
	fmt.Printf("  并发数:   %d\n", concurrency)
	fmt.Printf("  间隔:     %s\n", interval)
	fmt.Println("====================================")
	fmt.Println()

	// 创建 WaitGroup 管理所有 worker 的协程
	var wg sync.WaitGroup

	// 启动指定数量的 worker（每个在独立的 goroutine 中）
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go worker(i, &wg)
	}

	// wg.Wait() 会永远阻塞（worker 永不停止），
	// 按 Ctrl+C 可以终止整个进程
	wg.Wait()
}
