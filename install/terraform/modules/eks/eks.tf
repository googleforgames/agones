# Copyright 2020 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


terraform {
  required_version = ">= 0.12.6"
}

provider "aws" {
  version = "= 2.51.0"
  region  = var.region
}

data "aws_availability_zones" "available" {
}

resource "aws_security_group" "worker_group_mgmt_one" {
  name_prefix = "worker_group_mgmt_one"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"

    cidr_blocks = [
      "10.0.0.0/8",
    ]
  }
  ingress {
    from_port = 7000
    to_port   = 8000
    protocol  = "udp"

    cidr_blocks = [
      "0.0.0.0/0",
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.21.0"

  name                 = "test-vpc-lt"
  cidr                 = "10.0.0.0/16"
  azs                  = data.aws_availability_zones.available.names
  public_subnets       = ["10.0.4.0/24", "10.0.5.0/24", "10.0.6.0/24"]
  enable_dns_hostnames = false

  tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }

  public_subnet_tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                    = "1"
  }
}

module "eks" {
  source          = "git::github.com/terraform-aws-modules/terraform-aws-eks.git?ref=v7.0.1"
  cluster_name    = var.cluster_name
  subnets         = module.vpc.public_subnets
  vpc_id          = module.vpc.vpc_id
  cluster_version = "1.15"

  worker_groups_launch_template = [
    {
      name                          = "default"
      instance_type                 = var.machine_type
      asg_desired_capacity          = var.node_count
      asg_min_size                  = var.node_count
      asg_max_size                  = var.node_count
      additional_security_group_ids = [aws_security_group.worker_group_mgmt_one.id]
      public_ip                     = true
    },
    // Node Pools with taints for metrics and system
    {
      name                 = "agones-system"
      instance_type        = var.machine_type
      asg_desired_capacity = 1
      kubelet_extra_args   = "--node-labels=agones.dev/agones-system=true --register-with-taints=agones.dev/agones-system=true:NoExecute"
      public_ip            = true
    },
    {
      name                 = "agones-metrics"
      instance_type        = var.machine_type
      asg_desired_capacity = 1
      kubelet_extra_args   = "--node-labels=agones.dev/agones-metrics=true --register-with-taints=agones.dev/agones-metrics=true:NoExecute"
      public_ip            = true
    }
  ]
}