# ============================================================
# 多阶段构建：编译 Go 二进制 → 最小化运行镜像
# ============================================================
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 依赖缓存层
COPY go.mod ./
RUN go mod download

# 编译
COPY . .
RUN CGO_ENABLED=0 go build -o client .

# ============================================================
# 运行阶段
# ============================================================
FROM alpine:3.19

WORKDIR /app

# 从编译阶段复制二进制
COPY --from=builder /app/client .

# 容器启动时直接连接 Docker Compose 网络中的 server 服务
# server 是 Docker Compose 中的服务名，Docker DNS 会自动解析
CMD ["./client", "-url", "http://server:8080", "-concurrency", "5", "-interval", "500ms"]
