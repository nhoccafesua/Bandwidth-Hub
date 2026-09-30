.PHONY: all build build-linux run run-python clean test-ram install

BINARY_NAME=bandwidth-hub
PORT=8888

all: build

build:
	@echo "==> Đang build Bandwidth Hub binary (Native OS)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/server
	@echo "[✓] Build hoàn tất: $(BINARY_NAME)"

build-linux:
	@echo "==> Đang cross-compile cho Linux AMD64 (VPS)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 ./cmd/server
	@echo "[✓] Binary Linux sẵn sàng: $(BINARY_NAME)-linux-amd64"

run: build
	./$(BINARY_NAME) -port $(PORT)

run-python:
	@echo "==> Khởi động Bandwidth Hub qua Python 3 siêu nhẹ (Zero-deps)..."
	python3 server.py $(PORT)

test-ram:
	bash scripts/trim_memory.sh --force

install:
	sudo bash scripts/install.sh

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-amd64
