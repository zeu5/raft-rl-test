# syntax=docker/dockerfile:1

FROM golang:1.21

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

RUN apt-get update && apt-get --assume-yes install cmake
RUN cd .. && mkdir deps && cd deps && git clone https://github.com/json-c/json-c.git && mkdir json-c-build && cd json-c-build && cmake ../json-c && cd ..
RUN apt-get install bash

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY . ./
RUN mv redisraft-source /deps/redisraft

# Build
# RUN CGO_ENABLED=0 GOOS=linux 
# go build -o /docker-gs-ping

# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can (optionally) document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
# EXPOSE 8080

# Run
# CMD [ "/docker-gs-ping" ]
