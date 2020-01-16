locals {
  ami_id = data.aws_ami.amazon-linux-2.id
  vpc_id = data.aws_subnet.selected.vpc_id
}
