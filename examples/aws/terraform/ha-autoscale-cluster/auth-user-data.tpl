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
%{ endif ~}
EOF
%{ if enable_athena ~}
cat >/usr/local/bin/teleport-patch-athena-config <<'PATCHSCRIPT'
#!/bin/bash
set -e
source /etc/teleport.d/conf
[[ -z "$TELEPORT_ATHENA_EVENTS_URI" ]] && exit 0
CONFIG=/etc/teleport.yaml
python3 -c "
import sys
path, uri = sys.argv[1], sys.argv[2]
with open(path) as f: lines = f.readlines()
lines = ['    audit_events_uri: ' + uri + '\n' if '    audit_events_uri: dynamodb://' in l else l for l in lines]
with open(path, 'w') as f: f.writelines(lines)
" "$CONFIG" "$TELEPORT_ATHENA_EVENTS_URI"
if ! grep -q "access_monitoring:" "$CONFIG"; then
cat >> "$CONFIG" <<ACFG
  access_monitoring:
    enabled: true
    role_arn: $TELEPORT_ACCESS_MONITORING_ROLE_ARN
    report_results: $TELEPORT_ACCESS_MONITORING_REPORT_RESULTS
    workgroup: $TELEPORT_ACCESS_MONITORING_WORKGROUP
ACFG
fi
PATCHSCRIPT
chmod +x /usr/local/bin/teleport-patch-athena-config
cat >/etc/systemd/system/teleport-patch-athena-config.service <<'SVCUNIT'
[Unit]
Description=Teleport Athena Config Patch
After=teleport-generate-config.service
Before=teleport-auth.service
[Service]
User=root
Type=oneshot
ExecStart=/usr/local/bin/teleport-patch-athena-config
[Install]
WantedBy=multi-user.target
SVCUNIT
systemctl daemon-reload
systemctl enable teleport-patch-athena-config.service
%{ endif ~}
