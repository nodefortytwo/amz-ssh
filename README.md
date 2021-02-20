# amz-ssh

a tool that replaces aws/aws-ec2-instance-connect-cli.

## Advantages

- Keys generated in memory and never stored to disk
- Native golang, no ssh exec
- Detects instance details based on tags, via spot requests
- supports tunneling

## Getting Started

In aws launch a bastion server compatible with EC2 Instance Connect either via spot request or standard on demand. by default
`amz-ssh` looks for requests / instances tagged with `role:bastion`. The instance must be
 accessible to your network, usually this mean has a public ip.
 
see [EC2 Instance Connect docs](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-connect-methods.html) for more information

see this [simple Terraform bastion example](./examples/spot-bastion)

## How to use

`amz-ssh` is designed to be run in an already authenticated environment eg via [aws-vault](https://github.com/99designs/aws-vault).

Connect to a bastion / jump host launched by a spot request with a tag `role:bastion`

`amz-ssh`

Connect to a specific instance (ignoring tags etc)

`amz-ssh -i i-0eaa4d1c7f350216e`

Connect to a bastion with the tag `job:bastion-special`

`amz-ssh --tag job:bastion-special`

Tunnel through the default bastion

`amz-ssh -t somedatabase.example.com:5432`

Tunnel to a host through a specific instance

`amz-ssh -i i-0eaa4d1c7f350216e -t somedatabase.example.com:5432`

SSH to another host via the bastion

`amz-ssh -d i-0eaa4d1c7f350216e`

You can add as many `-d` flags as you want to continue chaining connections

`amz-ssh -d i-0eaa4d1c7f350216e -d i-0eaa4d1c7f67546e`

Specify the username and port

`amz-ssh -d ubuntu@i-0eaa4d1c7f350216e:2222`

## Manual

```
NAME:
   amz-ssh - connect to an ec2 instance via ec2 connect

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   0.0.0

COMMANDS:
   update   Update the cli
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --region value, -r value         [$AWS_REGION]
   --tag value                     (default: "role:bastion")
   --instance-id value, -i value   instance id to ssh to or tunnel through
   --user value, -u value          OS user of bastion (default: "ec2-user")
   --tunnel value, -t value        Host to tunnel to
   --destination value, -d value   destination to ssh to via the bastion. This flag can be provided multiple times to allow for multple hops
   --port value, -p value          (default: 22)
   --local-port value, --lp value  local port to map to, defaults to tunnel port (default: 0)
   --debug                         (default: Print debug information)
   --help, -h                      show help (default: false)
   --version, -v                   print the version (default: false)
```
