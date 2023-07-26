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

FROM ruby:3.2-alpine as final

RUN apk --no-cache add curl make libc6-compat
RUN gem install passenger

RUN mkdir /app
WORKDIR /app

COPY --from=builder /go/src/github.com/toptal/passenger_exporter/passenger_exporter passenger_exporter

ENTRYPOINT [ "/app/passenger_exporter" ]
