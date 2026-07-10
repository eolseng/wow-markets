WAILS := go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0

.PHONY: check addon-check companion companion-build companion-check contract-check fmt

check: contract-check addon-check companion-check

contract-check:
	cd contracts/scan-archive/v1 && shasum -a 256 -c checksums.txt

addon-check:
	@LUA_BIN="$$(command -v lua5.1 || command -v luajit || command -v lua)"; \
	test -n "$$LUA_BIN" || (echo "Lua 5.1 is required for addon-check" >&2; exit 1); \
	"$$LUA_BIN" addon/tests/addon_test.lua addon

companion:
	cd companion && $(WAILS) dev

companion-build:
	cd companion && $(WAILS) build

companion-check:
	cd companion && test -z "$$(gofmt -l internal *.go)" \
		&& node --check frontend/dist/app.js \
		&& node --check frontend/dist/view-model.mjs \
		&& node --test frontend/view-model.test.mjs \
		&& go test ./... \
		&& go build ./...

fmt:
	cd companion && gofmt -w internal *.go
