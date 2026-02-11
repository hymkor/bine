ifeq ($(OS),Windows_NT)
    SHELL=CMD.EXE
    SET=set
    WHICH=where.exe
    DEL=del
    NUL=nul
else
    SET=export
    WHICH=which
    DEL=rm
    NUL=/dev/null
endif
ifndef GO
    SUPPORTGO=go1.20.14
    GO:=$(shell $(WHICH) $(SUPPORTGO) 2>$(NUL) || echo go)
endif
NAME=$(notdir $(CURDIR))
VERSION=$(shell git describe --tags 2>$(NUL))
GOOPT=-ldflags "-s -w -X main.version=$(VERSION)"
EXE=$(shell $(GO) env GOEXE)

all:
	$(GO) fmt ./...
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) && $(GO) build -C cmd/bine $(GOOPT) -o $(CURDIR)

test:
	$(GO) test -v

_dist:
	$(SET) "CGO_ENABLED=0" && $(GO) build $(GOOPT) && $(GO) build -C cmd/bine $(GOOPT) -o $(CURDIR)
	zip -9 $(NAME)-$(VERSION)-$(GOOS)-$(GOARCH).zip $(NAME)$(EXE)

dist:
	$(SET) "GOOS=linux" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=linux" && $(SET) "GOARCH=amd64" && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=386"   && $(MAKE) _dist
	$(SET) "GOOS=windows" && $(SET) "GOARCH=amd64" && $(MAKE) _dist

release:
	$(GO) run github.com/hymkor/latest-notes@master | gh release create -d --notes-file - -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

clean:
	$(DEL) *.zip $(NAME)$(EXE)

manifest:
	$(GO) run github.com/hymkor/make-scoop-manifest@master -all *-windows-*.zip > $(NAME).json

.PHONY: all test _dist dist clean manifest
