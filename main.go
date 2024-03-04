package main

import (
	"fmt"

	"github.com/zeu5/raft-rl-test/benchmarks"
	"github.com/zeu5/raft-rl-test/explorer"
)

// main entry point to all the experiments
func main() {
	// rootCommand defines a command line argument parser (some arguments and a subcommand to run)
	rootCommand := benchmarks.GetRootCommand()

	rootCommand.AddCommand(explorer.ExploreCommand())

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
	}
}
