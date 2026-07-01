.PHONY: test-integration test-resend fmt imports lint modernize sw sw-check fix

CURRENT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ROOT_DIR := $(abspath $(CURRENT_DIR)/..)

DOCKER_COMPOSE = docker compose -f $(ROOT_DIR)/compose.yml
DOCKER_COMPOSE_DEV = docker compose -f $(ROOT_DIR)/compose.yml -f $(ROOT_DIR)/compose.override.dev.yml

test-integration:
	$(DOCKER_COMPOSE_DEV) exec api bash -c 'RUN_INTEGRATION_TESTS=true go test ./tests/integration/... -v -count=1'

# Runs real Resend email send. Override recipient: make test-resend TO=you@example.com
test-resend:
	RUN_RESEND_TESTS=true \
	SMTP_PASSWORD=$(shell grep '^SMTP_PASSWORD=' $(ROOT_DIR)/.env | cut -d= -f2) \
	SMTP_FROM=$(shell grep '^SMTP_FROM=' $(ROOT_DIR)/.env | cut -d= -f2) \
	RESEND_TEST_TO=$(or $(TO),$(shell grep '^RESEND_TEST_TO=' $(ROOT_DIR)/.env | cut -d= -f2)) \
	FRONTEND_DOMAIN=$(shell grep '^FRONTEND_DOMAIN=' $(ROOT_DIR)/.env | cut -d= -f2) \
	go test ./tests/integration/ -run TestResendSender -v -timeout 5s

fmt:
	$(DOCKER_COMPOSE) exec api gofumpt -w .

imports:
	$(DOCKER_COMPOSE) exec api gci write -s standard -s default -s "prefix(github.com/dionisvl/avi)" .

lint:
	$(DOCKER_COMPOSE) exec api golangci-lint run --config=.golangci.yml ./...

modernize:
	$(DOCKER_COMPOSE) exec api /go/bin/modernize -fix ./...
	$(MAKE) fmt
	$(MAKE) imports

fix:
	$(MAKE) fmt
	$(MAKE) imports

sw:
	$(DOCKER_COMPOSE_DEV) exec api swag init -g cmd/api/main.go -o docs --outputTypes json,yaml --parseDependency --parseInternal

sw-check:
	$(DOCKER_COMPOSE_DEV) exec api swag init -g cmd/api/main.go -o docs --outputTypes json,yaml --parseDependency --parseInternal
	git diff --exit-code -- docs/swagger.json docs/swagger.yaml
