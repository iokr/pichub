# syntax=docker/dockerfile:1

# ---------------------------- 前端构建 ----------------------------
FROM node:22-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY web/ ./
RUN npm run build


# ---------------------------- 后端构建 ----------------------------
FROM golang:1.24-alpine AS go-builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /src/web/dist ./internal/web/dist

ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -tags embed \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /out/pichub ./cmd/pichub


# ---------------------------- 运行镜像 ----------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl \
    && addgroup -S -g 10001 pichub \
    && adduser -S -u 10001 -G pichub -h /app pichub

WORKDIR /app

COPY --from=go-builder /out/pichub /usr/local/bin/pichub
COPY configs/config.yaml /app/config.example.yaml

RUN mkdir -p /app/data/uploads /app/data/thumbs /app/data/logs /app/data/cache \
    && chown -R pichub:pichub /app

ENV TZ=Asia/Shanghai

USER pichub

EXPOSE 8080

VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["pichub"]
CMD ["-c", "/app/config.yaml"]
