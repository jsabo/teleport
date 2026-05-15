#!/bin/bash
cat >/etc/teleport.d/conf <<EOF
TELEPORT_ROLE=auth
EC2_REGION=${region}
TELEPORT_AUTH_SERVER_LB=${auth_server_addr}
TELEPORT_AUTH_TYPE=${teleport_auth_type}
TELEPORT_CLUSTER_NAME=${cluster_name}
TELEPORT_DOMAIN_ADMIN_EMAIL=${email}
TELEPORT_DOMAIN_NAME=${domain_name}
TELEPORT_DYNAMO_TABLE_NAME=${dynamo_table_name}
TELEPORT_DYNAMO_EVENTS_TABLE_NAME=${dynamo_events_table_name}
TELEPORT_LICENSE_PATH=${license_path}
TELEPORT_LOCKS_TABLE_NAME=${locks_table_name}
TELEPORT_S3_BUCKET=${s3_bucket}
USE_ACM=${use_acm}
USE_TLS_ROUTING=${use_tls_routing}
%{ if enable_athena ~}
TELEPORT_ATHENA_EVENTS_URI="${athena_events_uri}"
TELEPORT_ACCESS_MONITORING_ROLE_ARN=${access_monitoring_role_arn}
TELEPORT_ACCESS_MONITORING_REPORT_RESULTS=${access_monitoring_report_results}
TELEPORT_ACCESS_MONITORING_WORKGROUP=${access_monitoring_workgroup}
TELEPORT_ATHENA_MIGRATION_MODE=${athena_migration_mode}
%{ endif ~}
EOF
