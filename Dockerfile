FROM ubuntu:20.04

RUN apt-get update && apt-get install -y \
    # Just so we can get trusted certificates
    curl

WORKDIR /lyncser
