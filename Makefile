.PHONY: build run clean test fmt vet lint deps help

# 应用名称
APP_NAME=keepa
MML_CLIENT_NAME=mml-client

# 构建目录
BUILD_DIR=bin

# Go 参数
GO=go
GOFMT=gofmt
GOLINT=golint

help: ## 显示帮助信息
	@echo "可用的命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## 下载依赖
	$(GO) mod download
	$(GO) mod tidy

fmt: ## 格式化代码
	$(GOFMT) -s -w .

vet: ## 运行 go vet
	$(GO) vet ./...

test: ## 运行测试
	$(GO) test -v ./...

build: ## 构建应用
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/keepa

build-client: ## 构建 MML Client
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(MML_CLIENT_NAME) ./cmd/mml-client

build-all: build build-client ## 构建所有应用（包括 MML Client）

run: build ## 构建并运行应用
	./$(BUILD_DIR)/$(APP_NAME)

clean: ## 清理构建文件
	rm -rf $(BUILD_DIR)
	rm -rf logs

install: build ## 安装到系统
	cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/

