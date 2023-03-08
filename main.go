package main

import "github.com/spf13/cobra"

var (
	episodes int
	horizon  int
	saveFile string
)

func main() {
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVarP(&horizon, "horizon", "h", 50, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&saveFile, "save", "-s", "save.png", "Save the plot to the specified file")
	rootCommand.AddCommand(OneCommand())
}
