.PHONY: build test vet cover docker clean

BINARY  := model-server
BINDIR  := bin

build:
	go build -trimpath -o $(BINDIR)/$(BINARY) ./cmd/$(BINARY)

test:
	go test ./... -race -count=1

vet:
	go vet ./...

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | grep total

docker:
	docker build -t $(BINARY):latest -f deploy/docker/Dockerfile .

clean:
	rm -rf $(BINDIR)/ coverage.out
