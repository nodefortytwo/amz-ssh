# amz-ssh

a tool that replaces aws/aws-ec2-instance-connect-cli.

## Advantages

- Keys generated in memory and never stored to disk
- Native golang, no ssh exec
- Detects instance details based on tags, via spot requests
- supports tunneling


## How to use

Connect to a bastion launched by a spot request with a tag `role:bastion` in `eu-west-1
`

`amz-ssh` is designed to be run in an already authenticated environment eg via `aws-vault`.

`aws-vault exec {profile} -- amz-ssh`

Connect to a specifc a specific instance

`amz-ssh -i i-0eaa4d1c7f350216e`

Tunnel through the default bastion

`amz-ssh -t somedatabase.example.com:5432`

SSH to another host after connecting to the bastion

`amz-ssh -d i-0eaa4d1c7f350216e`

You can add as many `-d` flags as you want to continue chaining connections

`amz-ssh -d i-0eaa4d1c7f350216e -d i-0eaa4d1c7f67546e`

Specify the username and port

`amz-ssh -d ubuntu@i-0eaa4d1c7f350216e:2222`

## Disclaimer

This tool was built purely to satisfy my requirements. It is not very flexible

