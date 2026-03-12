# The Terraform Provider Configuration
# DO NOT DELETE OR ALTER THIS FILE.

# This configuration determines how your terraform itself will run.
# Terraform keeps track of the infrastructure it knows about via a terraform
# statefile (i.e. tfstate), and we are configuring that the terraform statefile
# be stored in an S3 bucket.
#
# We are also configuring terraform to use a DynamoDB lock table in order to
# correctly update the terraform statefile and access the mutex lock (locks
# against concurrent applications of terraform changes).

terraform {
  backend "s3" {
    bucket         = "ibotta-infrastructure"
    key            = "terraform/atlantis/passenger-datadog-monitor/monolith-production/terraform.tfstate"
    dynamodb_table = "infrastructure-terraform-lock"
    region         = "us-east-1"
  }

  required_version = ">= 1.9.8"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.100.0"
    }
  }
}

provider "aws" {
  region      = "us-east-1"

  assume_role {
    role_arn = "arn:aws:iam::264606497040:role/${var.provider_role}"
  }
}

variable "provider_role" {
  type        = string
  default     = "atlantis/spoke/atlantis-microservices"
  description = "The provider role variable with a default allows overriding when running local plans"
}