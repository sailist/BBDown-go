.PHONY: build test lint fmt vet clean

build:
	go build -o bbdown ./cmd/bbdown

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f bbdown bbdown-go
