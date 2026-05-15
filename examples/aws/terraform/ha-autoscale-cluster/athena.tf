// Athena audit backend + Access Monitoring resources.
// Only created when var.enable_athena = true.
// This replaces DynamoDB as the audit events backend and enables the Access Monitoring
// feature in the Teleport UI. See https://goteleport.com/docs/identity-governance/access-monitoring/

locals {
  // Glue names cannot contain hyphens
  athena_db_name    = replace(var.cluster_name, "-", "_")
  athena_table_name = "audit_events"

  // Full Athena URI written into teleport.yaml audit_events_uri.
  // try() handles count=0 safely when enable_athena=false.
  athena_events_uri = var.enable_athena ? format(
    "athena://%s.%s?topicArn=%s&largeEventsS3=s3://%s/large_payloads&locationS3=s3://%s/events&workgroup=%s&queueURL=%s&queryResultsS3=s3://%s/query_results&limiterBurst=450&limiterRefillAmount=450&limiterRefillTime=1s",
    local.athena_db_name,
    local.athena_table_name,
    try(aws_sns_topic.audit_topic[0].arn, ""),
    try(aws_s3_bucket.transient_storage[0].bucket, ""),
    try(aws_s3_bucket.long_term_storage[0].bucket, ""),
    try(aws_athena_workgroup.workgroup[0].name, ""),
    try(aws_sqs_queue.audit_queue[0].url, ""),
    try(aws_s3_bucket.transient_storage[0].bucket, ""),
  ) : ""

  access_monitoring_role_arn       = try(aws_iam_role.access_monitoring_role[0].arn, "")
  access_monitoring_workgroup_name = try(aws_athena_workgroup.access_monitoring_workgroup[0].name, "")
  access_monitoring_report_results = var.enable_athena ? format("s3://%s/report_results", try(aws_s3_bucket.long_term_storage[0].bucket, "")) : ""
}

// KMS key used for SNS, SQS, and S3 encryption across the audit pipeline.
resource "aws_kms_key" "audit_key" {
  count               = var.enable_athena ? 1 : 0
  description         = "${var.cluster_name} Teleport audit log encryption"
  enable_key_rotation = true
}

resource "aws_kms_key_policy" "audit_key_policy" {
  count  = var.enable_athena ? 1 : 0
  key_id = aws_kms_key.audit_key[0].id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "Default Policy"
        Effect   = "Allow"
        Action   = ["kms:*"]
        Resource = "*"
        Principal = {
          AWS = data.aws_caller_identity.current.account_id
        }
      },
      {
        Sid    = "SnsUsage"
        Effect = "Allow"
        Action = ["kms:GenerateDataKey", "kms:Decrypt"]
        Principal = {
          Service = "sns.amazonaws.com"
        }
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:SourceAccount" = data.aws_caller_identity.current.account_id
          }
          ArnLike = {
            "aws:SourceArn" = aws_sns_topic.audit_topic[0].arn
          }
        }
      },
    ]
  })
}

resource "aws_kms_alias" "audit_key_alias" {
  count         = var.enable_athena ? 1 : 0
  name          = "alias/${var.cluster_name}-audit"
  target_key_id = aws_kms_key.audit_key[0].key_id
}

// Auth service publishes each audit event to this SNS topic.
resource "aws_sns_topic" "audit_topic" {
  count             = var.enable_athena ? 1 : 0
  name              = "${var.cluster_name}-audit-events"
  kms_master_key_id = aws_kms_key.audit_key[0].arn
}

// Dead-letter queue for events that could not be processed after max_receive_count attempts.
resource "aws_sqs_queue" "audit_queue_dlq" {
  count                             = var.enable_athena ? 1 : 0
  name                              = "${var.cluster_name}-audit-events-dlq"
  kms_master_key_id                 = aws_kms_key.audit_key[0].arn
  kms_data_key_reuse_period_seconds = 300
  message_retention_seconds         = 604800 // 7 days
}

// Auth service reads batches from this queue, converts to Parquet, and writes to S3.
resource "aws_sqs_queue" "audit_queue" {
  count                             = var.enable_athena ? 1 : 0
  name                              = "${var.cluster_name}-audit-events"
  kms_master_key_id                 = aws_kms_key.audit_key[0].arn
  kms_data_key_reuse_period_seconds = 300
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.audit_queue_dlq[0].arn
    maxReceiveCount     = 10
  })
}

resource "aws_sns_topic_subscription" "audit_sqs_target" {
  count                = var.enable_athena ? 1 : 0
  topic_arn            = aws_sns_topic.audit_topic[0].arn
  protocol             = "sqs"
  endpoint             = aws_sqs_queue.audit_queue[0].arn
  raw_message_delivery = true
}

data "aws_iam_policy_document" "audit_sqs_policy" {
  count = var.enable_athena ? 1 : 0
  statement {
    actions   = ["SQS:SendMessage"]
    effect    = "Allow"
    resources = [aws_sqs_queue.audit_queue[0].arn]
    principals {
      type        = "Service"
      identifiers = ["sns.amazonaws.com"]
    }
    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = [aws_sns_topic.audit_topic[0].arn]
    }
  }
}

resource "aws_sqs_queue_policy" "audit_policy" {
  count     = var.enable_athena ? 1 : 0
  queue_url = aws_sqs_queue.audit_queue[0].url
  policy    = data.aws_iam_policy_document.audit_sqs_policy[0].json
}

// Long-term S3 bucket: Parquet audit event files + Access Monitoring report results.
resource "aws_s3_bucket" "long_term_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = "${var.cluster_name}-teleport-audit-longterm"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "long_term_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.long_term_storage[0].id
  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.audit_key[0].arn
      sse_algorithm     = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_ownership_controls" "long_term_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.long_term_storage[0].id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_versioning" "long_term_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.long_term_storage[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "long_term_storage" {
  count                   = var.enable_athena ? 1 : 0
  bucket                  = aws_s3_bucket.long_term_storage[0].id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

// Transient S3 bucket: large event payloads + Athena query results.
// A lifecycle rule expires objects after 1 day to control costs.
resource "aws_s3_bucket" "transient_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = "${var.cluster_name}-teleport-audit-transient"
}

resource "aws_s3_bucket_lifecycle_configuration" "transient_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.transient_storage[0].id
  rule {
    id     = "expire-transient"
    status = "Enabled"
    filter {}
    expiration {
      days = 1
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "transient_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.transient_storage[0].id
  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.audit_key[0].arn
      sse_algorithm     = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_ownership_controls" "transient_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.transient_storage[0].id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_versioning" "transient_storage" {
  count  = var.enable_athena ? 1 : 0
  bucket = aws_s3_bucket.transient_storage[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_public_access_block" "transient_storage" {
  count                   = var.enable_athena ? 1 : 0
  bucket                  = aws_s3_bucket.transient_storage[0].id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

// Glue database and table that Athena queries against.
// Uses Partition Projection to avoid running MSCK REPAIR TABLE after each deploy.
resource "aws_glue_catalog_database" "audit_db" {
  count = var.enable_athena ? 1 : 0
  name  = local.athena_db_name
}

resource "aws_glue_catalog_table" "audit_table" {
  count         = var.enable_athena ? 1 : 0
  name          = local.athena_table_name
  database_name = aws_glue_catalog_database.audit_db[0].name
  table_type    = "EXTERNAL_TABLE"

  parameters = {
    "EXTERNAL"                            = "TRUE"
    "projection.enabled"                  = "true"
    "projection.event_date.type"          = "date"
    "projection.event_date.format"        = "yyyy-MM-dd"
    "projection.event_date.interval"      = "1"
    "projection.event_date.interval.unit" = "DAYS"
    "projection.event_date.range"         = "NOW-4YEARS,NOW"
    "storage.location.template"           = format("s3://%s/events/$${event_date}/", aws_s3_bucket.long_term_storage[0].bucket)
    "classification"                      = "parquet"
    "parquet.compression"                 = "SNAPPY"
  }

  storage_descriptor {
    location      = format("s3://%s", aws_s3_bucket.long_term_storage[0].bucket)
    input_format  = "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat"
    output_format = "org.apache.hadoop.hive.ql.io.parquet.MapredParquetOutputFormat"
    ser_de_info {
      name                  = "audit_events"
      parameters            = { "serialization.format" = "1" }
      serialization_library = "org.apache.hadoop.hive.ql.io.parquet.serde.ParquetHiveSerDe"
    }
    columns {
      name = "uid"
      type = "string"
    }
    columns {
      name = "session_id"
      type = "string"
    }
    columns {
      name = "event_type"
      type = "string"
    }
    columns {
      name = "event_time"
      type = "timestamp"
    }
    columns {
      name = "event_data"
      type = "string"
    }
    columns {
      name = "user"
      type = "string"
    }
  }

  partition_keys {
    name = "event_date"
    type = "date"
  }
}

// Athena workgroup for audit event queries.
resource "aws_athena_workgroup" "workgroup" {
  count         = var.enable_athena ? 1 : 0
  name          = "${var.cluster_name}-audit"
  force_destroy = true
  configuration {
    bytes_scanned_cutoff_per_query = 1073741824 // 1 GB
    engine_version {
      selected_engine_version = "Athena engine version 3"
    }
    result_configuration {
      output_location = format("s3://%s/query_results", aws_s3_bucket.transient_storage[0].bucket)
      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = aws_kms_key.audit_key[0].arn
      }
    }
  }
}

// Dedicated Athena workgroup for Access Monitoring SQL reports.
resource "aws_athena_workgroup" "access_monitoring_workgroup" {
  count         = var.enable_athena ? 1 : 0
  name          = "${var.cluster_name}-access-monitoring"
  force_destroy = true
  configuration {
    publish_cloudwatch_metrics_enabled = true
    bytes_scanned_cutoff_per_query     = 322122547200 // ~300 GB
    engine_version {
      selected_engine_version = "Athena engine version 3"
    }
    result_configuration {
      output_location = format("s3://%s/results", aws_s3_bucket.transient_storage[0].bucket)
      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = aws_kms_key.audit_key[0].arn
      }
    }
  }
}

// IAM role assumed by the Teleport Auth Service to execute Access Monitoring queries.
// Uses a separate role from the auth server instance profile so permissions
// are scoped to query execution only.
resource "aws_iam_role" "access_monitoring_role" {
  count = var.enable_athena ? 1 : 0
  name  = "${var.cluster_name}-access-monitoring"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "IamPrincipal"
      Effect = "Allow"
      Principal = {
        AWS = [aws_iam_role.auth.arn]
      }
      Action = ["sts:AssumeRole", "sts:TagSession"]
    }]
  })
}

data "aws_iam_policy_document" "access_monitoring_policy" {
  count = var.enable_athena ? 1 : 0

  statement {
    actions = [
      "s3:ListBucketMultipartUploads",
      "s3:GetBucketLocation",
      "s3:ListBucketVersions",
      "s3:ListBucket",
    ]
    resources = [
      aws_s3_bucket.transient_storage[0].arn,
      aws_s3_bucket.long_term_storage[0].arn,
    ]
  }

  statement {
    actions = [
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:PutObject",
    ]
    resources = [
      "${aws_s3_bucket.long_term_storage[0].arn}/report_results/*",
      "${aws_s3_bucket.transient_storage[0].arn}/results/*",
    ]
  }

  statement {
    actions = [
      "s3:ListMultipartUploadParts",
      "s3:GetObjectVersion",
      "s3:GetObject",
      "s3:AbortMultipartUpload",
    ]
    resources = [
      "${aws_s3_bucket.transient_storage[0].arn}/results/*",
      "${aws_s3_bucket.long_term_storage[0].arn}/events/*",
      "${aws_s3_bucket.long_term_storage[0].arn}/report_results/*",
    ]
  }

  statement {
    actions = [
      "glue:GetTable",
      "athena:StartQueryExecution",
      "athena:GetQueryResults",
      "athena:GetQueryExecution",
    ]
    resources = [
      aws_glue_catalog_table.audit_table[0].arn,
      aws_glue_catalog_database.audit_db[0].arn,
      "arn:aws:glue:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:catalog",
      aws_athena_workgroup.access_monitoring_workgroup[0].arn,
    ]
  }

  statement {
    actions = [
      "kms:GenerateDataKey",
      "kms:Decrypt",
    ]
    resources = [aws_kms_key.audit_key[0].arn]
  }
}

resource "aws_iam_policy" "access_monitoring_policy" {
  count  = var.enable_athena ? 1 : 0
  name   = "${var.cluster_name}-access-monitoring"
  policy = data.aws_iam_policy_document.access_monitoring_policy[0].json
}

resource "aws_iam_role_policy_attachment" "access_monitoring_policy_attachment" {
  count      = var.enable_athena ? 1 : 0
  role       = aws_iam_role.access_monitoring_role[0].name
  policy_arn = aws_iam_policy.access_monitoring_policy[0].arn
}
