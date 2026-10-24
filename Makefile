# PicHub Makefile
# 常用目标：make build（含前端）/ make build-backend / make dev / make docker

APP        := pichub
BIN_DIR    := bin
MAIN       := ./cmd/pichub
WEB_DIR    := web
EMBED_DIR  := internal/web/dist
CONFIG     ?= config.yaml

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

GOFLAGS    := -trimpath
GOTAGS     := embed

.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: deps
deps: ## 安装前后端依赖
	go mod download
	cd $(WEB_DIR) && npm ci --no-audit --no-fund

.PHONY: web
web: ## 构建前端并同步到 embed 目录
	cd $(WEB_DIR) && npm run build
	@mkdir -p $(EMBED_DIR)
	cp -r $(WEB_DIR)/dist/. $(EMBED_DIR)/

.PHONY: build
build: web ## 构建单二进制（内嵌前端）
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -tags "$(GOTAGS)" -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) $(MAIN)

.PHONY: build-backend
build-backend: ## 仅构建后端（不内嵌前端）
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) $(MAIN)

.PHONY: linux
linux: web ## 交叉编译 Linux amd64 单二进制
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -tags "$(GOTAGS)" -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-linux-amd64 $(MAIN)

.PHONY: typecheck
typecheck: ## 前端 TypeScript 类型检查
	cd $(WEB_DIR) && npm run typecheck

.PHONY: test
test: ## 运行 Go 测试
	go test ./...

.PHONY: vet
vet: ## 运行 go vet
	go vet ./...

.PHONY: fmt
fmt: ## 格式化 Go 代码
	go fmt ./...

.PHONY: run
run: ## 运行后端（默认读取 config.yaml）
	go run $(MAIN) -c $(CONFIG)

.PHONY: dev-web
dev-web: ## 启动前端开发服务器
	cd $(WEB_DIR) && npm run dev

.PHONY: dev
dev: ## 启动后端调试模式（需先启动 mysql/redis）
	go run $(MAIN) -c $(CONFIG)

.PHONY: config
config: ## 生成配置文件（已存在则跳过）
	@test -f config.yaml || cp configs/config.example.yaml config.yaml
	@echo "config ready: config.yaml"

.PHONY: docker
docker: ## 构建 Docker 镜像
	docker build -t $(APP):$(VERSION) .

.PHONY: up
up: ## 启动 docker compose 栈
	docker compose up -d --build

.PHONY: down
down: ## 停止 docker compose 栈
	docker compose down

.PHONY: logs
logs: ## 查看 compose 日志
	docker compose logs -f pichub

.PHONY: clean
clean: ## 清理构建产物
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist
	find $(EMBED_DIR) -mindepth 1 ! -name '.gitkeep' -exec rm -rf {} +
