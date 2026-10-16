.PHONY: build test run

build:
	go build -o {{PROJECT_NAME_KEBAB}} .

test:
	go vet ./...
	go test ./...

run:
	go run . -name you
