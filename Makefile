.PHONY: help generate lint-specs build dev test drift clean init validate \
	skill-sync-status skill-sync-push skill-sync-pull

# Repo root from this Makefile path (avoids Git Bash pwd + Windows Python -> C:\c\... mangling)
ROOT := $(patsubst %/,%,$(subst \,/,$(dir $(lastword $(MAKEFILE_LIST)))))
SPEC_DIR := $(ROOT)/specs
FRONTEND_DIR := $(ROOT)/frontend
GENERATORS_DIR := $(ROOT)/generators
BACKEND_DIR := $(ROOT)/backend
SPEC_ENGINE_DIR := $(ROOT)/spec-engine
PYTHON ?= python3
# UTF-8 for Python on Windows GBK consoles; harmless on Linux/macOS
export PYTHONUTF8 := 1
export PYTHONIOENCODING := utf-8

help:
	@echo "VibeX Workbench -- spec tooling"
	@echo "  (spec change -> ship: SPEC-TO-SHIP-WORKFLOW.md)"
	@echo "  (governance vs landing gaps: SPEC-LANDING-GAPS.md)"
	@echo ""
	@echo "  make init          First-time setup (npm install in frontend)"
	@echo "  make lint-specs    Validate spec YAML syntax and parent refs"
	@echo "  make validate      Deep check: parent chain + level consistency"
	@echo "  make gen-graph     Emit dependency graph (Mermaid)"
	@echo "  make generate      Run all code generators"
	@echo "  make dev           Frontend dev server"
	@echo "  make test          Frontend tests"
	@echo "  make drift         Spec vs code drift check"
	@echo "  make build         Production build"
	@echo "  make clean         Remove generated artifacts"
	@echo ""
	@echo "  make self-generate Self-bootstrap (validate + generate + criteria check)"
	@echo "  make spec-criteria  Criteria check L1+L2 (file existence + imports)"
	@echo "  make spec-iterate  Full iteration: check + self-heal + agent gaps"
	@echo "  make lint-all      lint-specs + validate + drift"

init:
	@echo "[init] Setting up VibeX Workbench..."
	@cd $(FRONTEND_DIR) && npm install
	@echo "[init] Done."

# --- Spec validation ---
# Tier-1 gates for specs/: lint-specs (syntax + parent file refs) then validate_chain.
# Layer contract checklist: specs/meta/spec-layer-contract.yaml ; rollout: VISION-TRACEABILITY-ROLLOUT.md

lint-specs:
	@echo "[lint-specs] Validating spec YAML..."
	@cd "$(ROOT)" && $(PYTHON) generators/validate_specs.py specs
	@echo "[lint-specs] OK."

validate: lint-specs
	@echo "[validate] Parent chain + level consistency..."
	@cd "$(ROOT)" && $(PYTHON) spec-engine/validate_chain.py specs
	@echo "[validate] OK."

gen-graph:
	@echo "[gen-graph] Writing dependency graph..."
	@cd "$(ROOT)" && $(PYTHON) spec-engine/mermaid_gen.py specs \
		--output frontend/src/lib/generated/dependency-graph.mermaid
	@echo "[gen-graph] OK."

# --- Code generation ---

generate: lint-specs
	@echo "[generate] Running generators..."
	@cd "$(ROOT)" && $(PYTHON) generators/gen.py specs frontend
	@echo "[generate] OK."
	@echo "  Run: make dev"

# --- Spec-first self-correction loop ---
# 3-layer criteria:
#   L1 (structural): YAML syntax + parent chain   → validate_specs.py
#   L2 (semantic)  : file existence + imports      → spec_criteria_validator.py
#   L3 (behavioral): acceptance_criteria          → needs agent/runtime
#
# Flow: lint-specs → generate → spec-criteria → [gap修复循环]
#   self-heal   → 自动修正（template 同步等）
#   agent action → 推给 openclaw agent（critical gaps only）
#   skip         → 记录并继续

SPEC_CRITERIA_REPORT := /tmp/spec_criteria_report.json

spec-criteria: lint-specs generate
	@echo "[spec-criteria] Checking criteria (L1+L2)..."
	@cd "$(ROOT)" && $(PYTHON) generators/spec_criteria_validator.py \
		--level L1L2 --json --critical-only > $(SPEC_CRITERIA_REPORT) 2>&1 || true
	@echo "[spec-criteria] Done. Report: $(SPEC_CRITERIA_REPORT)"
	@cd "$(ROOT)" && $(PYTHON) generators/spec_criteria_validator.py \
		--level L1L2 --json --agent-targets 2>/dev/null || echo "No critical gaps."

# spec-iterate: 完整迭代循环（lint + generate + criteria + agent修正）
# dry-run by default; use SPEC_ITERATE_EXEC=1 to execute
spec-iterate: lint-specs generate
	@echo "[spec-iterate] Criteria check..."
	@cd "$(ROOT)" && $(PYTHON) generators/spec_criteria_validator.py \
		--level L1L2 --json > $(SPEC_CRITERIA_REPORT) 2>&1 || true
	@echo "[spec-iterate] Self-correction (dry-run)..."
	@cd "$(ROOT)" && $(PYTHON) generators/spec_self_corrector.py \
		--report $(SPEC_CRITERIA_REPORT) --dry-run
	@if [ "$(SPEC_ITERATE_EXEC)" = "1" ]; then \
		echo "[spec-iterate] Executing self-heal..."; \
		cd "$(ROOT)" && $(PYTHON) generators/spec_self_corrector.py \
			--report $(SPEC_CRITERIA_REPORT) --execute; \
	fi
	@echo "[spec-iterate] Done. Critical gaps: run SPEC_ITERATE_EXEC=1 make spec-iterate for auto-fix."

self-generate: validate generate spec-criteria
	@echo "[self-generate] Bootstrap complete."
	@echo "  Generated from this repo's specs."
	@echo "  Criteria report: $(SPEC_CRITERIA_REPORT)"

# --- Dev / build ---

build: generate
	@echo "[build] Production build..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "[build] OK."

dev: generate
	@echo "[dev] Stopping existing processes..."
	@cd $(ROOT) && go run scripts/kill-dev.go
	@sleep 1
	@echo "[dev] Starting Go backend..."
	@cd $(ROOT)/agent && WORKSPACE_ROOT=$(ROOT) go run ./cmd/web/ &
	@sleep 3
	@echo "[dev] Starting frontend..."
	@cd $(FRONTEND_DIR) && npm run dev

test:
	@echo "[test] Running frontend tests..."
	@cd $(FRONTEND_DIR) && npm test

drift:
	@echo "[drift] Checking schema drift..."
	@cd "$(ROOT)" && $(PYTHON) generators/drift_check.py specs frontend

lint-all: validate drift
	@echo "[lint-all] All checks passed."

clean:
	@echo "[clean] Removing generated files..."
	@rm -rf $(FRONTEND_DIR)/src/generated
	@rm -rf $(FRONTEND_DIR)/src/lib/generated
	@echo "[clean] OK."

# --- Skill Sync ---
# Bidirectional: live ~/.hermes/skills <-> repo skills/
# Tracked skills: SKILL.md repo_tracked: true
#
# make skill-sync-status    sync status
# make skill-sync-push      live -> repo
# make skill-sync-push SKILL=devops/vibex-qa-entry-points  single skill
# make skill-sync-pull      repo -> live

SKILLS_SCRIPT := $(ROOT)/scripts/skill-sync.sh
SKILLS_LIVE := $(ROOT)/skills
export SKILLS_LIVE_DIR := $(SKILLS_LIVE)

skill-sync-status:
	@SKILLS_LIVE_DIR=$(SKILLS_LIVE) bash $(SKILLS_SCRIPT) status

skill-sync-push:
	@SKILLS_LIVE_DIR=$(SKILLS_LIVE) bash $(SKILLS_SCRIPT) push $(SKILL)

skill-sync-pull:
	@SKILLS_LIVE_DIR=$(SKILLS_LIVE) bash $(SKILLS_SCRIPT) pull $(SKILL)

# ── Agent binary ────────────────────────────────────────────────
# agent/cmd/web is a separate Go module.
# Build result placed at backend/vibex-backend (where main.go OnDomReady looks for it).

.PHONY: agent-build
agent-build:
	@echo "[agent-build] Building Go agent (from $(ROOT))..."
	@cd $(ROOT)/agent && go build -o ../backend/vibex-backend.exe ./cmd/web
	@echo "[agent-build] OK → $(ROOT)/backend/vibex-backend.exe"

# ── Frontend build ─────────────────────────────────────────────
# Only builds the SvelteKit frontend (no spec generation).
# Use 'make build' if you need the full generate+build pipeline.

.PHONY: frontend-build
frontend-build:
	@echo "[frontend-build] Installing dependencies..."
	@cd $(FRONTEND_DIR) && npm install
	@echo "[frontend-build] Cleaning old build artifacts..."
	@rm -rf $(FRONTEND_DIR)/build
	@echo "[frontend-build] Building SvelteKit frontend..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "[frontend-build] Adding cache-busting version to index.html..."
	@echo "[cache-bust] v=$(FRONTEND_DIR)/build/index.html..."
	@$(PYTHON) $(ROOT)/scripts/cache_bust.py $(FRONTEND_DIR)/build/index.html
	@echo "[frontend-build] OK → frontend/build/"

# ── Wails Native Shell ──────────────────────────────────────
# Required: webkit2gtk-4.1 (not 4.0) on this system → tag webkit2_41
WAILS_TAGS := webkit2_41

# Detect OS via Go (reliable on WSL/Windows)
IS_WINDOWS := $(shell go env GOOS 2>/dev/null | grep -qi windows && echo 1 || echo 0)

# Detect headless: no DISPLAY + no WSLG display → use xvfb
IS_HEADLESS := $(shell if [ -z "$$DISPLAY" ] && [ -z "$$WAYLAND_DISPLAY" ]; then echo 1; else echo 0; fi)

# wails binary location
WAILS_BIN := $(shell which wails 2>/dev/null || echo /root/go/bin/wails)

# ── Windows hosts fix ─────────────────────────────────────────
# Windows can't resolve wails.localhost (no mDNS/Bonjour).
# Auto-add 127.0.0.1 wails.localhost to hosts file if needed.
.PHONY: wails-hosts-setup
wails-hosts-setup:
	@if [ "$(IS_WINDOWS)" = "1" ]; then \
		echo "[wails-hosts] Running hosts fix script..."; \
		powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$$(cygpath -w $(ROOT)/scripts/wails-hosts.ps1)"; \
	else \
		echo "[wails-hosts] Non-Windows: skipping hosts setup."; \
	fi

.PHONY: wails-dev
wails-dev: agent-build frontend-build wails-hosts-setup
	@echo "[wails-dev] Starting VibeX Workbench..."
	@if [ "$(IS_HEADLESS)" = "1" ]; then \
		cd $(ROOT) && GOFLAGS="-tags=$(WAILS_TAGS)" xvfb-run -a $(WAILS_BIN) dev -tags "$(WAILS_TAGS)"; \
	else \
		cd $(ROOT) && GOFLAGS="-tags=$(WAILS_TAGS)" $(WAILS_BIN) dev -tags "$(WAILS_TAGS)"; \
	fi

.PHONY: wails-dev-browser
wails-dev-browser: agent-build wails-hosts-setup
	@echo "[wails-dev-browser] Starting VibeX Workbench with devtools..."
	@if [ "$(IS_HEADLESS)" = "1" ]; then \
		cd $(ROOT) && GOFLAGS="-tags=$(WAILS_TAGS)" xvfb-run -a $(WAILS_BIN) dev -tags "$(WAILS_TAGS)" -devtools; \
	else \
		cd $(ROOT) && GOFLAGS="-tags=$(WAILS_TAGS)" $(WAILS_BIN) dev -tags "$(WAILS_TAGS)" -devtools; \
	fi

.PHONY: wails-build
wails-build: agent-build build
	@echo "[wails-build] Building production binary..."
	@cd $(ROOT) && wails build -tags "$(WAILS_TAGS)"
