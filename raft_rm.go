package main

import "github.com/spf13/cobra"

func RaftRM(episodes, horizon int, savePath string) {

}

func RaftRMCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "raft-rm",
		Run: func(cmd *cobra.Command, args []string) {
			RaftRM(episodes, horizon, saveFile)
		},
	}
	cmd.PersistentFlags().IntVarP(&requests, "requests", "r", 1, "Number of requests to run with")
	return cmd
}
