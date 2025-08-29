terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Configure AWS Provider for LocalStack
provider "aws" {
  region = var.aws_region
  
  # LocalStack configuration (only when using LocalStack)
  dynamic "endpoints" {
    for_each = var.use_localstack ? [1] : []
    content {
      sqs            = var.localstack_endpoint
      s3             = var.localstack_endpoint
      lambda         = var.localstack_endpoint
      iam            = var.localstack_endpoint
      cloudwatchlogs = var.localstack_endpoint
      cloudwatch     = var.localstack_endpoint
    }
  }
  
  # Skip validation for LocalStack
  skip_credentials_validation = var.use_localstack
  skip_metadata_api_check     = var.use_localstack
  skip_requesting_account_id  = var.use_localstack
  
  # LocalStack credentials
  access_key = var.use_localstack ? "test" : null
  secret_key = var.use_localstack ? "test" : null

  # Force path-style addressing for S3 (required for LocalStack)
  s3_use_path_style = var.use_localstack
}

# ==================== Variables ====================

variable "aws_region" {
  description = "AWS region for deployment"
  type        = string
  default     = "us-east-2"
}

variable "environment" {
  description = "Environment name (local, dev, staging, prod)"
  type        = string
  default     = "local"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "audit-reports"
}

variable "use_localstack" {
  description = "Whether to use LocalStack for local development"
  type        = bool
  default     = false
}

variable "localstack_endpoint" {
  description = "LocalStack endpoint URL"
  type        = string
  default     = "http://localhost:4566"
}

variable "lambda_timeout" {
  description = "Lambda function timeout in seconds"
  type        = number
  default     = 180
}

variable "lambda_memory_size" {
  description = "Lambda function memory size in MB"
  type        = number
  default     = 512
}

variable "lambda_reserved_concurrent_executions" {
  description = "Reserved concurrent executions for Lambda"
  type        = number
  default     = 10
}

variable "sqs_visibility_timeout" {
  description = "SQS visibility timeout in seconds"
  type        = number
  default     = 180
}

variable "sqs_message_retention" {
  description = "SQS message retention period in seconds"
  type        = number
  default     = 345600  # 4 days
}

variable "dlq_message_retention" {
  description = "DLQ message retention period in seconds"
  type        = number
  default     = 1209600  # 14 days
}

variable "max_receive_count" {
  description = "Maximum number of receives before sending to DLQ"
  type        = number
  default     = 5
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 7
}

variable "lambda_deployment_package" {
  description = "Path to Lambda deployment package"
  type        = string
  default     = "lambda/downloader.zip"
}


# ==================== S3 Buckets ====================

# S3 Bucket for storing downloaded reports
resource "aws_s3_bucket" "reports" {
  bucket        = "${var.project_name}-${var.environment}"
  force_destroy = true

  tags = {
    Name        = "${var.project_name}-${var.environment}"
    Environment = var.environment
    Purpose     = "Store downloaded audit reports"
  }
}

# S3 Bucket for Lambda deployment packages
resource "aws_s3_bucket" "lambda_deployments" {
  bucket        = "${var.project_name}-${var.environment}-lambda-deployments"
  force_destroy = true

  tags = {
    Name        = "${var.project_name}-${var.environment}-lambda-deployments"
    Environment = var.environment
    Purpose     = "Lambda deployment packages"
  }
}

# ==================== SQS Queues ====================

# Dead Letter Queue for failed messages
resource "aws_sqs_queue" "dlq" {
  name                       = "${var.project_name}-${var.environment}-downloader-dlq"
  message_retention_seconds  = var.dlq_message_retention
  visibility_timeout_seconds = 300

  tags = {
    Name        = "${var.project_name}-${var.environment}-downloader-dlq"
    Environment = var.environment
    Purpose     = "Dead letter queue for failed download messages"
  }
}

# SQS Queue for downloader worker
resource "aws_sqs_queue" "downloader" {
  name                       = "${var.project_name}-${var.environment}-downloader"
  visibility_timeout_seconds = var.sqs_visibility_timeout
  message_retention_seconds  = var.sqs_message_retention
  max_message_size          = 262144  # 256 KB
  receive_wait_time_seconds = 20      # Long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-downloader"
    Environment = var.environment
    Worker      = "downloader"
  }
}

# ==================== IAM Roles and Policies ====================

# IAM Role for Lambda execution
resource "aws_iam_role" "lambda_execution" {
  name = "${var.project_name}-${var.environment}-lambda-downloader"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-lambda-downloader"
    Environment = var.environment
  }
}

# IAM Policy for Lambda execution
resource "aws_iam_policy" "lambda_downloader" {
  name        = "${var.project_name}-${var.environment}-lambda-downloader-policy"
  description = "Policy for Lambda downloader worker"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # CloudWatch Logs permissions
      {
        Sid    = "CloudWatchLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:${var.aws_region}:*:*"
      },
      # SQS permissions
      {
        Sid    = "SQSOperations"
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:ChangeMessageVisibility"
        ]
        Resource = aws_sqs_queue.downloader.arn
      },
      # DLQ permissions
      {
        Sid    = "DLQOperations"
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.dlq.arn
      },
      # S3 permissions
      {
        Sid    = "S3Operations"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.reports.arn}/*"
      },
      {
        Sid    = "S3ListBucket"
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.reports.arn
      },
      # CloudWatch Metrics permissions
      {
        Sid    = "CloudWatchMetrics"
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData"
        ]
        Resource = "*"
      }
    ]
  })
}

# Attach policy to role
resource "aws_iam_role_policy_attachment" "lambda_downloader" {
  role       = aws_iam_role.lambda_execution.name
  policy_arn = aws_iam_policy.lambda_downloader.arn
}

# ==================== CloudWatch ====================

# CloudWatch Log Group for Lambda
resource "aws_cloudwatch_log_group" "downloader" {
  name              = "/aws/lambda/${var.project_name}-${var.environment}-downloader"
  retention_in_days = var.log_retention_days

  tags = {
    Name        = "/aws/lambda/${var.project_name}-${var.environment}-downloader"
    Environment = var.environment
    Worker      = "downloader"
  }
}

# CloudWatch Metric Alarm for DLQ
resource "aws_cloudwatch_metric_alarm" "dlq_messages" {
  alarm_name          = "${var.project_name}-${var.environment}-downloader-dlq-messages"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name        = "ApproximateNumberOfMessagesVisible"
  namespace          = "AWS/SQS"
  period             = "300"
  statistic          = "Average"
  threshold          = "10"
  alarm_description  = "Alert when DLQ has messages"
  treat_missing_data = "notBreaching"

  dimensions = {
    QueueName = aws_sqs_queue.dlq.name
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-downloader-dlq-alarm"
    Environment = var.environment
  }
}

# CloudWatch Metric Alarm for Lambda Errors
resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  alarm_name          = "${var.project_name}-${var.environment}-downloader-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name        = "Errors"
  namespace          = "AWS/Lambda"
  period             = "300"
  statistic          = "Sum"
  threshold          = "5"
  alarm_description  = "Alert when Lambda function has errors"
  treat_missing_data = "notBreaching"

  dimensions = {
    FunctionName = aws_lambda_function.downloader.function_name
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-downloader-error-alarm"
    Environment = var.environment
  }
}

# ==================== Lambda Function ====================

# Lambda Function for Downloader
resource "aws_lambda_function" "downloader" {
  function_name = "${var.project_name}-${var.environment}-downloader"
  role          = aws_iam_role.lambda_execution.arn
  
  # For Go Lambda
  handler = "bootstrap"
  runtime = "provided.al2023"  # Amazon Linux 2023 runtime
  
  # Performance settings
  timeout     = var.lambda_timeout
  memory_size = var.lambda_memory_size
  
  # Deployment package
  filename         = var.lambda_deployment_package
  source_code_hash = fileexists(var.lambda_deployment_package) ? filebase64sha256(var.lambda_deployment_package) : null

  environment {
    variables = {
      # Environment config
      ENVIRONMENT = var.environment
      AWS_REGION  = var.aws_region
      
      # S3 config
      S3_BUCKET = aws_s3_bucket.reports.id
      
      # SQS config
      DLQ_URL = aws_sqs_queue.dlq.url
      
      # Logging
      LOG_LEVEL = var.environment == "prod" ? "info" : "debug"
      
      # Metrics
      METRICS_NAMESPACE = "${var.project_name}/${var.environment}"
    }
  }

  # Concurrency control
  reserved_concurrent_executions = var.lambda_reserved_concurrent_executions

  # Enable X-Ray tracing (optional)
  tracing_config {
    mode = var.environment == "prod" ? "Active" : "PassThrough"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-downloader"
    Environment = var.environment
    Worker      = "downloader"
  }

  depends_on = [
    aws_iam_role_policy_attachment.lambda_downloader,
    aws_cloudwatch_log_group.downloader
  ]
}

# ==================== Event Source Mapping ====================

# SQS to Lambda Event Source Mapping
resource "aws_lambda_event_source_mapping" "downloader" {
  event_source_arn = aws_sqs_queue.downloader.arn
  function_name    = aws_lambda_function.downloader.arn
  
  # Batch configuration
  batch_size                         = 10
  maximum_batching_window_in_seconds = 5
  
  # Enable partial batch responses for better error handling
  function_response_types = ["ReportBatchItemFailures"]
  
}

# ==================== Outputs ====================

output "lambda_function" {
  value = {
    arn           = aws_lambda_function.downloader.arn
    function_name = aws_lambda_function.downloader.function_name
    role_arn      = aws_iam_role.lambda_execution.arn
  }
  description = "Lambda function details"
}

output "sqs_queues" {
  value = {
    downloader = {
      url = aws_sqs_queue.downloader.url
      arn = aws_sqs_queue.downloader.arn
    }
    dlq = {
      url = aws_sqs_queue.dlq.url
      arn = aws_sqs_queue.dlq.arn
    }
  }
  description = "SQS queue details"
}

output "s3_buckets" {
  value = {
    reports = {
      id  = aws_s3_bucket.reports.id
      arn = aws_s3_bucket.reports.arn
    }
    lambda_deployments = {
      id  = aws_s3_bucket.lambda_deployments.id
      arn = aws_s3_bucket.lambda_deployments.arn
    }
  }
  description = "S3 bucket details"
}

output "cloudwatch" {
  value = {
    log_group = aws_cloudwatch_log_group.downloader.name
    alarms = {
      dlq_messages   = aws_cloudwatch_metric_alarm.dlq_messages.alarm_name
      lambda_errors  = aws_cloudwatch_metric_alarm.lambda_errors.alarm_name
    }
  }
  description = "CloudWatch resources"
}

output "region" {
  value       = var.aws_region
  description = "AWS region"
}

output "environment" {
  value       = var.environment
  description = "Environment name"
}