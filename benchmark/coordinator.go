package benchmark

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	api "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Worker struct {
	Environment  *Environment
	Component    *Component
	Address      string
	Connection   *grpc.ClientConn
	ResultStream api.BenchmarkWorker_StartWorkerClient
}

func AllocateWorkers(deployment *Deployment) []*Worker {
	workers := make([]*Worker, len(deployment.Components))
	envComponentsMap := createEnvComponentsMap(deployment.Components)

	log.Printf("We have %d components and %d env-keys", len(deployment.Components), len(envComponentsMap))

	//TODO check if all env refs exist beforehand in validation phase
	//TODO deploy workers on envs. Kubernetes API here?
	for i, e := range deployment.Environments {
		workers[i] = &Worker{
			Environment: e,
			//Component:   envComponentsMap[e.Identifier],
			//TODO set address of created worker here
			Address: "",
		}
	}
	return workers
}

func SetupConnections(workers []*Worker) {
	// Create credentials that skip root CA verification
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	// Establish connections to all workers.
	for _, w := range workers {
		conn, err := grpc.Dial(w.Address, grpc.WithTransportCredentials(creds))
		if err != nil {
			log.Printf("Couldnt connect to worker: %v, error was: %v", w, err)
		}
		w.Connection = conn
	}
}

func StartBenchmark(workers []*Worker, benchmarkConf *BenchmarkConfig) {
	//start benchmark on all workers and keep receiving their results
	//need to fork out into separate threads and write results to files/database
	dirname := benchmarkConf.ResultDirPrefix + time.Now().Format(time.RFC3339)
	err := os.Mkdir(dirname, 0700)
	if err != nil {
		log.Printf("Couldn't create output file, reason: %v", err)
	}
	finishedChannels := make([]chan bool, 0)
	for _, w := range workers {
		clientStub := api.NewBenchmarkWorkerClient(w.Connection)
		spanSequence := make([]*api.SpanModel, 1)
		spanSequence[0] = w.Component.ToSpanModel()
		clientStream, err := clientStub.StartWorker(context.Background(), &api.WorkerConfiguration{
			WorkerId:         w.Component.Identifier,
			EnvironmentId:    w.Component.EnvironmentRef,
			RuntimeSeconds:   benchmarkConf.Runtime,
			SpanSequence:     spanSequence,
			TargetThroughput: benchmarkConf.Throughput,
		})
		if err != nil {
			log.Printf("Couldn't call worker %v, error was : %v", w, err)
		}
		w.ResultStream = clientStream
		finishedChan := make(chan bool, 1)
		finishedChannels = append(finishedChannels, finishedChan)
		go WriteResults(w, dirname, finishedChan)
	}
	//wait until benchmark is over, plus six seconds (results stream poll interval + 1)
	<-time.NewTimer(time.Second * time.Duration(benchmarkConf.Runtime)).C
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
	fileHandle, err := os.Create(resultDir + "/" + worker.Component.Identifier + ".csv")
	if err != nil {
		log.Printf("Couldn't create output file, reason: %v", err)
	}
	writer := csv.NewWriter(fileHandle)
	writer.Write([]string{"SpanId", "SpanCreationBeginTime", "SpanCreationEndTime", "SpanFinishBeginTime", "SpanFinishEndTime"})
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			//Problem: ticker interval is semi-meaningless, since Recv() blocks until a result is received.
			//This means that the interval of receiving results here is strongly correlated to the interval with which results are sent to the stream.
			resultPackage, err := worker.ResultStream.Recv()
			if err != nil {
				if err == io.EOF {
					writer.Flush()
					fileHandle.Close()
					return
				}
				log.Printf("Error receiving result: %v", err)
			}
			log.Printf("Received result package. Size: %d", len(resultPackage.GetResults()))
			if resultPackage != nil {
				for _, res := range resultPackage.GetResults() {
					//TODO need different serialization here - we currently write to CSV, long-term there should be an interface for arbitrary storage
					resArray := []int64{res.SpanId, res.SpanCreationBeginTime, res.SpanCreationEndTime, res.SpanFinishBeginTime, res.SpanFinishEndTime}
					writer.Write(intToStringArray(resArray))
				}
			}
		case <-finishedChannel:
			writer.Flush()
			fileHandle.Close()
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

func createEnvComponentsMap(components []*Component) map[string][]*Component {
	result := make(map[string][]*Component)
	for _, c := range components {
		list, exists := result[c.EnvironmentRef]
		if !exists {
			list = make([]*Component, 0)
			result[c.EnvironmentRef] = list
		}
		list = append(list, c)
	}
	return result
}
