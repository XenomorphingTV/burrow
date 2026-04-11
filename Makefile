BINARY   := burrow
MODULE   := github.com/xenomorphingtv/burrow
PREFIX   ?= /usr/local
BINDIR   := $(PREFIX)/bin
MANDIR   := $(PREFIX)/share/man/man1

VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-s -w -X $(MODULE)/internal/version.Version=$(VERSION)"

.PHONY: build test vet check install install-man uninstall-man clean release

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/burrow

test:
	go test ./...

vet:
	go vet ./...

check: vet test

install: build
	install -Dm755 $(BINARY) $(BINDIR)/$(BINARY)
	$(MAKE) install-man

install-man:
	install -Dm644 burrow.1 $(MANDIR)/burrow.1

uninstall-man:
	rm -f $(MANDIR)/burrow.1

release:
	goreleaser release --clean

clean:
	rm -f $(BINARY)
