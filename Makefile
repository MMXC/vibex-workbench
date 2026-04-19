.PHONY: help generate lint-specs build dev test drift clean init validate

SPEC_DIR := $(shell pwd)/specs
FRONTEND_DIR := $(shell pwd)/frontend
GENERATORS_DIR := $(shell pwd)/generators
BACKEND_DIR := $(shell pwd)/backend
SPEC_ENGINE_DIR := $(shell pwd)/spec-engine

help:
	@echo "VibeX Workbench — Spec 工程自举工作台"
	@echo ""
	@echo "  make init          初始化项目（首次克隆后）"
	@echo "  make lint-specs    验证 spec YAML 语法和从属关系"
	@echo "  make validate      深度验证：从属链 + 语义 + 层级一致性"
	@echo "  make gen-graph     生成从属关系图（Mermaid）"
	@echo "  make generate      运行所有代码生成器"
	@echo "  make dev           启动前端开发服务器"
	@echo "  make test          运行前端测试"
	@echo "  make drift         检测 spec 与代码的漂移"
	@echo "  make build         构建生产版本"
	@echo "  make clean         清理生成文件"
	@echo ""
	@echo "  make self-generate 生成自身（自举验证）"
	@echo "  make lint-all      全量验证（lint + validate + drift）"

init:
	@echo "初始化 VibeX Workbench..."
	@cd $(FRONTEND_DIR) && npm install
	@echo "✅ 初始化完成"

# ── Spec 验证 ──────────────────────────────────────────────

lint-specs:
	@echo "🔍 验证 spec YAML 语法..."
	@python3 $(GENERATORS_DIR)/validate_specs.py $(SPEC_DIR)
	@echo "✅ 语法验证通过"

validate: lint-specs
	@echo "🔍 从属链 + 层级一致性深度验证..."
	@python3 $(SPEC_ENGINE_DIR)/validate_chain.py $(SPEC_DIR)
	@echo "✅ 深度验证通过"

gen-graph:
	@echo "📊 生成从属关系图..."
	@python3 $(SPEC_ENGINE_DIR)/mermaid_gen.py $(SPEC_DIR) 		--output $(FRONTEND_DIR)/src/lib/generated/dependency-graph.mermaid
	@echo "✅ 从属图生成完成"

# ── 代码生成 ──────────────────────────────────────────────

generate: lint-specs
	@echo "⚙️  生成前端代码..."
	@python3 $(GENERATORS_DIR)/gen.py $(SPEC_DIR) $(FRONTEND_DIR)
	@echo "✅ 代码生成完成"
	@echo "  运行: make dev"

self-generate: validate generate
	@echo "🔄 自举生成完成"
	@echo "  已用 vibex-workbench 的 specs 生成自身代码"

# ── 开发 / 构建 ──────────────────────────────────────────────

build: generate
	@echo "🏗️  构建生产版本..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "✅ 构建完成"

dev: generate
	@echo "🚀 启动开发服务器..."
	@cd $(FRONTEND_DIR) && npm run dev

test:
	@echo "🧪 运行测试..."
	@cd $(FRONTEND_DIR) && npm test

drift:
	@echo "🔎 检测 Schema Drift..."
	@python3 $(GENERATORS_DIR)/drift_check.py $(SPEC_DIR) $(FRONTEND_DIR)

lint-all: validate drift
	@echo "✅ 全量验证完成"

clean:
	@echo "🧹 清理生成文件..."
	@rm -rf $(FRONTEND_DIR)/src/generated
	@rm -rf $(FRONTEND_DIR)/src/lib/generated
	@echo "✅ 清理完成"
