package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	connect "github.com/aws/aws-sdk-go/service/ec2instanceconnect"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2" // imports as package "cli"

	"github.com/nodefortytwo/amz-ssh/pkg/sshutils"
	"github.com/nodefortytwo/amz-ssh/pkg/update"
)

var version = "0.0.0"

func main() {
	rand.Seed(time.Now().Unix())
	setupSignalHandlers()
	app := &cli.App{
		Name:    "amz-ssh",
		Usage:   "connect to an ec2 instance via ec2 connect",
		Version: version,
		Action:  run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "region",
				Aliases: []string{"r"},
				EnvVars: []string{"AWS_REGION"},
				Value:   "eu-west-1",
			},
			&cli.StringFlag{
				Name:  "tag",
				Value: "role:bastion",
			},
			&cli.StringFlag{
				Name:    "instance-id",
				Aliases: []string{"i"},
				Usage:   "instance id to ssh to or tunnel through",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "user",
				Aliases: []string{"u"},
				Usage:   "OS user of bastion",
				Value:   "ec2-user",
			},
			&cli.StringFlag{
				Name:    "tunnel",
				Aliases: []string{"t"},
				Usage:   "Host to tunnel to",
			},
			&cli.StringSliceFlag{
				Name:    "destination",
				Aliases: []string{"d"},
				Usage:   "destination to ssh to via the bastion. This flag can be provided multiple times to allow for multple hops",
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   22,
			},
			&cli.IntFlag{
				Name:    "local-port",
				Aliases: []string{"lp"},
				Usage:   "local port to map to, defaults to tunnel port",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Print debug information",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "update",
				Usage:  "Update the cli",
				Action: update.Handler,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)

	}
}
func setupSignalHandlers() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nGoodbye!")
		os.Exit(0)
	}()
}

func run(c *cli.Context) error {
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	var tagName string
	var tagValue string

	if parts := strings.Split(c.String("tag"), ":"); len(parts) == 2 {
		tagName = parts[0]
		tagValue = parts[1]
	} else {
		return fmt.Errorf("%s is not a valid tag definition, use key:value", c.String("tag"))
	}

	ec2Client := ec2Client(c.String("region"))
	connectClient := connectClient(c.String("region"))

	instanceID := c.String("instance-id")
	if instanceID == "" {
		var err error
		instanceID, err = resolveBastionInstanceID(ec2Client, tagName, tagValue)
		if err != nil {
			return err
		}
	}

	bastionAddr := fmt.Sprintf("%s@%s:%d", c.String("user"), instanceID, c.Int("port"))
	bastionEndpoint, err := sshutils.NewEC2Endpoint(bastionAddr, ec2Client, connectClient)
	if err != nil {
		return err
	}

	if tunnel := sshutils.NewEndpoint(c.String("tunnel")); tunnel.Host != "" {
		p := c.Int("local-port")
		if p == 0 {
			p = tunnel.Port
		}
		return sshutils.Tunnel(p, tunnel, bastionEndpoint)
	}

	chain := []sshutils.EndpointIface{
		bastionEndpoint,
	}

	for _, ep := range c.StringSlice("destination") {
		destEndpoint, err := sshutils.NewEC2Endpoint(ep, ec2Client, connectClient)
		if err != nil {
			return err
		}
		destEndpoint.UsePrivate = true
		chain = append(chain, destEndpoint)
	}

	return sshutils.Connect(chain...)
}

func getSpotRequestByTag(ec2Client *ec2.EC2, tagName, tagValue string) (*ec2.DescribeSpotInstanceRequestsOutput, error) {
	return ec2Client.DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + tagName),
				Values: aws.StringSlice([]string{tagValue}),
			},
			{
				Name:   aws.String("state"),
				Values: aws.StringSlice([]string{"active"}),
			},
			{
				Name:   aws.String("status-code"),
				Values: aws.StringSlice([]string{"fulfilled"}),
			},
		},
	})
}

func getInstanceByTag(ec2Client *ec2.EC2, tagName, tagValue string) (*ec2.DescribeInstancesOutput, error) {
	return ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + tagName),
				Values: aws.StringSlice([]string{tagValue}),
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: aws.StringSlice([]string{"running"}),
			},
		},
	})
}

func resolveBastionInstanceID(ec2Client *ec2.EC2, tagName, tagValue string) (string, error) {
	log.Info("Looking for bastion spot request")
	siro, err := getSpotRequestByTag(ec2Client, tagName, tagValue)
	if err != nil {
		return "", err
	}

	if len(siro.SpotInstanceRequests) > 0 {
		return aws.StringValue(siro.SpotInstanceRequests[rand.Intn(len(siro.SpotInstanceRequests))].InstanceId), nil
	}

	log.Info("No spot requests found, looking for instance directly")
	dio, err := getInstanceByTag(ec2Client, tagName, tagValue)
	if err != nil {
		return "", err
	}

	if len(dio.Reservations) > 0 {
		res := dio.Reservations[rand.Intn(len(dio.Reservations))]
		return aws.StringValue(res.Instances[rand.Intn(len(res.Instances))].InstanceId), nil
	}

	return "", errors.New("unable to find any valid bastion instances")
}

func ec2Client(region string) *ec2.EC2 {
	sess, err := awsSession.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatal(err)
	}

	return ec2.New(sess)
}

func connectClient(region string) *connect.EC2InstanceConnect {
	sess, err := awsSession.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatal(err)
	}

	return connect.New(sess)
}
