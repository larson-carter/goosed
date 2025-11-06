MODULES := $(shell find . -name go.mod -not -path "./vendor/*")
SERVICES := api bootd orchestrator blueprints inventory artifacts-gw pxe-stack

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
