.PHONY: all
all: fmt gomod readme

.PHONY: fmt
fmt:
	@echo "format go project..."
	@gofmt -s -w . 2>&1

.PHONY: gomod
gomod:
	@echo "running go mod tidy..."
	@go mod tidy

.PHONY: readme
readme:
	@echo "running doctoc..."
	@doctoc . 2>&1
