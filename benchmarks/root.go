package benchmarks

import "github.com/spf13/cobra"

var (
	episodes int
	horizon  int
	saveFile string
	runs     int
)

func GetRootCommand() *cobra.Command {
	rootCommand := &cobra.Command{}
	rootCommand.PersistentFlags().IntVarP(&episodes, "episodes", "e", 10000, "Number of episodes to run")
	rootCommand.PersistentFlags().IntVar(&horizon, "horizon", 100, "Horizon of each episode")
	rootCommand.PersistentFlags().StringVarP(&saveFile, "save", "s", "results", "Save the result data in the specified folder")
	rootCommand.PersistentFlags().IntVar(&runs, "runs", 1, "Number of experiment runs")
	// adding the subcommands here
	rootCommand.AddCommand(RedisTestCommand())
	rootCommand.AddCommand(RedisRaftCommand())
	rootCommand.AddCommand(RedisRaftRMCommand())
	rootCommand.AddCommand(RaftCommand())
	rootCommand.AddCommand(RaftPartCommand())
	rootCommand.AddCommand(EtcdRaftBugsCommand())
	rootCommand.AddCommand(PaxosPartCommand())
	rootCommand.AddCommand(PaxosRewardCommand())
	rootCommand.AddCommand(PaxosRewardMachineCommand())
	rootCommand.AddCommand(GridRewardCommand())
	rootCommand.AddCommand(GridRewardMachineCommand())
	rootCommand.AddCommand(RSLExplorationCommand())
	rootCommand.AddCommand(RSLRewardMachineCommand())
	rootCommand.AddCommand(RatisExplorationCommand())
	rootCommand.AddCommand(CometCommand())
	rootCommand.AddCommand(CometRMCommand())
	rootCommand.AddCommand(RatisDebugCommand())
	return rootCommand
}
