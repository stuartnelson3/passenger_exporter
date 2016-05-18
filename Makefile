VERSION   := $(shell cat VERSION)
BIN       := passenger_exporter_nginx
CONTAINER := passenger_exporter_nginx
GOOS      ?= linux
GOARCH    ?= amd64

GOFLAGS   := -ldflags "-X main.Version=$(VERSION)" -a -installsuffix cgo
TAR       := $(BIN)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz
DST       ?= http://ent.int.s-cloud.net/iss/$(BIN)

PREFIX                  ?= $(shell pwd)

default: $(BIN)

$(BIN):
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) promu build --prefix $(PREFIX)

release: $(TAR)
	curl -XPOST --data-binary @$< $(DST)/$<

build-docker: $(BIN)
	docker build -t $(CONTAINER) .

$(TAR): $(BIN)
	tar czf $@ $<

