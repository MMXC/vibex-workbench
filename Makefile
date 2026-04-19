.PHONY: help generate lint-specs build dev test drift clean init

SPEC_DIR := $(shell pwd)/specs
FRONTEND_DIR := $(shell pwd)/frontend
GENERATORS_DIR := $(shell pwd)/generators
BACKEND_DIR := $(shell pwd)/backend

help:
	@echo "VibeX Workbench — Makefile"
	@echo ""
	@echo "  make init         初始化项目（首次克隆后）"
	@echo "  make lint-specs   验证 spec YAML 语法和从属关系"
	@echo "  make generate     运行所有生成器"
	@echo "  make dev          启动前端开发服务器"
	@echo "  make test         运行前端测试"
	@echo "  make drift        检测 spec 与代码的漂移"
	@echo "  make build        构建生产版本"
	@echo "  make clean        清理生成文件"

init:
	@echo "初始化 VibeX Workbench..."
	@cd $(FRONTEND_DIR) && npm install
	@echo "✅ 初始化完成"
	@echo ""
	@echo "下一步: make lint-specs"

lint-specs:
	@echo "验证 spec 从属关系..."
	@python3 $(GENERATORS_DIR)/validate_specs.py $(SPEC_DIR)
	@echo "✅ Spec 验证通过"

generate: lint-specs
	@echo "生成前端代码..."
	@python3 $(GENERATORS_DIR)/gen.py \
		$(SPEC_DIR) \
		$(FRONTEND_DIR)
	@echo "✅ 代码生成完成"
	@echo "  运行: make dev"

build: generate
	@echo "构建生产版本..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "✅ 构建完成"

dev: generate
	@echo "启动开发服务器..."
	@cd $(FRONTEND_DIR) && npm run dev

test:
	@echo "运行测试..."
	@cd $(FRONTEND_DIR) && npm test

drift:
	@echo "检测 Schema Drift..."
	@python3 $(GENERATORS_DIR)/drift_check.py $(SPEC_DIR) $(FRONTEND_DIR)

clean:
	@echo "清理生成文件..."
	@rm -rf $(FRONTEND_DIR)/src/generated
	@echo "✅ 清理完成"
