.PHONY: css css-build css-watch

css: css-build ## Alias for css-build

css-build: ## Build Tailwind CSS (minified)
	@npx tailwindcss@3.4.17 --config tailwind.config.js -i web/static/css/input.css -o web/static/css/tailwind.css --minify

css-watch: ## Watch and rebuild Tailwind CSS on changes
	@npx tailwindcss@3.4.17 --config tailwind.config.js -i web/static/css/input.css -o web/static/css/tailwind.css --watch

