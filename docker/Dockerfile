# MQTT 协议驱动镜像
#
# 构建并推送（把 IMAGE 换成你自己的仓库地址）：
#   docker buildx build --platform linux/amd64 \
#     -t IMAGE -f docker/Dockerfile . --push

FROM golang:1.24-alpine AS builder
WORKDIR /src

# 先拷依赖清单，利用层缓存
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/driver ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /etc/driver/res /mnt/logs

COPY --from=builder /out/driver /app/driver

# 设备接入端口（内嵌 MQTT broker）
EXPOSE 1883

ENTRYPOINT ["/app/driver"]