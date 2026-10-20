data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["bedrock-agentcore.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "harness" {
  name               = "${var.harness_name}-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
  tags               = var.tags
}

resource "aws_iam_role_policy" "harness" {
  name = "${var.harness_name}-policy"
  role = aws_iam_role.harness.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["bedrock:InvokeModel", "bedrock:InvokeModelWithResponseStream"]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "bedrock-agentcore:ListEvents",
          "bedrock-agentcore:CreateEvent",
          "bedrock-agentcore:GetEvent",
          "bedrock-agentcore:DeleteEvent",
          "bedrock-agentcore:InvokeMemory",
          "bedrock-agentcore:GetMemory",
          "bedrock-agentcore:ListMemories"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_bedrockagentcore_harness" "this" {
  harness_name       = var.harness_name
  execution_role_arn = aws_iam_role.harness.arn

  model {
    bedrock_model_config {
      model_id = var.model_id
    }
  }

  system_prompt {
    text = var.system_prompt
  }

  max_iterations  = var.max_iterations
  max_tokens      = var.max_tokens
  timeout_seconds = var.timeout_seconds

  tags = var.tags
}
