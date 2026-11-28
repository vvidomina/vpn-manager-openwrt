# Сборка и проверки этапа 1 (Go skeleton). Цели установки на роутер нет:
# пакет и разворачивание на AX3000T в этап 1 не входят (№44 §5.2).
#
# GO можно переопределить: make GO=/path/to/go build

GO ?= go
BIN ?= vpnmand
OUTDIR ?= build
OUT ?= $(OUTDIR)/$(BIN)
PKG ?= ./cmd/vpnmand

.PHONY: all build test vet fmt fmt-check cross clean

all: fmt-check vet test build

# Каталог сборки создаётся до компилятора: в чистом клоне $(OUTDIR) ещё нет.
build:
	@mkdir -p $(OUTDIR)
	$(GO) build -o $(OUT) $(PKG)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

# Проверка форматирования без изменения файлов: расхождения — ошибка.
fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt требуется для:"; echo "$$out"; exit 1; fi

# Кросс-компиляция под целевую платформу без cgo (вне сборочного SDK OpenWrt).
cross:
	@mkdir -p $(OUTDIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -o $(OUTDIR)/$(BIN)-linux-arm64 $(PKG)

clean:
	rm -rf $(OUTDIR)
