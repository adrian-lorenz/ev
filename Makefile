.PHONY: build install run tidy

build:
	go build -ldflags="-s -w -X main.version=v$(shell cat VERSION)" -o ev .

install:
	go install .

run:
	go run . $(ARGS)

tidy:
	go mod tidy
