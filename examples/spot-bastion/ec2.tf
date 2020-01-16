resource "aws_spot_instance_request" "server" {
  ami                         = local.ami_id
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [aws_security_group.bastion.id]
  associate_public_ip_address = true
  wait_for_fulfillment        = true

  root_block_device {
    volume_size           = 15
    delete_on_termination = true
  }

  tags = merge({ "Name" = "Bastion Host" }, var.tags)
}

