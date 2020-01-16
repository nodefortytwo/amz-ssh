variable "instance_type" {
  description = "Instance type to use for the bastion server"
  default = "t3a.nano"
}

variable "tags" {
  type = map(string)
  description = "tags to add to resources created"
}

variable "subnet_id" {
  type = string
  description = "(public} subnet to launch the instance in"
}
