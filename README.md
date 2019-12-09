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

`aws-vault exec {profile} -- amz-ssh`

Connect to a specifc a specific instance

`aws-vault exec {profile} -- amz-ssh -i i-0eaa4d1c7f350216e`

Tunnel through the default bastion

`aws-vault exec {profile} -- amz-ssh -t somedatabase.example.com:5432`


## Disclaimer

This tool was built purely to satisfy my requirements. It is not very flexible

I run all of my aws utils via aws-vault or similar so assume role etc happens there

`aws-vault exec {profile} -- amz-ssh`