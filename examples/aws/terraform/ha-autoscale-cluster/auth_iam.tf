// Auth instance profile and roles
resource "aws_iam_role" "auth" {
  name = "${var.cluster_name}-auth"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {"Service": "ec2.amazonaws.com"},
            "Action": "sts:AssumeRole"
        }
    ]
}
EOF

}

// Auth servers publish various secrets to SSM parameter store
// for example join tokens, so other nodes and proxies can join the cluster.
resource "aws_iam_instance_profile" "auth" {
  name       = "${var.cluster_name}-auth"
  role       = aws_iam_role.auth.name
  depends_on = [aws_iam_role_policy.auth_ssm]
}

resource "aws_iam_role_policy" "auth_ssm" {
  name = "${var.cluster_name}-auth-ssm"
  role = aws_iam_role.auth.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ssm:DescribeParameters",
                "ssm:GetParameters",
                "ssm:GetParametersByPath",
                "ssm:GetParameter",
                "ssm:PutParameter",
                "ssm:DeleteParameter"
            ],
            "Resource": "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/teleport/${var.cluster_name}/*"
        },
        {
         "Effect":"Allow",
         "Action":[
            "kms:Decrypt"
         ],
         "Resource":[
            "arn:aws:kms:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:key/${data.aws_kms_alias.ssm.target_key_id}"
         ]
      }
    ]
}
EOF

}

// Auth server uses DynamoDB as a backend, and this is to allow read/write from the dynamo tables
resource "aws_iam_role_policy" "auth_dynamo" {
  name = "${var.cluster_name}-auth-dynamo"
  role = aws_iam_role.auth.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllActionsOnTeleportDB",
            "Effect": "Allow",
            "Action": "dynamodb:*",
            "Resource": "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${aws_dynamodb_table.teleport.name}"
        },
        {
            "Sid": "AllActionsOnTeleportEventsDB",
            "Effect": "Allow",
            "Action": "dynamodb:*",
            "Resource": "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${aws_dynamodb_table.teleport_events.name}"
        },
        {
            "Sid": "AllActionsOnTeleportEventsIndexDB",
            "Effect": "Allow",
            "Action": "dynamodb:*",
            "Resource": "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${aws_dynamodb_table.teleport_events.name}/index/*"
        },
        {
            "Sid": "AllActionsOnTeleportStreamsDB",
            "Effect": "Allow",
            "Action": "dynamodb:*",
            "Resource": "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${aws_dynamodb_table.teleport.name}/stream/*"
        }
    ]
}
EOF

}

// Allow auth servers to update locks
resource "aws_iam_role_policy" "auth_locks" {
  name = "${var.cluster_name}-auth-locks"
  role = aws_iam_role.auth.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllActionsOnLocks",
            "Effect": "Allow",
            "Action": "dynamodb:*",
            "Resource": "arn:aws:dynamodb:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/${aws_dynamodb_table.locks.name}"
        }
    ]
}
EOF

}

// S3 is used for Let's Encrypt, auth servers request certificates from Let's Encrypt
// and publish to S3 encrypted bucket. SSM is not used, because certificates and private keys
// are too big for SSM.
resource "aws_iam_role_policy" "auth_s3" {
  name = "${var.cluster_name}-auth-s3"
  role = aws_iam_role.auth.id

  policy = <<EOF
{
   "Version": "2012-10-17",
   "Statement": [
     {
       "Effect": "Allow",
       "Action": [
         "s3:ListBucket",
         "s3:ListBucketVersions",
         "s3:ListBucketMultipartUploads",
         "s3:AbortMultipartUpload"
        ],
       "Resource": ["arn:aws:s3:::${aws_s3_bucket.certs.bucket}"]
     },
     {
       "Effect": "Allow",
       "Action": [
         "s3:PutObject",
         "s3:GetObject",
         "s3:GetObjectVersion"
       ],
       "Resource": ["arn:aws:s3:::${aws_s3_bucket.certs.bucket}/*"]
     }
   ]
 }

EOF

}

// Athena audit backend permissions - only attached when enable_athena = true.
// Auth service publishes to SNS, consumes from SQS, writes Parquet to S3, and
// assumes the access_monitoring_role to execute Access Monitoring queries.
resource "aws_iam_role_policy" "auth_athena" {
  count = var.enable_athena ? 1 : 0
  name  = "${var.cluster_name}-auth-athena"
  role  = aws_iam_role.auth.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "AllowPublishSNS"
        Effect   = "Allow"
        Action   = ["sns:Publish"]
        Resource = aws_sns_topic.audit_topic[0].arn
      },
      {
        Sid    = "AllowReceiveSQS"
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
        ]
        Resource = aws_sqs_queue.audit_queue[0].arn
      },
      {
        Sid    = "AllowListingMultipartUploads"
        Effect = "Allow"
        Action = [
          "s3:ListBucketMultipartUploads",
          "s3:GetBucketLocation",
          "s3:ListBucketVersions",
          "s3:ListBucket",
        ]
        Resource = [
          aws_s3_bucket.transient_storage[0].arn,
          aws_s3_bucket.long_term_storage[0].arn,
        ]
      },
      {
        Sid    = "AllowMultipartAndObjectAccess"
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:ListMultipartUploadParts",
          "s3:GetObjectVersion",
          "s3:GetObject",
          "s3:DeleteObjectVersion",
          "s3:DeleteObject",
          "s3:AbortMultipartUpload",
        ]
        Resource = [
          "${aws_s3_bucket.transient_storage[0].arn}/results/*",
          "${aws_s3_bucket.transient_storage[0].arn}/large_payloads/*",
          "${aws_s3_bucket.long_term_storage[0].arn}/events/*",
        ]
      },
      {
        Sid    = "AllowAthenaKMSUsage"
        Effect = "Allow"
        Action = [
          "kms:GenerateDataKey",
          "kms:Decrypt",
        ]
        Resource = aws_kms_key.audit_key[0].arn
      },
      {
        Sid    = "AllowAthenaQuery"
        Effect = "Allow"
        Action = [
          "glue:GetTable",
          "athena:StartQueryExecution",
          "athena:GetQueryResults",
          "athena:GetQueryExecution",
        ]
        Resource = [
          aws_glue_catalog_table.audit_table[0].arn,
          aws_glue_catalog_database.audit_db[0].arn,
          "arn:aws:glue:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:catalog",
          aws_athena_workgroup.workgroup[0].arn,
        ]
      },
      {
        Sid    = "AssumeAccessMonitoringRole"
        Effect = "Allow"
        Action = ["sts:AssumeRole", "sts:TagSession"]
        Resource = aws_iam_role.access_monitoring_role[0].arn
      },
    ]
  })
}

// Auth server uses route53 to get certs for domain, this allows
// read/write operations from the zone.
resource "aws_iam_role_policy" "auth_route53" {
  name = "${var.cluster_name}-auth-route53"
  role = aws_iam_role.auth.id

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Id": "certbot-dns-route53 policy",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "route53:ListHostedZones",
                "route53:GetChange"
            ],
            "Resource": [
                "*"
            ]
        },
        {
            "Effect" : "Allow",
            "Action" : [
                "route53:ChangeResourceRecordSets"
            ],
            "Resource" : [
                "arn:aws:route53:::hostedzone/${data.aws_route53_zone.proxy.zone_id}"
            ]
        }
    ]
}
EOF

}

