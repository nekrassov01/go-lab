package main

import (
	"context"
	"sync"

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

var state = []string{
	"pending",
	"running",
	"stopping",
	"stopped",
}

func retryOpt(opt *ec2.Options) {
	opt.RetryMaxAttempts = 3
	opt.RetryMode = aws.RetryModeStandard
}

func getNameTagValue(tags []types.Tag) string {
	for _, t := range tags {
		if *t.Key == "Name" {
			return *t.Value
		}
	}
	return ""
}

func getAwsRegion(ctx context.Context, cfg *aws.Config) ([]string, error) {
	client := ec2.NewFromConfig(*cfg)

	obj, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{}, retryOpt)
	if err != nil {
		return nil, err
	}

	var res []string
	for _, r := range obj.Regions {
		//fmt.Println(aws.ToString(r.RegionName))
		res = append(res, aws.ToString(r.RegionName))
	}

	return res, nil
}

func getAwsInstanceSync(ctx context.Context, cfg *aws.Config, filter ...types.Filter) ([]instanceInfo, error) {
	regions, err := getAwsRegion(ctx, cfg)
	if err != nil {
		return nil, err
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

	for _, region := range regions {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			return nil, err
		}

		client := ec2.NewFromConfig(cfg)

		obj, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: f}, retryOpt)
		if err != nil {
			return nil, err
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

	return res, nil
}

func getAwsInstanceAsync(ctx context.Context, cfg *aws.Config, filter ...types.Filter) ([]instanceInfo, error) {
	regions, err := getAwsRegion(ctx, cfg)
	if err != nil {
		return nil, err
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

	ich := make(chan instanceInfo)
	ech := make(chan error)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, region := range regions {
		wg.Add(1)

		go func(region string) {
			defer wg.Done()

			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
			if err != nil {
				ech <- err
				return
			}

			client := ec2.NewFromConfig(cfg)

			obj, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: f}, retryOpt)
			if err != nil {
				ech <- err
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
					ich <- out
				}
			}
		}(region)
	}

	go func() {
		wg.Wait()
		close(ich)
		close(ech)
	}()

	var res []instanceInfo

	for {
		select {
		case i, ok := <-ich:
			if ok {
				res = append(res, i)
			} else {
				ich = nil
			}
		case err, ok := <-ech:
			if ok {
				return nil, err
			} else {
				ech = nil
			}
		}
		if ich == nil && ech == nil {
			break
		}
	}

	return res, nil
}

func getAwsInstanceAsync2(ctx context.Context, cfg *aws.Config, filter ...types.Filter) ([]instanceInfo, error) {
	regions, err := getAwsRegion(ctx, cfg)
	if err != nil {
		return nil, err
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

	var mu sync.Mutex
	var res []instanceInfo

	eg, ctx := errgroup.WithContext(ctx)

	for _, region := range regions {
		region := region

		eg.Go(func() error {
			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
			if err != nil {
				return err
			}

			client := ec2.NewFromConfig(cfg)

			obj, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: f}, retryOpt)
			if err != nil {
				return err
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

					mu.Lock()
					res = append(res, out)
					mu.Unlock()
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return res, nil
}
