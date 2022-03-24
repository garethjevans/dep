.PHONY: build
build:
	go build -o build/dep cmd/main/main.go

test:
	go test -v ./...
