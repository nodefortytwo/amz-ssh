package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	connect "github.com/aws/aws-sdk-go/service/ec2instanceconnect"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"golang.org/x/crypto/ssh"

	sshutils "github.com/nodefortytwo/amz-ssh/pkg/sshutils"
)

var region = "eu-west-1"

func main() {
	app := &cli.App{
		Name:   "amz-ssh",
		Usage:  "connect to an ec2 instance via ec2 connect",
		Action: run,
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
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)

	}
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
	log.Infof("Instance id: %s", instanceID)
	instanceOutput, err := getInstance(instanceID)
	if err != nil {
		return err
	}
	if len(instanceOutput.Reservations) == 0 || len(instanceOutput.Reservations[0].Instances) == 0 {
		return errors.New("instance not found")
	}

	instance := instanceOutput.Reservations[0].Instances[0]

	privateKey, publicKey, err := sshutils.GenerateKeys()
	if err != nil {
		return err
	}
	user := c.String("user")
	err = sendPublicKey(instance, user, publicKey)
	if err != nil {
		return err
	}

	return SSHConnect(aws.StringValue(instance.PublicIpAddress), user, privateKey)
}

func SSHConnect(ip, user, privateKey string) error {

	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return nil
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", ip+":22", sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial: %s", err)
	}

	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %s", err)
	}
	defer sess.Close()

	// Set IO
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	sess.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := sess.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}

	if err := sess.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	sess.Wait()

	return nil
}

func sendPublicKey(instance *ec2.Instance, user, publicKey string) error {

	out, err := connectClient().SendSSHPublicKey(&connect.SendSSHPublicKeyInput{
		AvailabilityZone: instance.Placement.AvailabilityZone,
		InstanceId:       instance.InstanceId,
		InstanceOSUser:   aws.String(user),
		SSHPublicKey:     aws.String(publicKey),
	})
	if err != nil {
		return err
	}

	if *out.Success != true {
		return fmt.Errorf("request failed but no error was returned. Request ID: %s", aws.StringValue(out.RequestId))
	}

	return nil
}

func getInstance(id string) (*ec2.DescribeInstancesOutput, error) {
	return ec2Client().DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
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
