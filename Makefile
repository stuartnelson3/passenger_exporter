VERSION   := 0.0.1
BIN       := passenger_exporter_nginx
CONTAINER := passenger_exporter_nginx
GOFLAGS   := -ldflags "-X main.Version=$(VERSION)" -a -installsuffix cgo

default: $(BIN)

build-docker: $(BIN)
	docker build -t $(CONTAINER) .

$(BIN):
	CGO_ENABLED=0 GOOS=linux go build $(GOFLAGS) -o $(BIN) .
