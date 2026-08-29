# ui-engine Makefile — удобная обёртка над task и go
# Использование: make build, make test, make dev, make clean

BIN := bin/ui-engine
WASM := bin/core.wasm
TS_OUT := bin/ts

.PHONY: all build test lint clean dev dev-app docs docs-dev docs-preview cover help install pull-build

all: build test lint ## полная проверка (build + test + lint)

build: ## собрать CLI и WASM
	@mkdir -p bin
	@cd cli && go build -o ../bin/ui-engine .
	@mkdir -p bin
	@cd core/wasm && GOOS=js GOARCH=wasm go build -o ../../bin/core.wasm .
	@npx tsc --outDir bin/ts 2>/dev/null || true
	@echo "✓ build: $(BIN) + $(WASM)"

test: ## тесты Go (core + cli)
	@cd core && go vet ./... && go test ./...
	@cd cli && go vet ./... && go test ./...

lint: ## валидация YAML примера
	@$(BIN) lint examples/counter || (echo "lint failed, run 'make build' first" && exit 1)

gen: ## генерация Go-типов из YAML
	@$(BIN) gen examples/counter

dev: ## dev-сервер примера + доки (5173)
	@trap 'kill 0' INT; $(BIN) dev examples/counter & npx vitepress dev docs --port 5173 & wait

dev-app: ## только dev-сервер примера (8033)
	@$(BIN) dev examples/counter

docs: ## собрать доку
	@npx vitepress build docs

docs-dev: ## dev-сервер доки (5173)
	@npx vitepress dev docs --port 5173

docs-preview: ## превью собранной доки
	@npx vitepress preview docs

serve: ## статический сервер собранного примера
	@$(BIN) serve examples/counter

clean: ## удалить артефакты
	@rm -rf bin build dist coverage node_modules/.cache
	@rm -rf examples/counter/bin
	@rm -rf examples/counter/generated
	@echo "✓ clean"

cover: ## покрытие тестами
	@cd core && go test -coverprofile=../bin/cover.out ./... && go tool cover -html=../bin/cover.out -o ../bin/cover.html
	@echo "✓ coverage: bin/cover.html"

install: ## установить CLI в /usr/local/bin
	@cp $(BIN) /usr/local/bin/ui-engine
	@echo "✓ installed to /usr/local/bin/ui-engine"

pull-build: ## тянуть готовый билд с Releases
	@./scripts/pull-build.sh

help: ## показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

# Алиасы для task
task-%:
	@task $*
