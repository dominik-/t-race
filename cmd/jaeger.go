package cmd

import (
	"github.com/spf13/cobra"
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

}
