package benchmark

import (
	"bufio"
	"context"
	"log"
	"os"
	"time"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/proto"
	"google.golang.org/grpc"
)

type Worker struct {
	Component    *Component
	Address      string
	Connection   *grpc.ClientConn
	ResultStream proto.BenchmarkWorker_StartWorkerClient
}

func AllocateWorkers(rootComponent *Component, adresses []string) []*Worker {
	//Use the parsed components and combine with ip addresses to allocate "environments"
	//traverse component tree
	componentsInOrder := make([]*Component, 0)
	workers := make([]*Worker, len(adresses))
	rootComponent.AddComponentsToSlice(componentsInOrder)
	if len(componentsInOrder) != len(adresses) {
		log.Fatal("Not enough workers for components.")
	}
	for i, c := range componentsInOrder {
		workers[i] = &Worker{
			Component: c,
			Address:   adresses[i],
		}
	}
	return workers
}

func SetupConnections(workers []*Worker) {
	for _, w := range workers {
		conn, err := grpc.Dial(w.Address)
		if err != nil {
			log.Printf("Couldnt connect to worker: %v, error was: %v", w, err)
		}
		w.Connection = conn
	}
}

func StartBenchmark(workers []*Worker, benchmarkConf *BenchmarkConfig) {
	//start benchmark on all workers and keep receiving their results
	//need to fork out into separate threads and write results to files/database
	for _, w := range workers {
		clientStub := proto.NewBenchmarkWorkerClient(w.Connection)
		spanSequence := make([]*proto.SpanModel, 1)
		spanSequence[0] = w.Component.ToSpanModel()
		clientStream, err := clientStub.StartWorker(context.Background(), &proto.WorkerConfiguration{
			EnvironmentId:  w.Component.DeploymentKey,
			RuntimeSeconds: benchmarkConf.Runtime,
			SpanSequence:   spanSequence,
		})
		if err != nil {
			log.Printf("Couldn't call worker %v, error was : %v", w, err)
		}
		w.ResultStream = clientStream
		go WriteResults(w, benchmarkConf)
	}

}

func WriteResults(worker *Worker, benchmark *BenchmarkConfig) {
	fileHandle, err := os.Create(benchmark.ResultDirPrefix + "/" + worker.Component.Identifier)
	if err != nil {
		log.Printf("Couldn't create output file, reason: %v", err)
	}
	writer := bufio.NewWriter(fileHandle)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			resultPackage, err := worker.ResultStream.Recv()
			if err != nil {
				log.Printf("Error receiving result: %v", err)
			}
			if resultPackage != nil {
				for _, res := range resultPackage.GetResults() {
					//TODO need different serialization here - write to CSV? long-term there should be a interface for arbitrary storage
					writer.WriteString(res.String())
				}
			}
		}
	}

}
