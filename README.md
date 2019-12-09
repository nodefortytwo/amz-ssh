# amz-ssh
a tool that replaces aws/aws-ec2-instance-connect-cli.

## Advantages

- Keys generated in memory and never stored to disk
- Native golang, no ssh exec
- Detects instance details based on tags, via spot requests
- will support tunnels


## Disclaimer

This tool was built purely to satisfy my requirements. It is not very flexible