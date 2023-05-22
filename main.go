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

func main() {
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 50, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&saveFile, "save", "s", "results", "Save the result data in the specified folder")
	rootCommand.AddCommand(RaftCommand())
	rootCommand.AddCommand(PaxosCommand())
	rootCommand.AddCommand(PaxosPartCommand())

	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
	}
}
