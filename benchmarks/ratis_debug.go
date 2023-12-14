package benchmarks

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/ratis"
)

var ratisDebugLogFile string

func RatisDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ratis-debug",
		Run: func(cmd *cobra.Command, args []string) {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)

			doneCh := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				select {
				case <-sigCh:
				case <-doneCh:
				}
				cancel()
			}()

			clusterConfig := &ratis.RatisClusterConfig{
				NumNodes:            3,
				BasePort:            5000,
				BaseInterceptPort:   2023,
				InterceptListenPort: 7074,
				RatisJarPath:        "/Users/srinidhin/Local/github/berkay/ratis-fuzzing/ratis-examples/target/ratis-examples-2.5.1.jar",
				WorkingDir:          "/Users/srinidhin/Local/github/berkay/ratis-fuzzing",
				GroupID:             "02511d47-d67c-49a3-9011-abb3109a44c1",
			}
			cluster := ratis.NewCluster(clusterConfig)

			network := ratis.NewInterceptNetwork(ctx, clusterConfig.InterceptListenPort)
			network.Start()

			cluster.Start()

			time.Sleep(10 * time.Second)
			cluster.Destroy()

			os.WriteFile(ratisDebugLogFile, []byte(cluster.GetLogs()), 0644)

			close(doneCh)
		},
	}

	cmd.PersistentFlags().StringVarP(&ratisDebugLogFile, "save", "s", "results_ratis.log", "Path to save the log file to")
	return cmd
}
