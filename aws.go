package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"golang.org/x/sync/errgroup"
)

type instanceInfo struct {
	Name             string
	InstanceId       string
	PrivateIpAddress string
	PublicIpAddress  string
	AvailabilityZone string
	State            types.InstanceStateName
}

var waitMillisecond = 1000

var state = []string{
	"pending",
	"running",
	"stopping",
	"stopped",
}

func getAwsRegion(cfg *aws.Config) ([]string, error) {
	client := ec2.NewFromConfig(*cfg)

	retryOpt := func(opt *ec2.Options) {
		opt.RetryMaxAttempts = 3
		opt.RetryMode = aws.RetryModeStandard
	}

	obj, err := client.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{}, retryOpt)
	if err != nil {
		return nil, fmt.Errorf("`client.DescribeRegions()` failed: %w", err)
	}

	var res []string
	for _, r := range obj.Regions {
		res = append(res, aws.ToString(r.RegionName))
	}

	return res, nil
}

func getNameTagValue(tags []types.Tag) string {
	for _, t := range tags {
		if *t.Key == "Name" {
			return *t.Value
		}
	}
	return ""
}

func getAwsInstanceSync(region string, filter ...types.Filter) ([]instanceInfo, error) {
	baseTime := time.Now()

	retryOpt := func(opt *ec2.Options) {
		opt.RetryMaxAttempts = 3
		opt.RetryMode = aws.RetryModeStandard
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("`config.LoadDefaultConfig()` failed: %w", err)
	}

	regions, err := getAwsRegion(&cfg)
	if err != nil {
		return nil, fmt.Errorf("`getAwsRegion()` failed: %w", err)
	}

	f := []types.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: state,
		},
	}

	if filter != nil {
		f = append(f, filter...)
	}

	var res []instanceInfo

	for _, rg := range regions {
		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(rg))
		if err != nil {
			return nil, fmt.Errorf("`config.LoadDefaultConfig()` failed: %w", err)
		}

		client := ec2.NewFromConfig(cfg)

		obj, err := client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{Filters: f}, retryOpt)
		if err != nil {
			return nil, fmt.Errorf("`client.DescribeInstances()` failed: %w", err)
		}

		for _, r := range obj.Reservations {
			for _, i := range r.Instances {
				out := instanceInfo{
					getNameTagValue(i.Tags),
					aws.ToString(i.InstanceId),
					aws.ToString(i.PrivateIpAddress),
					aws.ToString(i.PublicIpAddress),
					aws.ToString(i.Placement.AvailabilityZone),
					i.State.Name,
				}
				res = append(res, out)
			}
		}
	}

	printInfo(baseTime)

	return res, nil
}

func getAwsInstanceAsync(region string, filter ...types.Filter) ([]instanceInfo, error) {
	baseTime := time.Now()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	retryOpt := func(opt *ec2.Options) {
		opt.RetryMaxAttempts = 3
		opt.RetryMode = aws.RetryModeStandard
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("`config.LoadDefaultConfig()` failed: %w", err)
	}

	regions, err := getAwsRegion(&cfg)
	if err != nil {
		return nil, fmt.Errorf("`getAwsRegion()` failed: %w", err)
	}

	f := []types.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: state,
		},
	}

	if filter != nil {
		f = append(f, filter...)
	}

	instanceCh := make(chan instanceInfo)
	errorCh := make(chan error)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, rg := range regions {
		wg.Add(1)

		go func(rg string) {
			defer wg.Done()

			time.Sleep(time.Duration(r.Intn(waitMillisecond)) * time.Millisecond)

			cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(rg))
			if err != nil {
				errorCh <- fmt.Errorf("`config.LoadDefaultConfig()` failed: %w", err)
				return
			}

			client := ec2.NewFromConfig(cfg)

			obj, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: f}, retryOpt)
			if err != nil {
				errorCh <- fmt.Errorf("`client.DescribeInstances()` failed: %w", err)
				return
			}

			for _, r := range obj.Reservations {
				for _, i := range r.Instances {
					out := instanceInfo{
						getNameTagValue(i.Tags),
						aws.ToString(i.InstanceId),
						aws.ToString(i.PrivateIpAddress),
						aws.ToString(i.PublicIpAddress),
						aws.ToString(i.Placement.AvailabilityZone),
						i.State.Name,
					}
					instanceCh <- out
				}
			}

		}(rg)
	}

	go func() {
		wg.Wait()
		close(instanceCh)
		close(errorCh)
	}()

	var res []instanceInfo

	for {
		select {
		case i, ok := <-instanceCh:
			if ok {
				res = append(res, i)
			} else {
				instanceCh = nil
			}
		case err, ok := <-errorCh:
			if ok {
				return nil, err
			} else {
				errorCh = nil
			}
		}
		if instanceCh == nil && errorCh == nil {
			break
		}
	}

	printInfo(baseTime)

	return res, nil
}

func getAwsInstanceAsync2(region string, filter ...types.Filter) ([]instanceInfo, error) {
	baseTime := time.Now()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	retryOpt := func(opt *ec2.Options) {
		opt.RetryMaxAttempts = 3
		opt.RetryMode = aws.RetryModeStandard
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("`config.LoadDefaultConfig()` failed: %w", err)
	}

	regions, err := getAwsRegion(&cfg)
	if err != nil {
		return nil, fmt.Errorf("`getAwsRegion()` failed: %w", err)
	}

	f := []types.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: state,
		},
	}

	if filter != nil {
		f = append(f, filter...)
	}

	ch := make(chan instanceInfo)
	eg, ctx := errgroup.WithContext(context.Background())

	for _, rg := range regions {
		rg := rg

		eg.Go(func() error {
			time.Sleep(time.Duration(r.Intn(waitMillisecond)) * time.Millisecond)

			cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(rg))
			if err != nil {
				return fmt.Errorf("`config.LoadDefaultConfig()` failed: %w", err)
			}

			client := ec2.NewFromConfig(cfg)

			obj, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: f}, retryOpt)
			if err != nil {
				return fmt.Errorf("`client.DescribeInstances()` failed: %w", err)
			}

			for _, r := range obj.Reservations {
				for _, i := range r.Instances {
					out := instanceInfo{
						getNameTagValue(i.Tags),
						aws.ToString(i.InstanceId),
						aws.ToString(i.PrivateIpAddress),
						aws.ToString(i.PublicIpAddress),
						aws.ToString(i.Placement.AvailabilityZone),
						i.State.Name,
					}
					select {
					case ch <- out:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}

			return nil
		})
	}

	go func() {
		eg.Wait()
		close(ch)
	}()

	var res []instanceInfo

	for c := range ch {
		res = append(res, c)
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	printInfo(baseTime)

	return res, nil
}
