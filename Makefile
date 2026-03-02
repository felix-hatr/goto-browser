BINARY  := zebro
VERSION ?= dev
LDFLAGS := -buildvcs=false -ldflags="-X 'main.Version=$(VERSION)'"
BINDIR  := bin

.PHONY: build test install clean lint tidy completions

build:
	@mkdir -p $(BINDIR)
	go build $(LDFLAGS) -o $(BINDIR)/$(BINARY) ./cmd/zebro

test:
	go test -buildvcs=false ./...

install:
	go install $(LDFLAGS) ./cmd/zebro

clean:
	rm -rf $(BINDIR)

lint:
	golangci-lint run

tidy:
	go mod tidy

completions: build
	@mkdir -p completions
	./$(BINDIR)/$(BINARY) completion bash > completions/zebro.bash
	./$(BINDIR)/$(BINARY) completion zsh > completions/zebro.zsh
	./$(BINDIR)/$(BINARY) completion fish > completions/zebro.fish

# Quick local run
run: build
	./$(BINDIR)/$(BINARY) $(ARGS)
