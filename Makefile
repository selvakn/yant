.PHONY: all help build build-frontend run test integration-test coverage clean deps lint docker-build docker-run

# Configurable variables (override at command line: make run ADDR=:9090)
BINARY            := ./bin/server
ADDR              := :8080
DB                := ./notes.db
NOTES_DIR         := ./notes
UPLOADS_DIR       := ./uploads
COVERAGE_THRESHOLD := 90
TEST_FLAGS        :=
DOCKER_IMAGE      := yant
DOCKER_TAG        := latest

all: help

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | \
	 awk 'BEGIN{FS=":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Compile the server binary to $(BINARY)
	mkdir -p bin
	cd backend && go build -o ../$(BINARY) ./cmd/server

build-frontend: ## Build tldraw bundle (requires Node.js)
	cd frontend-build && npm install && npm run build

run: build ## Build and start the server (default: :8080)
	$(BINARY) -addr $(ADDR) -db $(DB) -notes $(NOTES_DIR) -uploads $(UPLOADS_DIR)

test: ## Run the full test suite
	cd backend && go test $(TEST_FLAGS) ./...

coverage: ## Run tests and enforce ≥90% line coverage
	cd backend && go test ./internal/... -coverpkg=./internal/... \
	    -coverprofile=../coverage.out $(TEST_FLAGS)
	@PCTG=$$(cd backend && go tool cover -func=../coverage.out | tail -1 | \
	    awk '{gsub(/%/,""); print int($$3)}'); \
	 echo "Coverage: $$PCTG%"; \
	 if [ "$$PCTG" -lt "$(COVERAGE_THRESHOLD)" ]; then \
	   echo "FAIL: coverage $$PCTG% < $(COVERAGE_THRESHOLD)%"; exit 1; \
	 fi

lint: ## Run go vet static analysis
	cd backend && go vet ./...

clean: ## Remove build artifacts (bin/, coverage.out)
	rm -rf ./bin ./coverage.out

deps: ## Tidy and download Go module dependencies
	cd backend && go mod tidy && go mod download

integration-test: docker-build ## Run integration tests against Docker image
	cd backend && go test -tags integration -timeout 300s -v ./internal/integration/...

docker-build: ## Build Docker image (DOCKER_IMAGE=yant DOCKER_TAG=latest)
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run container with persistent data volume
	docker run --rm -p $(ADDR):8080 -v yant-data:/data $(DOCKER_IMAGE):$(DOCKER_TAG)
