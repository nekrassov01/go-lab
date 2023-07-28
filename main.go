package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	name   = "go-lab"
	region = "ap-northeast-1"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	benchmark(func() {
		out, err := getAwsInstanceAsync(ctx, &cfg)
		if err != nil {
			printError(err)
			os.Exit(1)
		}
		printJson(out)
	})

	benchmark(func() {
		out, err := getAwsInstanceAsync2(ctx, &cfg)
		if err != nil {
			printError(err)
			os.Exit(1)
		}
		printJson(out)
	})

	benchmark(func() {
		out, err := getAwsInstanceSync(ctx, &cfg)
		if err != nil {
			printError(err)
			os.Exit(1)
		}
		printJson(out)
	})
}
