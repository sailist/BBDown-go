.PHONY: build test lint fmt vet clean

build:
	go build -o bbdown-go ./cmd/bbdown

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f bbdown-go
