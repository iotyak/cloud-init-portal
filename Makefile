BINARY := cloud-init-portal

.PHONY: build run test vet fmt

build:
	go build -o $(BINARY) .

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w *.go
