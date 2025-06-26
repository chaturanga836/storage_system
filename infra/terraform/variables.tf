variable "region" {
  description = "AWS region to deploy resources in"
  type        = string
  default     = "ap-south-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "ubuntu_ami_id" {
  description = "AMI ID for Ubuntu 24.04 LTS in ap-south-1"
  type        = string
  default     = "ami-0f5dce22c757b180b"
}

variable "volume_size" {
  description = "Size of EBS volume in GB"
  type        = number
  default     = 50
}

variable "ssh_key_name" {
  description = "Name of the AWS EC2 key pair"
  type        = string
  default     = "my-key"
}

variable "ssh_public_key_path" {
  description = "Path to your public SSH key"
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "security_group_name" {
  description = "Name of the EC2 security group"
  type        = string
  default     = "control-node-sg"
}

variable "ec2_name_tag" {
  description = "Tag to apply to the EC2 instance"
  type        = string
  default     = "storage-test-node"
}
