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
FROM scratch

# 从builder阶段复制必要的证书和时区文件（解决HTTPS和时区问题）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai

# 设置时区（可选，解决日志时区问题）
ENV TZ=Asia/Shanghai
ENV SERVER_EXTERNAL_URL="" \
    SERVER_INTERNAL_URL=http://breez-lnurl-service:4001 \
    DATABASE_URL=""


# 创建非root用户（k3s最佳实践，降低权限风险）
USER 1000:1000

# 创建工作目录
WORKDIR /app

# 从构建阶段复制编译好的可执行文件到运行阶段
COPY --from=builder /app/app ./

# 暴露端口（如果你的Go项目是Web服务，比如3001端口，需对应修改）
EXPOSE 4001

# 容器启动时执行的命令（运行编译后的程序）
CMD ["./app"]