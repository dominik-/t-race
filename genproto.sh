#!/bin/bash -e
#Small wrapper script to generate GRPC stubs for golang. Needs protoc and protoc-gen-go binaries on PATH.

targetdirname=api
input=./api/tracewriter.proto

protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    $input