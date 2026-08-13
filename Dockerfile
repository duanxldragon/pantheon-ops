# 多阶段构建 Dockerfile
# 用于构建 Pantheon Base 生产镜像

# 阶段 1: 构建前端
FROM node:24-alpine AS frontend-builder

WORKDIR /app/frontend

# 复制前端依赖文件
COPY frontend/package*.json ./

# 安装构建依赖
RUN npm ci --ignore-scripts

# 复制前端源码
COPY frontend/ ./

# 应用 React 19 兼容补丁并构建前端
RUN npm run patch:arco-react19 && npm run build

# 阶段 2: 构建后端
FROM golang:1.26.5-alpine AS backend-builder

ARG PANTHEON_VERSION=dev
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache ca-certificates git tzdata

# 复制 Go 模块文件
COPY backend/go.mod backend/go.sum ./

# 下载依赖
RUN go mod download

# 复制后端源码
COPY backend/ ./

# 构建后端（静态链接并注入发布版本）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -X pantheon-base/pkg/version.Version=${PANTHEON_VERSION}" \
    -o /app/server \
    ./cmd/server

# 阶段 3: 最终镜像
FROM alpine:3.19

# 安装运行时依赖
RUN apk --no-cache add ca-certificates curl tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 pantheon && \
    adduser -D -u 1000 -G pantheon pantheon && \
    mkdir -p /app/dist /app/uploads /var/log/pantheon && \
    chown -R pantheon:pantheon /app /var/log/pantheon

WORKDIR /app

# 从构建阶段复制文件
COPY --from=backend-builder /app/server /app/server
COPY --from=frontend-builder /app/frontend/dist /app/dist

# 切换到非 root 用户
USER pantheon

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# 启动服务
CMD ["/app/server"]
