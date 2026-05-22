# ============================================================
# 多阶段构建（Multi-stage Build）
# ============================================================
# 阶段 1：编译阶段（builder）
# 使用完整的 Go 镜像编译二进制文件
# ============================================================
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 缓存层
# 如果 go.mod / go.sum 没变，Docker 会复用缓存的 go mod download 层
COPY go.mod go.sum ./
RUN go mod download

# 再复制源代码并编译
COPY . .

# CGO_ENABLED=0 编译为纯静态二进制，不依赖 libc
# 这样可以将最终镜像缩减到 ~10MB（仅 alpine + 静态二进制）
RUN CGO_ENABLED=0 go build -o server .

# ============================================================
# 阶段 2：运行阶段（runtime）
# 使用最小化的 alpine 镜像，只包含编译好的二进制
# ============================================================
FROM alpine:3.19

WORKDIR /app

# 从编译阶段复制二进制文件
COPY --from=builder /app/server .

# 声明容器监听端口（文档作用，实际端口映射由 docker-compose 控制）
EXPOSE 8080

# 容器启动命令
CMD ["./server"]
