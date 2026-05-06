.PHONY: web-deps web-deps-alpine

ALPINE_VERSION ?= 3.15.12

web-deps: web-deps-alpine ## Prepare local frontend vendor assets

web-deps-alpine: ## Download Alpine.js to static vendor dir (skips if present; FORCE=1 to redownload)
	@mkdir -p web/static/vendor/alpinejs
	@if [ -f web/static/vendor/alpinejs/alpine.min.js ] && [ "$(FORCE)" != "1" ]; then \
		echo "Already exists: web/static/vendor/alpinejs/alpine.min.js (use FORCE=1 to redownload)"; \
	else \
		echo "Downloading Alpine.js $(ALPINE_VERSION)..."; \
		curl -fsSL "https://unpkg.com/alpinejs@$(ALPINE_VERSION)/dist/cdn.min.js" -o web/static/vendor/alpinejs/alpine.min.js; \
		echo "Saved: web/static/vendor/alpinejs/alpine.min.js"; \
	fi

