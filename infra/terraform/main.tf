provider "aws" {
  region = "ap-south-1" # Mumbai
}

resource "aws_key_pair" "default" {
  key_name   = "my-key"
  public_key = file("~/.ssh/id_rsa.pub") # Update to your SSH public key path
}

resource "aws_security_group" "control_node_sg" {
  name        = "control-node-sg"
  description = "Allow SSH and port 8081"

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # 🔒 Restrict for production
  }

  ingress {
    description = "Go API (8081)"
    from_port   = 8081
    to_port     = 8081
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # 🔒 Restrict for production
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "storage_test" {
  ami             = "ami-0f5dce22c757b180b" # ✅ Ubuntu 24.04 LTS (x86_64, ap-south-1)
  instance_type   = "t3.micro"
  key_name        = aws_key_pair.default.key_name
  security_groups = [aws_security_group.control_node_sg.name]

  user_data = file("init.sh") # Bootstrap Go + Python

  root_block_device {
    volume_size = 50
    volume_type = "gp3"
  }

  tags = {
    Name = "storage-test-node"
  }
}
