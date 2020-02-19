# T-Race

Tool to benchmark tracing systems by emulating (possibly complex) multi-service deployments. Implemented as a single executable in Golang.

Inputs for the benchmark are a model of a deployed application (_Service Model_, e.g., `test-2.yaml`), a set of _Workers_ and a set of _Sinks_ (e.g., `deployment_localhost_2.json`). Sinks are the endpoints of the distributed tracing backend.

T-Race is under development and considered a prototype. Use at your own discretion.

## Supported SUTs
As of now, T-Race implements an adapter for Jaeger https://www.jaegertracing.io/docs/1.11/. Traces are generated in the OpenTracing https://opentracing.io/ format. Workers communicate using gRPC, which is using HTTP2 for transport, i.e., propagated trace context data is marshalled to HTTP custom headers, check https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md for some details.

## Requirements
Currently I don't provide pre-compiled binaries, as such you need to build t-race yourself.

### Building
Prerequisites:
* Golang 1.12+ with dep enabled

Simply run `go build .` to download dependencies and create binaries for your local OS.

### Running a benchmark
The SUT setup is not managed by t-race, consequently you need to have a tracing system (i.e. Jaeger) up and running. Next, start t-race workers on each environment where you want to have a service deployed (see [Usage]). Create a deployment file or update `deployment_localhost_2.json`
with the workers' connection strings.

If you want to try out locally first, you can use the provided docker-compose file `docker-compose-jaeger-backend.yml` to setup a local
deployment of Jaeger, including Prometheus.

## Usage

TODO

## Roadmap

In the future, we plan to integrate different providers' interfaces, automating the deployment of workers. There is a (ver basic) `provider` interfaces conceptualizing this right now, but only the "static" provider is implemented, which means you need to supply worker IPs etc. through a JSON config file.

## Links
CQL to CSV export:
https://docs.datastax.com/en/archived/cql/3.3/cql/cql_reference/cqlshCopy.html