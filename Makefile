.PHONY: build clean test lint docker tidy

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/passenger-datadog-monitor .

clean:
	rm -rf ./bin
	go clean -testcache

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

docker:
	docker build -t passenger-datadog-monitor:latest .

tidy:
	go mod tidy
