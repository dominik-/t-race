variable "ami-id" {
  description = "AMI ID to be used"
  type = string
  default = "ami-0ae254c8a2d3346a7"
}

variable "az" {
  description = "AWS Availability Zone to use."
  type = string
  default = "eu-west-1a"
}

variable "my-key-name" {
  description = "Key name to use for ssh-ing into instance."
  type = string
  default = "dominik-desktop"
}

output "env_external" {
  value = "${join(",", aws_instance.environment.*.public_dns)}"
}

output "env_internal" {
  value = "${join(",", aws_instance.environment.*.private_ip)}"
}

output "backend" {
  value = aws_instance.tracing-backend.public_dns
}
