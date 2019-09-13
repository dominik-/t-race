provider "aws" {
  version = "~> 2.0"
  region  = "eu-west-1"
}

resource "aws_placement_group" "default-cluster" {
  name     = "default-cluster-pg"
  strategy = "cluster"
}

resource "aws_instance" "test-ecs-image-instance" {
  ami           = "${var.ami-id}"
  instance_type = "t2.micro"
  # placement_group = "default-cluster-pg"
  vpc_security_group_ids = ["${aws_security_group.t-race-sg.id}", "${aws_security_group.ssh.id}"]
  key_name = "${var.my-key-name}"
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnet" "default" {
  vpc_id            = "${data.aws_vpc.default.id}"
  default_for_az    = true
  availability_zone = "${var.az}"
}

data "aws_subnet_ids" "all" {
  vpc_id = "${data.aws_vpc.default.id}"
}

resource "aws_security_group" "t-race-sg" {
  name        = "t-race-sg"
  description = "SG for inbound traffic from benchmark master"
  vpc_id      = "${data.aws_vpc.default.id}"

  ingress {
    from_port   = 7000
    to_port     = 7099
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  ingress {
    from_port   = 8000
    to_port     = 8099
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  ingress {
    from_port   = 9000
    to_port     = 9099
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

resource "aws_security_group" "ssh" {
  name        = "ssh-sg"
  description = "SSH connection"
  vpc_id      = "${data.aws_vpc.default.id}"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}
