.PHONY: build test vet lint fmt

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fmt:
	./scripts/check-gofmt.sh
