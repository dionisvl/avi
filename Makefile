-include .env
export

dev:
	COMPOSE_PROFILES=dev docker compose -f compose.yml -f compose.override.dev.yml up -d

dev-build: fe-admin-build
	COMPOSE_PROFILES=dev docker compose -f compose.yml -f compose.override.dev.yml up -d --build

prod:
	docker compose --compatibility -f compose.yml -f compose.override.prod.yml up -d

prod-build: fe-admin-build
	docker compose --compatibility -f compose.yml -f compose.override.prod.yml up -d --build

down:
	docker compose -f compose.yml down

migrate:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_DSN_EXTERNAL)" $$(go env GOPATH)/bin/goose -dir api-go/internal/migrations up

migrate-status:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DB_DSN_EXTERNAL)" $$(go env GOPATH)/bin/goose -dir api-go/internal/migrations status

db-drop:
	docker compose -f compose.yml stop db
	docker compose -f compose.yml rm -f db
	docker volume rm $$(docker compose -f compose.yml ls --format json | jq -r '.[] | select(.Name != null) | .Name' | head -n1)_postgres_data
	docker compose -f compose.yml up -d db
	docker compose -f compose.yml restart api

logs:
	docker compose -f compose.yml logs -f

test:
	docker compose -f compose.yml exec api go test ./internal/... -v -count=1
	docker compose -f compose.yml exec api go test ./tests ./tests/integration -v -count=1

fe-admin-build:
	npm --prefix fe-admin run build

fe-admin:
	npm --prefix fe-admin test

lint:
	docker compose -f compose.yml exec api gci write -s standard -s default -s "prefix(github.com/dionisvl/avi)" .
	docker compose -f compose.yml exec api sh -lc 'out=$$(/go/bin/gofumpt -l .); if [ -n "$$out" ]; then echo "Files need gofumpt formatting:"; echo "$$out"; exit 1; fi'
	docker compose -f compose.yml exec api golangci-lint run ./...

modernize:
	docker compose -f compose.yml exec api /go/bin/modernize -fix ./...
	docker compose -f compose.yml exec api gofumpt -w .
	docker compose -f compose.yml exec api gci write -s standard -s default -s "prefix(github.com/dionisvl/avi)" .

sw:
	docker compose -f compose.yml -f compose.override.dev.yml exec api swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

sh:
	docker compose -f compose.yml -f compose.override.dev.yml exec api sh

# DEPLOY to prod
deploy-api:
	ssh -p $(DEPLOY_PORT) $(DEPLOY_USER)@$(DEPLOY_HOST) "cd /home/avi && git pull && docker compose -f compose.yml -f compose.override.prod.yml up -d --build --no-deps api"

deploy-fe: fe-admin-build
	ssh -p $(DEPLOY_PORT) $(DEPLOY_USER)@$(DEPLOY_HOST) "cd /home/avi && git pull && mkdir -p fe-admin/dist"
	rsync -az --delete --exclude env.js -e "ssh -p $(DEPLOY_PORT)" fe-admin/dist/ $(DEPLOY_USER)@$(DEPLOY_HOST):/home/avi/fe-admin/dist/
	ssh -p $(DEPLOY_PORT) $(DEPLOY_USER)@$(DEPLOY_HOST) "cd /home/avi && docker compose -f compose.yml -f compose.override.prod.yml up -d --no-deps fe-admin"

deploy-all:
	$(MAKE) deploy-api
	$(MAKE) deploy-fe
