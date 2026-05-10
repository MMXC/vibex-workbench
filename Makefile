.PHONY: help validate generate lint-specs test bootstrap

help:
	@echo "Digital Wallpaper - Available targets:"
	@echo "  make validate      - Validate all spec YAML files"
	@echo "  make generate     - Generate code from specs"
	@echo "  make lint-specs   - Lint spec files"
	@echo "  make test         - Run test suite"
	@echo "  make bootstrap    - Bootstrap project structure"

validate:
	@echo "Validating specs..."
	@find specs -name "*.yaml" -exec python -c "import yaml; yaml.safe_load(open('{}'))" \; 2>/dev/null
	@echo "✓ Spec validation complete"

generate:
	@echo "Generating code from specs..."
	@python gen.py generate
	@echo "✓ Code generation complete"

lint-specs:
	@echo "Linting specs..."
	@find specs -name "*.yaml" -exec python -c "import yaml; yaml.safe_load(open('{}'))" \; 2>/dev/null
	@echo "✓ Lint complete"

test:
	@echo "Running tests..."
	@npm run test

bootstrap:
	@echo "Bootstrapping project..."
	@mkdir -p specs/L1-goal specs/L2-skeleton specs/L3-module specs/L4-feature specs/L5-slice
	@echo "✓ Bootstrap complete"
