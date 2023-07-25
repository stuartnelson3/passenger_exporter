FROM golang:1.18-bullseye as builder

ARG VERSION=0.0.1
ARG COMMIT=HEAD
ARG DATE=now
ARG BUILT_BY=docker

WORKDIR $GOPATH/src/github.com/toptal/passenger_exporter
COPY . .
RUN go mod download
RUN go get github.com/toptal/passenger_exporter
RUN go  build -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -X main.builtBy=${BUILT_BY}"

FROM phusion/passenger-customizable:2.5.1 as final

RUN mkdir /app
WORKDIR /app

COPY --from=builder /go/src/github.com/toptal/passenger_exporter/passenger_exporter passenger_exporter

ENTRYPOINT [ "/app/passenger_exporter" ]
