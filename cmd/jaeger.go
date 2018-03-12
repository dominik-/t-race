package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gitlab.tu-berlin.de/dominik-ernst/tracer-benchmarks/jaeger"
)

var (
	runtime int64
	delay   int64
)

func init() {
	rootCmd.AddCommand(jaegerCmd)
	jaegerCmd.Flags().Int64VarP(&runtime, "runtime", "r", 1, "The runtime of each worker in minutes.")
	jaegerCmd.Flags().Int64VarP(&delay, "delay", "d", 10000, "The delay that is used between opening and closing a span. In microseconds.")
}

var jaegerCmd = &cobra.Command{
	Use:   "jaeger",
	Short: "Benchmark jaeger as SUT",
	Long:  `Runs the tracing benchmark against jaeger as SUT, using jaeger-specific configuration (agents, etc.)`,
	Run:   runJaegerBenchmark,
}

func runJaegerBenchmark(cmd *cobra.Command, args []string) {
	workers, err := cmd.InheritedFlags().GetInt("workers")
	interval, err := cmd.InheritedFlags().GetInt64("interval")
	if err != nil {
		log.Fatalf("Couldnt find one of the mandatory param values. Should never happen! Error was: %v", err)
	}

	writers := make([]*jaeger.RuntimeTraceWriter, 0)

	for i := 0; i < workers; i++ {
		writer := jaeger.NewRuntimeTraceWriter(runtime, delay, interval, fmt.Sprintf("Jaeger-%d", i))
		go writer.WriteSpansUntilFinished()
		writers = append(writers, writer)
	}
	fmt.Printf("Started benchmark for jaeger with params: runtime: %d m, delay: %d µs, threads: %d, interval: %d ms\n", runtime, delay, workers, interval)

	//create a timer with runtime + 1 minute
	timer := time.NewTimer((time.Duration(runtime) + 1) * time.Minute)

	//hook to SIGINT/SIGTERM
	sigTermRecv := make(chan os.Signal, 1)
	signal.Notify(sigTermRecv, syscall.SIGINT, syscall.SIGTERM)

	//we end if either SIGINT/SIGTERM is received or the time finishes
	for {
		select {
		case <-timer.C:
			os.Exit(0)
		case <-sigTermRecv:
			for _, w := range writers {
				w.TerminateSignal <- true
			}
			//after signalling to shutdown to all writers, wait half a second, then exit.
			<-time.NewTimer(500 * time.Millisecond).C
			log.Fatal("User arborted.")
		}
	}

}
