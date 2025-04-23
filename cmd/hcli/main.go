package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/alam0rt/tailscale-sidecar-injector/pkg/headscale"
)

func main() {
	ctx := context.Background()
	slog.SetLogLoggerLevel(slog.LevelDebug)
	client, err := headscale.New(ctx, "", "")
	if err != nil {
		panic(err)
	}

	c, err := client.PreAuthKeys().Create(ctx, "sammm", false, true, time.Now().Add(time.Second*30), []string{"tag:pod"})
	if err != nil {
		panic(err)
	}
	client.Logger.Info("response", "resp", c)
	client.Logger.Info("key found", "key", c.PreAuthKey.Key, "user", c.PreAuthKey.User)

	if err := client.PreAuthKeys().Expire(ctx, "sammm", c.PreAuthKey.Key); err != nil {
		panic(err)
	}

	b, err := client.PreAuthKeys().List(ctx, "sammm")
	if err != nil {
		panic(err)
	}

	for _, key := range b.PreAuthKeys {
		client.Logger.Info("key found", "key", key.Key, "user", "sammm")
	}
}
