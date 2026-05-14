write_confd_file() {
    cat << EOF > ${TELEPORT_CONFD_DIR?}/conf
TELEPORT_ROLE=auth
EC2_REGION=us-east-1
TELEPORT_AUTH_TYPE=github
TELEPORT_AUTH_SERVER_LB=gus-tftestkube4-auth-0f66dd17f8dd9825.elb.us-east-1.amazonaws.com
TELEPORT_CLUSTER_NAME=gus-tftestkube4
TELEPORT_DOMAIN_ADMIN_EMAIL=test@email.com
TELEPORT_DOMAIN_NAME=gus-tftestkube4.gravitational.io
TELEPORT_DYNAMO_TABLE_NAME=gus-tftestkube4
TELEPORT_DYNAMO_EVENTS_TABLE_NAME=
TELEPORT_LICENSE_PATH=/home/gus/downloads/teleport/license-gus.pem
TELEPORT_LOCKS_TABLE_NAME=gus-tftestkube4-locks
TELEPORT_S3_BUCKET=gus-tftestkube4.gravitational.io
TELEPORT_ATHENA_EVENTS_URI="athena://gus_tftestkube4.audit_events?topicArn=arn:aws:sns:us-east-1:123456789012:gus-tftestkube4-audit-events&largeEventsS3=s3://gus-tftestkube4-teleport-audit-transient/large_payloads&locationS3=s3://gus-tftestkube4-teleport-audit-longterm/events&workgroup=gus-tftestkube4-audit&queueURL=https://sqs.us-east-1.amazonaws.com/123456789012/gus-tftestkube4-audit-events&queryResultsS3=s3://gus-tftestkube4-teleport-audit-transient/query_results&limiterBurst=450&limiterRefillAmount=450&limiterRefillTime=1s"
TELEPORT_ACCESS_MONITORING_ROLE_ARN=arn:aws:iam::123456789012:role/gus-tftestkube4-access-monitoring
TELEPORT_ACCESS_MONITORING_REPORT_RESULTS=s3://gus-tftestkube4-teleport-audit-longterm/report_results
TELEPORT_ACCESS_MONITORING_WORKGROUP=gus-tftestkube4-access-monitoring
USE_ACM=false
USE_TLS_ROUTING=false
EOF
}

load fixtures/common

@test "[${TEST_SUITE?}] config file was generated without error" {
    [ ${GENERATE_EXIT_CODE?} -eq 0 ]
}

@test "[${TEST_SUITE?}] teleport.storage.type is dynamodb (cluster state backend unchanged)" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${TELEPORT_BLOCK?}"
    echo "${TELEPORT_BLOCK?}" | grep -E "^    type: dynamodb"
}

@test "[${TEST_SUITE?}] teleport.storage.audit_events_uri uses Athena" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${TELEPORT_BLOCK?}"
    echo "${TELEPORT_BLOCK?}" | grep -E "^    audit_events_uri: athena://"
}

@test "[${TEST_SUITE?}] teleport.storage.audit_events_uri does not use DynamoDB" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${TELEPORT_BLOCK?}"
    ! echo "${TELEPORT_BLOCK?}" | grep -E "^    audit_events_uri: dynamodb://"
}

@test "[${TEST_SUITE?}] teleport.storage.audit_sessions_uri uses S3" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${TELEPORT_BLOCK?}"
    echo "${TELEPORT_BLOCK?}" | grep -E "^    audit_sessions_uri: s3://"
}

@test "[${TEST_SUITE?}] auth_service.access_monitoring.enabled is true" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -A4 "^  access_monitoring:" | grep -q "enabled: true"
}

@test "[${TEST_SUITE?}] auth_service.access_monitoring.role_arn is set" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -E "^    role_arn: ${TELEPORT_ACCESS_MONITORING_ROLE_ARN?}"
}

@test "[${TEST_SUITE?}] auth_service.access_monitoring.report_results is set" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -E "^    report_results: ${TELEPORT_ACCESS_MONITORING_REPORT_RESULTS?}"
}

@test "[${TEST_SUITE?}] auth_service.access_monitoring.workgroup is set" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -E "^    workgroup: ${TELEPORT_ACCESS_MONITORING_WORKGROUP?}"
}

@test "[${TEST_SUITE?}] auth_service.cluster_name is set correctly" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -E "^  cluster_name: ${TELEPORT_CLUSTER_NAME?}"
}

@test "[${TEST_SUITE?}] auth_service.proxy_protocol is on" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -E "^  proxy_protocol: on"
}

@test "[${TEST_SUITE?}] auth_service.license_file is set" {
    load ${TELEPORT_CONFD_DIR?}/conf
    echo "${AUTH_BLOCK?}"
    echo "${AUTH_BLOCK?}" | grep -E "^  license_file: "
}
