provider "aws" {
  version = "~> 2.0"
  region  = "eu-west-1"
}

resource "aws_placement_group" "default-cluster" {
  name     = "default-cluster-pg"
  strategy = "cluster"
}

resource "aws_instance" "environment" {
  ami = "${var.ami-id}"
  instance_type = "t3a.micro"
  #placement_group = "default-cluster-pg"
  vpc_security_group_ids = ["${aws_security_group.t-race-sg.id}", "${aws_security_group.ssh.id}"]
  key_name = "${var.my-key-name}"
  count = 2

  provisioner "local-exec" {
    command = "echo first"
  }
}

resource "aws_instance" "tracing-backend" {
  ami = "${var.ami-id}"
  instance_type = "t3a.small"
  #placement_group = "default-cluster-pg"
  vpc_security_group_ids = ["${aws_security_group.jaeger-backend.id}", "${aws_security_group.ssh.id}"]
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

resource "aws_security_group" "jaeger-backend" {
  name        = "jaeger-backend-sg"
  description = "SG for inbound traffic to jaeger backend components"
  vpc_id      = "${data.aws_vpc.default.id}"
  # Jaeger Query Web UI
  ingress {
    from_port   = 16686
    to_port     = 16686
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  # Jaeger Collector gRPC
  ingress {
    from_port   = 14250
    to_port     = 14250
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  # Jaeger Collector direct from client via Thrift; Jaeger Collector Prometheus metrics
  ingress {
    from_port   = 14268
    to_port     = 14269
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  # Jaeger Agent Thrift UDP 
  ingress {
    from_port   = 6831
    to_port     = 6831
    protocol    = "UDP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  # Jaeger Agent Prometheus metrics
  ingress {
    from_port   = 14271
    to_port     = 14271
    protocol    = "TCP"
    cidr_blocks = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  # Prometheus
  ingress {
    from_port   = 9090
    to_port     = 9090
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

  egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    cidr_blocks     = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

}