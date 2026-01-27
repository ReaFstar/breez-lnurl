FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod tidy

# 复制整个项目代码到容器内
COPY . .

# 编译Go项目（静态编译，避免容器运行时缺少依赖）
# CGO_ENABLED=0：关闭CGO，生成静态二进制文件
# GOOS=linux：指定编译为Linux系统可执行文件
# -o app：编译后的可执行文件名为app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./

# 阶段2：运行阶段（使用轻量的alpine镜像，减小镜像体积）
FROM alpine:3.18

# 安装必要的基础工具（可选，如需要日志、调试）
RUN apk --no-cache add ca-certificates tzdata

# 设置时区（可选，解决日志时区问题）
ENV TZ=Asia/Shanghai
ENV SERVER_EXTERNAL_URL="" \
    SERVER_INTERNAL_URL=http://172.22.16.8:4001 \
    DATABASE_URL=""

# 创建工作目录
WORKDIR /app

# 从构建阶段复制编译好的可执行文件到运行阶段
COPY --from=builder /app/app ./

RUN chmod +x /app/app

# 暴露端口（如果你的Go项目是Web服务，比如3001端口，需对应修改）
EXPOSE 4001

# 容器启动时执行的命令（运行编译后的程序）
CMD ["./app"]