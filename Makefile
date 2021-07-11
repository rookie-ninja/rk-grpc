.PHONY: all
all: test lint readme fmt

.PHONY: lint
lint:
	@echo "running golangci-lint..."
	@golangci-lint run 2>&1

.PHONY: test
test:
	@echo "running go test..."
	@go test ./... 2>&1

.PHONY: fmt
fmt:
	@echo "format go project..."
	@gofmt -s -w . 2>&1

.PHONY: readme
readme:
	@echo "running doctoc..."
	@doctoc . 2>&1

.PHONY: buf
buf:
	@echo "running buf..."
	@buf generate --path boot/api/v1

.PHONY: pkger
pkger:
	@echo "running pkger..."
	@pkger -o boot
