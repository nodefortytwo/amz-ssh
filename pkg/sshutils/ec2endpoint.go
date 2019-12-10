package sshutils

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	connect "github.com/aws/aws-sdk-go/service/ec2instanceconnect"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type EC2Endpoint struct {
	InstanceID string
	Port       int
	User       string
	PrivateKey string
	PublicKey  string

	Instance      *ec2.Instance
	EC2Client     *ec2.EC2
	ConnectClient *connect.EC2InstanceConnect
}

func NewEC2Endpoint(InstanceID, user, privateKey, publicKey string, ec2Client *ec2.EC2, connectClient *connect.EC2InstanceConnect) (*EC2Endpoint, error) {
	endpoint := EC2Endpoint{
		InstanceID: InstanceID,
		User:       user,
		Port:       22,
		PrivateKey: privateKey,
		PublicKey:  publicKey,

		EC2Client:     ec2Client,
		ConnectClient: connectClient,
	}
	var err error

	endpoint.Instance, err = getEC2Instance(endpoint.InstanceID, endpoint.EC2Client)
	if err != nil {
		return &endpoint, err
	}

	return &endpoint, nil
}

func (e *EC2Endpoint) String() string {
	err := sendPublicKey(e.Instance, e.User, e.PublicKey, e.ConnectClient)
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%s:%d", aws.StringValue(e.Instance.PublicIpAddress), e.Port)
}

func (e *EC2Endpoint) GetSSHConfig() (*ssh.ClientConfig, error) {
	key, err := ssh.ParsePrivateKey([]byte(e.PrivateKey))
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: e.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func sendPublicKey(instance *ec2.Instance, user, publicKey string, client *connect.EC2InstanceConnect) error {

	out, err := client.SendSSHPublicKey(&connect.SendSSHPublicKeyInput{
		AvailabilityZone: instance.Placement.AvailabilityZone,
		InstanceId:       instance.InstanceId,
		InstanceOSUser:   aws.String(user),
		SSHPublicKey:     aws.String(publicKey),
	})
	if err != nil {
		return err
	}

	if !*out.Success {
		return fmt.Errorf("request failed but no error was returned. Request ID: %s", aws.StringValue(out.RequestId))
	}

	return nil
}

func getEC2Instance(id string, client *ec2.EC2) (*ec2.Instance, error) {
	instanceOutput, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})

	if err != nil {
		return nil, err
	}

	if len(instanceOutput.Reservations) == 0 || len(instanceOutput.Reservations[0].Instances) == 0 {
		return nil, errors.New("instance not found")
	}

	return instanceOutput.Reservations[0].Instances[0], nil
}
