package benchmarks

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/zeu5/raft-rl-test/redisraft"
)

func RedisTestCommand() *cobra.Command {
	return &cobra.Command{
		Use: "redis-cli",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := redis.NewClient(&redis.Options{
				Addr: "127.0.0.1:6379",
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			info, err := cli.Info(ctx, "raft").Result()
			if err == nil {
				redisraft.ParseInfo(info)
			}
			fmt.Printf("%s\n", info)

			return err
		},
	}
}
