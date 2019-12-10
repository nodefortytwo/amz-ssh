package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
			&cli.IntFlag{
				Name:        "port",
				Aliases:     []string{"p"},
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
	instanceID := c.String("instance-id")
	if instanceID == "" {
		log.Info("Looking for bastion spot request")
		siro, err := getSpotRequestByRole("bastion")
		if err != nil {
			return err
		}

		if len(siro.SpotInstanceRequests) > 0 {
			instanceID = aws.StringValue(siro.SpotInstanceRequests[0].InstanceId)
		} else {
			return errors.New("unable to find any valid bastion instances")
		}
	}

	privateKey, publicKey, err := sshutils.GenerateKeys()
	if err != nil {
		return err
	}
	user := c.String("user")

	bastionEndpoint, err := sshutils.NewEC2Endpoint(instanceID, user, privateKey, publicKey, ec2Client(), connectClient())
	if err != nil {
		return err
	}

	if tunnel := sshutils.NewEndpoint(c.String("tunnel")); tunnel.Host != "" {
		p := c.Int("port")
		if p == 0 {
			p = tunnel.Port
		}
		return sshutils.Tunnel(p, tunnel, bastionEndpoint)
	}

	return sshutils.Connect(bastionEndpoint)
}

func getSpotRequestByRole(role string) (*ec2.DescribeSpotInstanceRequestsOutput, error) {
	return ec2Client().DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:role"),
				Values: aws.StringSlice([]string{role}),
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
