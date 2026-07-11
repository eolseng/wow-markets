WAILS := go run github.com/wailsapp/wails/v2/cmd/wails@v2.13.0

.PHONY: addon-check addon-package check companion companion-build companion-check contract-check fmt release-check

check: contract-check addon-check companion-check

release-check:
	./scripts/release/validate-version.sh companion "$$(jq -r .info.productVersion companion/wails.json)"
	./scripts/release/test-windows-file-version.sh
	./scripts/release/validate-version.sh addon "$$(awk -F ': ' '/^## Version: / { print $$2; exit }' addon/WoWMarkets/WoWMarkets.toc)"
	./scripts/release/package-addon.sh dist/wow-markets-addon.zip

addon-package:
	./scripts/release/package-addon.sh dist/wow-markets-addon.zip

contract-check:
	cd contracts/scan-archive/v1 && shasum -a 256 -c checksums.txt

addon-check:
	@LUA_BIN="$$(command -v lua5.1 || command -v luajit || command -v lua)"; \
	test -n "$$LUA_BIN" || (echo "Lua 5.1 is required for addon-check" >&2; exit 1); \
	"$$LUA_BIN" addon/tests/addon_test.lua addon
	./scripts/release/validate-addon-distribution.sh

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
