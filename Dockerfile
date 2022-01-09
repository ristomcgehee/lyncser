FROM golang:1.17 AS builder

COPY . /build

WORKDIR /build
RUN make build


FROM ubuntu:20.04

RUN apt-get update && apt-get install -y \
    # Just so we can get trusted certificates
    curl

COPY --from=builder /build/lyncser /usr/local/bin/lyncser
