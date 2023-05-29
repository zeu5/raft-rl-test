package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	episodes int
	horizon  int
	saveFile string
)

// main entry point to all the experiments
func main() {
	// rootCommand defines a command line argument parser (some arguments and a subcommand to run)
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 50, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&saveFile, "save", "s", "results", "Save the result data in the specified folder")
	// adding the subcommands here
	rootCommand.AddCommand(RaftCommand())
	rootCommand.AddCommand(PaxosCommand())
	rootCommand.AddCommand(PaxosPartCommand())
	rootCommand.AddCommand(PaxosRewardCommand())

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
	}
}
