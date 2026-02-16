.PHONY: migrate-up migrate-down migrate-status migrate-create migrate-reset graphql dev server debug readme seed seed-avatars convert-avatars test test-db-setup setup swagger gotestsum-install test-dots go-tests lint check

GO := $(shell which go 2>/dev/null || echo /opt/homebrew/bin/go)
DOCKER_COMPOSE := $(shell if command -v docker-compose >/dev/null 2>&1; then echo docker-compose; else echo "docker compose"; fi)

migrate-up:
	$(GO) run cmd/migrate/main.go -command up

migrate-down:
	$(GO) run cmd/migrate/main.go -command down

migrate-status:
	$(GO) run cmd/migrate/main.go -command status

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	$(GO) run cmd/migrate/main.go -command create $(NAME)

migrate-reset:
	@echo "Dropping and recreating database..."
	@docker-compose exec -T postgres psql -U postgres -c "DROP DATABASE IF EXISTS moonshine;" 2>/dev/null || true
	@docker-compose exec -T postgres psql -U postgres -c "CREATE DATABASE moonshine;" 2>/dev/null || true

dev:
	@if command -v air > /dev/null; then \
		air; \
	elif [ -f ~/go/bin/air ]; then \
		~/go/bin/air; \
	else \
		echo "air not found. Install it with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

server: dev

debug:
	@if command -v dlv > /dev/null; then \
		dlv debug ./cmd/server --headless --listen=:2345 --api-version=2 --accept-multiclient; \
	elif [ -f ~/go/bin/dlv ]; then \
		~/go/bin/dlv debug ./cmd/server --headless --listen=:2345 --api-version=2 --accept-multiclient; \
	else \
		echo "delve not found. Install it with: go install github.com/go-delve/delve/cmd/dlv@latest"; \
		exit 1; \
	fi

readme:
	@if command -v glow > /dev/null; then \
		glow README.md; \
	elif [ -f ~/go/bin/glow ]; then \
		~/go/bin/glow README.md; \
	else \
		echo "glow not found. Install it with: go install github.com/charmbracelet/glow@latest"; \
		echo "Or use VS Code: Press Ctrl+Shift+V to preview markdown"; \
		exit 1; \
	fi

seed:
	$(GO) run cmd/seed/main.go

setup: migrate-reset migrate-up seed
	@echo "Database setup completed!"

test-db-setup:
	@echo "Setting up test database..."
	@echo "Starting postgres container..."
	@$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for postgres to become ready..."
	@for i in $$(seq 1 60); do \
		if $(DOCKER_COMPOSE) exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then \
			echo "Postgres is ready"; \
			break; \
		fi; \
		if [ $$i -eq 60 ]; then \
			echo "Postgres did not become ready in time"; \
			exit 1; \
		fi; \
		sleep 1; \
	done
	@echo "Applying migrations to test database (database will be created automatically if needed)..."
	@DATABASE_NAME=moonshine_test $(GO) run cmd/migrate/main.go -command up

test: test-db-setup
	@DATABASE_NAME=moonshine_test $(GO) test ./... -v 2>&1 | tee /tmp/test_output.txt | awk ' \
	BEGIN { main_pass=0; main_fail=0; sub_pass=0; sub_fail=0 } \
	/^--- PASS/ { main_pass++ } \
	/^--- FAIL/ { main_fail++ } \
	/^    --- PASS/ { sub_pass++ } \
	/^    --- FAIL/ { sub_fail++ } \
	END { \
		total_pass = main_pass + sub_pass; \
		total_fail = main_fail + sub_fail; \
		print ""; \
		print "=== Статистика тестов ==="; \
		print ""; \
		print "Основных тестов:"; \
		print "  ✓ Пройдено: " main_pass; \
		print "  ✗ Провалено: " main_fail; \
		print ""; \
		print "Подтестов:"; \
		print "  ✓ Пройдено: " sub_pass; \
		print "  ✗ Провалено: " sub_fail; \
		print ""; \
		print "Всего:"; \
		print "  ✓ Пройдено: " total_pass; \
		print "  ✗ Провалено: " total_fail; \
		print ""; \
		if (total_fail > 0) exit 1 \
	}'

gotestsum-install:
	$(GO) install gotest.tools/gotestsum@latest

test-dots: test-db-setup
	@if command -v gotestsum > /dev/null; then \
		DATABASE_NAME=moonshine_test gotestsum --format dots -- -count=1 ./...; \
	elif [ -f ~/go/bin/gotestsum ]; then \
		DATABASE_NAME=moonshine_test ~/go/bin/gotestsum --format dots -- -count=1 ./...; \
	else \
		echo "gotestsum not found. Install it with: make gotestsum-install"; \
		exit 1; \
	fi

go-tests: test-dots

lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	elif [ -f ~/go/bin/golangci-lint ]; then \
		~/go/bin/golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install it with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

check: lint go-tests

swagger:
	@if command -v swag > /dev/null; then \
		swag init -g cmd/server/main.go -o cmd/server/docs; \
	elif [ -f ~/go/bin/swag ]; then \
		~/go/bin/swag init -g cmd/server/main.go -o cmd/server/docs; \
	else \
		echo "swag not found. Install it with: go install github.com/swaggo/swag/cmd/swag@latest"; \
		exit 1; \
	fi

convert-avatars:
	@if command -v convert > /dev/null || command -v magick > /dev/null; then \
		cd frontend/assets/images/players/avatars && \
		counter=1 && \
		for file in *.gif; do \
			if [ -f "$$file" ]; then \
				if command -v convert > /dev/null; then \
					convert "$$file" "$$counter.png" && \
					echo "Converted $$file to $$counter.png"; \
				elif command -v magick > /dev/null; then \
					magick "$$file" "$$counter.png" && \
					echo "Converted $$file to $$counter.png"; \
				fi && \
				counter=$$((counter + 1)); \
			fi; \
		done && \
		echo "Conversion complete. You can now delete .gif files if needed."; \
	else \
		echo "ImageMagick not found. Install it first:"; \
		echo "  Ubuntu/Debian: sudo apt-get install imagemagick"; \
		echo "  macOS: brew install imagemagick"; \
		echo "  Or convert manually using online tools"; \
		exit 1; \
	fi
