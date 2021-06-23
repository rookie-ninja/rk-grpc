.PHONY: all
all: buf readme

.PHONY: buf
buf:
	@echo "running buf..."
	@buf generate

.PHONY: readme
readme:
	@echo "running doctoc..."
	@doctoc . 2>&1