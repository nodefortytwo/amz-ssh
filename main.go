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
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2" // imports as package "cli"

	"github.com/nodefortytwo/amz-ssh/pkg/sshutils"
)

var version = "0.0.0"
var region = "eu-west-1"

func main() {
	rand.Seed(time.Now().Unix())
	SetupSignalHandlers()
	app := &cli.App{
		Name:    "amz-ssh",
		Usage:   "connect to an ec2 instance via ec2 connect",
		Version: version,
		Action:  run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "region",
				Aliases:     []string{"r"},
				Destination: &region,
				Value:       "eu-west-1",
			},
			&cli.StringFlag{
				Name:  "tag",
				Value: "role:bastion",
			},
			&cli.StringFlag{
				Name:        "instance-id",
				Aliases:     []string{"i"},
				DefaultText: "instance id to ssh or tunnel through",
				Value:       "",
			},
			&cli.StringFlag{
				Name:        "user",
				Aliases:     []string{"u"},
				DefaultText: "os user of bastion",
				Value:       "ec2-user",
			},
			&cli.StringFlag{
				Name:        "tunnel",
				Aliases:     []string{"t"},
				DefaultText: "Host to tunnel to",
			},
			&cli.StringSliceFlag{
				Name:        "destination",
				Aliases:     []string{"d"},
				DefaultText: "destination to ssh to. multiple instances can be delimited by a comma",
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   22,
			},
			&cli.IntFlag{
				Name:        "local-port",
				Aliases:     []string{"lp"},
				DefaultText: "local port to map to, defaults to tunnel port",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "update",
				Usage:  "Update the cli",
				Action: update,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)

	}
}
func SetupSignalHandlers() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nGoodbye!")
		os.Exit(0)
	}()
}

func run(c *cli.Context) error {
	var tagName string
	var tagValue string

	if parts := strings.Split(c.String("tag"), ":"); len(parts) == 2 {
		tagName = parts[0]
		tagValue = parts[1]
	} else {
		return fmt.Errorf("%s is not a valid tag definition, use key:value", c.String("tag"))
	}

	instanceID, err := resolveBastionInstanceID(c.String("instance-id"), tagName, tagValue)
	if err != nil {
		return err
	}
	bastionAddr := fmt.Sprintf("%s@%s:%d", c.String("user"), instanceID, c.Int("port"))
	bastionEndpoint, err := sshutils.NewEC2Endpoint(bastionAddr, ec2Client(), connectClient())
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
		destEndpoint, err := sshutils.NewEC2Endpoint(ep, ec2Client(), connectClient())
		if err != nil {
			return err
		}
		destEndpoint.UsePrivate = true
		chain = append(chain, destEndpoint)
	}

	return sshutils.Connect(chain...)
}

func getSpotRequestByTag(tagName, tagValue string) (*ec2.DescribeSpotInstanceRequestsOutput, error) {
	return ec2Client().DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
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

func getInstanceByTag(tagName, tagValue string) (*ec2.DescribeInstancesOutput, error) {
	return ec2Client().DescribeInstances(&ec2.DescribeInstancesInput{
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

func resolveBastionInstanceID(instanceID, tagName, tagValue string) (string, error) {
	if instanceID != "" {
		return instanceID, nil
	}

	log.Info("Looking for bastion spot request")
	siro, err := getSpotRequestByTag(tagName, tagValue)
	if err != nil {
		return "", err
	}

	if len(siro.SpotInstanceRequests) > 0 {
		return aws.StringValue(siro.SpotInstanceRequests[rand.Intn(len(siro.SpotInstanceRequests))].InstanceId), nil
	}

	log.Info("No spot requests found, looking for instance directly")
	dio, err := getInstanceByTag(tagName, tagValue)
	if err != nil {
		return "", err
	}

	if len(dio.Reservations) > 0 {
		res := dio.Reservations[rand.Intn(len(dio.Reservations))]
		return aws.StringValue(res.Instances[rand.Intn(len(res.Instances))].InstanceId), nil
	}

	return "", errors.New("unable to find any valid bastion instances")
	// TODO: look for asg by tag
}

func ec2Client() *ec2.EC2 {
	sess, err := awsSession.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatal(err)
	}

	return ec2.New(sess)
}

func connectClient() *connect.EC2InstanceConnect {
	sess, err := awsSession.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.Fatal(err)
	}

	return connect.New(sess)
}

func update(c *cli.Context) error {
	v := semver.MustParse(c.App.Version)
	latest, err := selfupdate.UpdateSelf(v, "nodefortytwo/amz-ssh")
	if err != nil {
		log.Println("Binary update failed:", err)
		return nil
	}
	if latest.Version.Equals(v) {
		// latest version is the same as current version. It means current binary is up to date.
		log.Println("Current binary is the latest version", version)
	} else {
		log.Println("Successfully updated to version", latest.Version)
		log.Println("Release note:\n", latest.ReleaseNotes)
	}

	return nil
}
