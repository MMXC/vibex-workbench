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
	@echo "  make self-generate Self-bootstrap (validate + generate)"
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

self-generate: validate generate
	@echo "[self-generate] Bootstrap complete."
	@echo "  Generated from this repo's specs."

# --- Dev / build ---

build: generate
	@echo "[build] Production build..."
	@cd $(FRONTEND_DIR) && npm run build
	@echo "[build] OK."

dev: generate
	@echo "[dev] Starting dev server..."
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
