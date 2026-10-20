terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.90.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = ">= 2.4.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

module "bedrock_harness" {
  source = "./modules/bedrock_harness"

  harness_name  = var.harness_name
  model_id      = var.model_id
  system_prompt = var.system_prompt
  tags          = var.tags
}

module "a2a_lambda" {
  source = "./modules/a2a_lambda"

  function_name     = "${var.harness_name}-lambda"
  harness_arn       = module.bedrock_harness.harness_arn
  model_id          = var.model_id
  lambda_source_dir = "${path.module}/lambda_src"
  tags              = var.tags
}
