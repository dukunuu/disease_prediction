# Makefile — run migrations inside Docker
COMPOSE      = docker-compose
SERVICE      = backend

# Use exec for a running container; or use `run --rm` if you want a
# fresh one-off container each time
DCEXEC       = $(COMPOSE) exec -T $(SERVICE)

# assemble the migration command, escaping $ so it's evaluated inside
# the container
MIGRATE      = migrate -database $${DB_URL} -path db/schema 

.PHONY: help create-migration migrate-up migrate-down migrate-to \
        migrate-version

.DEFAULT_GOAL := help

help:
	@echo "Available targets:"
	@echo "  create-migration  Create a new SQL migration (prompts for name)"
	@echo "  migrate-up        Apply migrations (prompts for steps, blank=all)"
	@echo "  migrate-down      Roll back migrations (prompts for steps, blank=1)"
	@echo "  migrate-to        Migrate to a specific version (prompts for ver)"
	@echo "  migrate-version   Show current migration version"
	@echo "	 sqlcgen           Generate sqlc schema and queries"

create-migration:
	@read -p "Migration name (e.g. add_users_table): " name; \
	if [ -z "$$name" ]; then \
	  echo "❌ Migration name is required."; exit 1; \
	fi; \
	$(COMPOSE) exec -T $(SERVICE) sh -c \
	  "migrate create -ext sql -dir db/schema -seq $$name"

migrate-up:
	@read -p "Number of steps to apply (blank = all): " steps; \
	if [ -z "$$steps" ]; then \
	  $(DCEXEC) sh -c '$(MIGRATE) up'; \
	else \
	  $(DCEXEC) sh -c '$(MIGRATE) up $$steps'; \
	fi

migrate-down:
	@read -p "Number of steps to revert (blank = 1): " steps; \
	if [ -z "$$steps" ]; then \
	  $(DCEXEC) sh -c '$(MIGRATE) down 1'; \
	else \
	  $(DCEXEC) sh -c '$(MIGRATE) down $$steps'; \
	fi

migrate-to:
	@read -p "Target version (e.g. 5): " version; \
	if [ -z "$$version" ]; then \
	  echo "❌ Version is required."; exit 1; \
	fi; \
	$(DCEXEC) sh -c '$(MIGRATE) goto $$version'

migrate-version:
	$(DCEXEC) sh -c '$(MIGRATE) version'

sqlcgen:
	$(DCEXEC) sh -c 'sqlc generate'

