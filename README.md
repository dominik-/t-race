#Tracer-Benchmarks

Tool to benchmark tracing systems. Started with jaeger, based on https://github.com/jaegertracing/jaeger-performance

##Requirements

* Docker
* Golang 1.9+

##Usage

1. Clone this repo and cd into folder
1. Start up jaeger using the Docker Compose file from https://github.com/jaegertracing/jaeger/tree/master/docker-compose:

    `sudo docker-compose -f jaeger-docker-compose.yml up -d`
1. Build the tool with go. Project uses Dep: https://github.com/golang/dep

    `dep ensure`

    `go build .`
1. Run the benchmark tool to see params (for now only jaeger):

    `./tracer-benchmarks jaeger -h`
1. Run a benchmark. Currently no results are persisted. You can check out the Jaeger GUI and see traces at http://localhost:16686/