package benchmark

import (
	"context"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/dominik-/t-race/api"
	"github.com/dominik-/t-race/executionmodel"
	"github.com/gocarina/gocsv"

	"google.golang.org/grpc"
)

var resultDirFormat = "2006-01-02T150405"

type Benchmark struct {
	Name    string
	Workers []*Worker
	Config  *executionmodel.BenchmarkConfig
}

type Worker struct {
	Config       *api.WorkerConfiguration
	Address      string
	Connection   *grpc.ClientConn
	ResultStream api.BenchmarkWorker_StartWorkerClient
}

func Setup(architecture *executionmodel.Architecture, serviceMap, workerMap, sinkMap map[string]string, config *executionmodel.BenchmarkConfig) *Benchmark {
	workers := make([]*Worker, 0)

	configs := executionmodel.MapArchitectureToWorkers(*architecture, *config, sinkMap, serviceMap)

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
		Name:    architecture.Name,
		Workers: workers,
		Config:  config,
	}
}

func (benchmark *Benchmark) StartBenchmark() {
	//start benchmark on all workers and keep receiving their results
	//need to fork out into separate threads and write results to files/database
	rootDir := "results"
	fInfo, err := os.Stat(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(rootDir, 0700)
		}
		if err != nil {
			log.Fatalf("Could't find or create output directory: %v", err)
		}
	} else {
		if !fInfo.IsDir() {
			log.Fatalf("A file called %s is conflicting with creating the output root directory.", rootDir)
		}
	}
	dirname := rootDir + "/" + benchmark.Config.ResultDirPrefix + time.Now().Format(resultDirFormat)
	err = os.Mkdir(dirname, 0700)
	if err != nil {
		log.Printf("Couldn't create output directory, reason: %v", err)
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
	<-time.NewTimer(time.Second * time.Duration(benchmark.Config.Runtime)).C
	//we wait additional time to make sure we received all events.
	toleranceDuration := 30 * time.Second
	log.Printf("Runtime finished. Waiting %v for final benchmark results...", toleranceDuration)
	<-time.NewTimer(toleranceDuration).C
	for _, channel := range finishedChannels {
		channel <- true
	}
	<-time.NewTimer(1 * time.Second).C
	log.Println("Finishing benchmark.")
	os.Exit(0)
}

func CSVWriterToFile(file *os.File) *gocsv.SafeCSVWriter {
	csvWriter := csv.NewWriter(file)
	return gocsv.NewSafeCSVWriter(csvWriter)
}

func WriteResults(worker *Worker, resultDir string, finishedChannel <-chan bool) {
	fileHandle, err := os.Create(resultDir + "/" + worker.Config.WorkerId + ".csv")
	if err != nil {
		log.Printf("Couldn't create output file, reason: %v", err)
	}
	defer fileHandle.Close()
	writer := CSVWriterToFile(fileHandle)
	firstWrite := true
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
				log.Printf("Error receiving result from worker/service %s: %v", worker.Config.ServiceName, err)
			} else {
				log.Printf("Received result package from worker/service %s. Size: %d", worker.Config.ServiceName, len(resultPackage.GetResults()))
			}
			if resultPackage != nil {
				if firstWrite {
					gocsv.MarshalCSV(resultsToRecords(resultPackage, worker.Config), writer)
					firstWrite = false
				} else {
					gocsv.MarshalCSVWithoutHeaders(resultsToRecords(resultPackage, worker.Config), writer)
				}
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
