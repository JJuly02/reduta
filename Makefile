GO_IMG := golang:1.23

.PHONY: up down logs ps test tidy vet fmt
up: ; docker compose up --build
down: ; docker compose down -v
logs: ; docker compose logs -f server
ps: ; docker compose ps
test: ; docker run --rm -v $$PWD:/src -w /src $(GO_IMG) go test ./...
vet: ; docker run --rm -v $$PWD:/src -w /src $(GO_IMG) go vet ./...
tidy: ; docker run --rm -v $$PWD:/src -w /src $(GO_IMG) go mod tidy
fmt: ; docker run --rm -v $$PWD:/src -w /src $(GO_IMG) gofmt -w .
