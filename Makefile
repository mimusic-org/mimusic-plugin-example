# 颜色输出
BLUE=\033[0;34m
GREEN=\033[0;32m
NC=\033[0m # No Color


# 默认变量
PLUGIN_NAME ?= example
VERSION ?= 1.0.0

.PHONY: help
help: ## 显示帮助信息
	@echo "$(BLUE)MiMusic 插件示例构建工具$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[0;32m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: build
build: ## 编译插件为 WASM 格式
	@echo "$(BLUE)正在构建 ${PLUGIN_NAME}.wasm...$(NC)"
	@rm -f ${PLUGIN_NAME}.wasm
	GOOS=wasip1 GOARCH=wasm go build -o ${PLUGIN_NAME}.wasm -buildmode=c-shared
	@echo "$(GREEN)✓ 构建完成: ${PLUGIN_NAME}.wasm$(NC)"

.PHONY: info
info: ## 显示插件信息
	@echo "$(BLUE)插件名称: ${PLUGIN_NAME}$(NC)"
	@echo "$(BLUE)版本: ${VERSION}$(NC)"
	@echo "$(BLUE)目标架构: WASIP1/WASM$(NC)"

all: build info ## 完整构建流程
