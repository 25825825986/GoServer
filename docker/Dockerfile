# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译Linux可执行文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app ./cmd/main.go

# 运行阶段 - 使用更小的镜像
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从builder阶段复制编译好的二进制文件
COPY --from=builder /app/app .
COPY --from=builder /app/web ./web

# 暴露端口
EXPOSE 8080 8081

# 启动应用
CMD ["./app"]
