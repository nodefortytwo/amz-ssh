# amz-ssh

a tool that replaces aws/aws-ec2-instance-connect-cli.

## Advantages

- Keys generated in memory and never stored to disk
- Native golang, no ssh exec
- Detects instance details based on tags, via spot requests
- supports tunneling

## Getting Started

In aws launch a bastion service either via spot request or standard on demand. by default
`amz-ssh` looks for requests / instances tagged with `role:bastion`. The instance must be
 accessible to your network, usually this mean has a public ip.
 
 see [EC2 Instance Connect docs](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-connect-methods.html) for more information

## How to use

`amz-ssh` is designed to be run in an already authenticated environment eg via [aws-vault](https://github.com/99designs/aws-vault).

Connect to a bastion launched by a spot request with a tag `role:bastion` in `eu-west-1
`

`amz-ssh`

Connect to a specific a specific instance

`amz-ssh -i i-0eaa4d1c7f350216e`

Connect to a bastion with the tag `job:bastion-special`

`amz-ssh -t job:bastion-special`

Tunnel through the default bastion

`amz-ssh -t somedatabase.example.com:5432`

SSH to another host via the bastion

`amz-ssh -d i-0eaa4d1c7f350216e`

You can add as many `-d` flags as you want to continue chaining connections

`amz-ssh -d i-0eaa4d1c7f350216e -d i-0eaa4d1c7f67546e`

Specify the username and port

`amz-ssh -d ubuntu@i-0eaa4d1c7f350216e:2222`