package benchmark

import (
	"context"
	"crypto/x509"
	"encoding/csv"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	api "gitlab.tubit.tu-berlin.de/dominik-ernst/trace-writer-api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Worker struct {
	Component    *Component
	Address      string
	Connection   *grpc.ClientConn
	ResultStream api.BenchmarkWorker_StartWorkerClient
}

func AllocateWorkers(rootComponent *Component, adresses []string) []*Worker {
	//Use the parsed components and combine with ip addresses to allocate "environments"
	//traverse component tree
	componentsInOrder := make([]*Component, 0)
	workers := make([]*Worker, len(adresses))
	componentsInOrder = AddComponentsToSlice(componentsInOrder, rootComponent)
	envComponentsMap := createEnvComponentsMap(componentsInOrder)

	log.Printf("We have %d components, %d workers and %d env-keys", len(componentsInOrder), len(workers), len(envComponentsMap))

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
	// Read cert file
	FrontendCert, _ := ioutil.ReadFile("./certs/frontend.cert")

	// Create CertPool
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(FrontendCert)

	// Create credentials
	credsClient := credentials.NewClientTLSFromCert(roots, "")
	// Establish connections to all workers. TLS-encrypted with static certificate
	for _, w := range workers {
		conn, err := grpc.Dial(w.Address, grpc.WithTransportCredentials(credsClient))
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
			EnvironmentId:    w.Component.DeploymentKey,
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
	toleranceDuration := 5 * time.Second
	log.Printf("Runtime finished. Waiting %v for final benchmark results...", toleranceDuration)
	<-time.NewTimer(5 * time.Second).C
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
			//Problem: ticker is pretty meaningless, since Recv() blocks until a result is received.
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
		list, exists := result[c.DeploymentKey]
		if !exists {
			list = make([]*Component, 0)
			result[c.DeploymentKey] = list
		}
		list = append(list, c)
	}
	return result
}
