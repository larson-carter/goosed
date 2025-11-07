MODULES := $(shell find . -name go.mod -not -path "./vendor/*")
SERVICES := api bootd orchestrator blueprints inventory artifacts-gw pxe-stack

UI_API_IMAGE ?= goosed/ui-api:dev
UI_WEB_IMAGE ?= goosed/ui-web:dev

.PHONY: tidy lint test build run-api run-all

tidy:
	@for mod in $(MODULES); do \
		echo "==> $$mod"; \
		(cd $$(dirname $$mod) && go mod tidy); \
	done

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

test:
	@for mod in $(MODULES); do \
		echo "==> $$mod"; \
		(cd $$(dirname $$mod) && go test ./...); \
	done

build:
	@for svc in $(SERVICES); do \
		echo "==> services/$$svc"; \
		docker build -f services/$$svc/Dockerfile -t goosed/$$svc:dev .; \
	done

run-api:
	go run services/api/cmd/api/main.go

run-all:
        @echo "run-all is not yet implemented"

.PHONY: ui-build
ui-build:
	@docker build -f services/ui/api/Dockerfile -t $(UI_API_IMAGE) services/ui/api
	@docker build -f services/ui/web/Dockerfile -t $(UI_WEB_IMAGE) services/ui/web
