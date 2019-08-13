package benchmark

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gocarina/gocsv"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
	"google.golang.org/grpc"
)

var resultDirFormat = "2006-01-02T150405"

type Benchmark struct {
	Name    string
	Workers []*Worker
	Config  *BenchmarkConfig
}

type Worker struct {
	Config       *api.WorkerConfiguration
	Address      string
	Connection   *grpc.ClientConn
	ResultStream api.BenchmarkWorker_StartWorkerClient
}

func Setup(deployment *Model, serviceMap, workerMap, sinkMap map[string]string, config *BenchmarkConfig) *Benchmark {
	workers := make([]*Worker, 0)

	configs := MapDeploymentToWorkerConfigs(*deployment, *config, sinkMap, serviceMap)

	for id, config := range configs {
		workers = append(workers, &Worker{
			Address: workerMap[id],
			Config:  config,
		})
	}

	// Create credentials that skip root CA verification
	/* 	creds := credentials.NewTLS(&tls.Config{
	   		InsecureSkipVerify: true,
	   	})
	   	option := grpc.WithTransportCredentials(creds) */
	option := grpc.WithInsecure()
	// Establish connections to all workers.
	for _, w := range workers {
		conn, err := grpc.Dial(w.Address, option)
		if err != nil {
			log.Printf("Couldnt connect to worker: %v, error was: %v", w, err)
		}
		w.Connection = conn
	}
	return &Benchmark{
		Name:    deployment.Name,
		Workers: workers,
		Config:  config,
	}
}

func (benchmark *Benchmark) StartBenchmark() {
	//start benchmark on all workers and keep receiving their results
	//need to fork out into separate threads and write results to files/database
	dirname := benchmark.Config.ResultDirPrefix + time.Now().Format(resultDirFormat)
	err := os.Mkdir(dirname, 0700)
	if err != nil {
		log.Printf("Couldn't create output file, reason: %v", err)
	}
	finishedChannels := make([]chan bool, 0)
	for _, w := range benchmark.Workers {
		clientStub := api.NewBenchmarkWorkerClient(w.Connection)
		clientStream, err := clientStub.StartWorker(context.Background(), w.Config)
		if err != nil {
			log.Fatalf("Couldn't call worker %v, error was : %v", w, err)
		}
		w.ResultStream = clientStream
		finishedChan := make(chan bool, 1)
		finishedChannels = append(finishedChannels, finishedChan)
		go WriteResults(w, dirname, finishedChan)
	}
	//wait until benchmark is over, plus six seconds (results stream poll interval + 1)
	<-time.NewTimer(time.Second * time.Duration(benchmark.Config.Runtime)).C
	toleranceDuration := 10 * time.Second
	log.Printf("Runtime finished. Waiting %v for final benchmark results...", toleranceDuration)
	<-time.NewTimer(toleranceDuration).C
	for _, channel := range finishedChannels {
		channel <- true
	}
	<-time.NewTimer(1 * time.Second).C
	log.Println("Finishing benchmark.")
	os.Exit(0)
}

func WriteResults(worker *Worker, resultDir string, finishedChannel <-chan bool) {
	fileHandle, err := os.Create(resultDir + "/" + worker.Config.WorkerId + ".csv")
	if err != nil {
		log.Printf("Couldn't create output file, reason: %v", err)
	}
	defer fileHandle.Close()
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			//Problem: ticker interval is semi-meaningless, since Recv() blocks until a result is received.
			//This means that the interval of receiving results here is strongly correlated to the interval with which results are sent to the stream.
			resultPackage, err := worker.ResultStream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("Error receiving result from worker/service %s: %v", worker.Config.OperationName, err)
			} else {
				log.Printf("Received result package from worker/service %s. Size: %d", worker.Config.OperationName, len(resultPackage.GetResults()))
			}
			if resultPackage != nil {
				gocsv.MarshalFile(resultPackage.GetResults(), fileHandle)
			}
		case <-finishedChannel:
			return
		}
	}
}

func intToStringArray(array []int64) []string {
	res := make([]string, len(array))
	for idx, val := range array {
		res[idx] = strconv.FormatInt(val, 10)
	}
	return res
}

func createEnvServicesMap(components []*Service) map[string][]*Service {
	result := make(map[string][]*Service)
	for _, c := range components {
		list, exists := result[c.EnvironmentRef]
		if !exists {
			list = make([]*Service, 0)
			result[c.EnvironmentRef] = list
		}
		list = append(list, c)
	}
	return result
}
