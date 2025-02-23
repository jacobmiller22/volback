.DEFAULT: 
	build
.PHONY: fmt vet build test

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build -o ./bin/volume-backup ./cmd/volume-backup

test: 
	go test -v ./...
