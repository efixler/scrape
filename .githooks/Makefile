# Copy hooks into the .git/hooks directory

setup-hooks: ## setup the git hooks
	@find . -maxdepth 1 -type f ! -name 'Makefile' -exec cp {} ../.git/hooks/ \;
	@chmod +x ../.git/hooks/*
	@echo "Hooks copied to .git/hooks"

.PHONY: setup-hooks