# Core Configuration
aws_region   = "us-east-2"
environment  = "local"
project_name = "audit-reports"

# LocalStack Configuration (set to true for local development)
use_localstack      = true
localstack_endpoint = "http://localhost:4566"

# Lambda Configuration
lambda_timeout                        = 180
lambda_memory_size                   = 512
lambda_reserved_concurrent_executions = 10
lambda_deployment_package            = "placeholder.zip"

# SQS Configuration
sqs_visibility_timeout = 180
sqs_message_retention  = 345600  # 4 days
dlq_message_retention  = 1209600 # 14 days
max_receive_count      = 5

# CloudWatch Configuration
log_retention_days = 7