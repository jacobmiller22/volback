.PHONY: fmt vet build test
all: fmt vet test build 

build: vet
	go build -o ./bin/volback ./cmd/volback

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: 
	go test -v ./...
