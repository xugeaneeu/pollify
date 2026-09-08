COMPOSE      := docker compose -f docker-compose.prod.yml
DEV_COMPOSE  := docker compose
API_BASE     := http://127.0.0.1
ADMIN_PW     := admin123

.PHONY: help up down restart reset seed-admins wait-api logs ps psql shell-api dev-up dev-down

help:
	@echo "Pollify dev / local-prod helpers"
	@echo ""
	@echo "Production-style stack (Caddy + api + postgres on http://localhost):"
	@echo "  make up           build, start, and seed admin1..3@gmail.com / $(ADMIN_PW)"
	@echo "  make down         stop the stack (keeps DB volume)"
	@echo "  make reset        wipe DB + restart + reseed"
	@echo "  make restart      down + up (keeps DB)"
	@echo "  make seed-admins  (re)create the three admin accounts"
	@echo "  make logs         tail logs (all services)"
	@echo "  make psql         open psql shell against the postgres container"
	@echo "  make shell-api    open an /bin/sh in the api container"
	@echo ""
	@echo "Backend-only dev stack (api + postgres on :8080 / :5432):"
	@echo "  make dev-up       start dev stack"
	@echo "  make dev-down     stop dev stack"

# ── prod-style local stack ───────────────────────────────────────────

.env:
	@cp .env.example .env
	@sed -i 's|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=local-dev-pw|' .env
	@sed -i 's|^JWT_SECRET=.*|JWT_SECRET=local-dev-secret-not-for-prod|' .env
	@chmod 600 .env
	@echo "→ created .env with local-dev defaults (edit before any real deploy)"

up: .env
	$(COMPOSE) up -d --build
	@$(MAKE) --no-print-directory wait-api
	@$(MAKE) --no-print-directory seed-admins
	@echo ""
	@echo "✓ Pollify is up at $(API_BASE)"
	@echo "  admin accounts: admin1@gmail.com / $(ADMIN_PW)"
	@echo "                  admin2@gmail.com / $(ADMIN_PW)"
	@echo "                  admin3@gmail.com / $(ADMIN_PW)"

wait-api:
	@printf "→ waiting for api"
	@for i in $$(seq 1 60); do \
		if curl -fsS $(API_BASE)/healthz >/dev/null 2>&1; then echo " ready"; exit 0; fi; \
		printf "."; sleep 1; \
	done; \
	echo " timeout"; \
	exit 1

seed-admins:
	@echo "→ registering admin1..3 via /api/v1/auth/register"
	@for i in 1 2 3; do \
		curl -sS -o /dev/null -w "  admin$$i@gmail.com → %{http_code}\n" \
			-X POST $(API_BASE)/api/v1/auth/register \
			-H "Content-Type: application/json" \
			-d "{\"email\":\"admin$$i@gmail.com\",\"password\":\"$(ADMIN_PW)\",\"display_name\":\"Admin $$i\"}" || true; \
	done
	@echo "→ promoting admin1..3 to ADMIN"
	@$(COMPOSE) exec -T postgres psql -U pollify -d pollify -q \
		-c "UPDATE users SET role='ADMIN' WHERE email IN ('admin1@gmail.com','admin2@gmail.com','admin3@gmail.com');"

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) down
	@$(MAKE) --no-print-directory up

reset:
	$(COMPOSE) down -v
	@$(MAKE) --no-print-directory up

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

psql:
	$(COMPOSE) exec postgres psql -U pollify -d pollify

shell-api:
	$(COMPOSE) exec api /bin/sh

# ── dev-only stack (api + postgres without Caddy) ────────────────────

DEV_API_BASE := http://127.0.0.1:8080

dev-up:
	$(DEV_COMPOSE) up -d --build

dev-down:
	$(DEV_COMPOSE) down

dev-seed-admins:
	@printf "→ waiting for dev api"
	@for i in $$(seq 1 60); do \
		if curl -fsS $(DEV_API_BASE)/healthz >/dev/null 2>&1; then echo " ready"; break; fi; \
		printf "."; sleep 1; \
	done
	@echo "→ registering admin1..3 via /api/v1/auth/register"
	@for i in 1 2 3; do \
		curl -sS -o /dev/null -w "  admin$$i@gmail.com → %{http_code}\n" \
			-X POST $(DEV_API_BASE)/api/v1/auth/register \
			-H "Content-Type: application/json" \
			-d "{\"email\":\"admin$$i@gmail.com\",\"password\":\"$(ADMIN_PW)\",\"display_name\":\"Admin $$i\"}" || true; \
	done
	@echo "→ promoting admin1..3 to ADMIN"
	@$(DEV_COMPOSE) exec -T postgres psql -U pollify -d pollify -q \
		-c "UPDATE users SET role='ADMIN' WHERE email IN ('admin1@gmail.com','admin2@gmail.com','admin3@gmail.com');"
	@echo ""
	@echo "✓ admin accounts ready:"
	@echo "  admin1@gmail.com / $(ADMIN_PW)"
	@echo "  admin2@gmail.com / $(ADMIN_PW)"
	@echo "  admin3@gmail.com / $(ADMIN_PW)"
