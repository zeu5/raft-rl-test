package main

import (
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/fuzzing"
)

func FuzzCommand() *cobra.Command {
	return &cobra.Command{
		Use: "fuzz",
		RunE: func(cmd *cobra.Command, args []string) error {
			fuzzer := fuzzing.NewFuzzer(&fuzzing.FuzzerConfig{
				Iterations: episodes,
				Steps:      horizon,
				TLCAddr:    "127.0.0.1:2023",
				Mutator:    &fuzzing.EmptyMutator{},
				RaftEnvironmentConfig: fuzzing.RaftEnvironmentConfig{
					Replicas:      3,
					ElectionTick:  10,
					HeartbeatTick: 2,
				},
			})
			return fuzzer.Run()
		},
	}
}
