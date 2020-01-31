#!/bin/bash -e
#Small wrapper script to generate GRPC stubs for golang. Needs protoc and protoc-gen-go binaries on PATH.

targetdirname=api
input=./api/tracewriter.proto

protoc --go_out=plugins=grpc:$targetdirname -I ./$targetdirname $input