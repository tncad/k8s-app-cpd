provider "aws" {
  profile = "default"
  region  = "us-east-1"
}
resource "aws_instance" "example" {
  ami           = "ami-2757f631"
  instance_type = "t2.micro"
}
output "instance_ip_addr" {
  value       = "${aws_instance.example.*.public_ip}"
  description = "Pulic IP address of the AWS EC2 instance."
}
