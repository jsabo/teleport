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
%{ if enable_athena ~}
cat >/usr/local/bin/teleport-patch-athena-config <<'PATCHSCRIPT'
#!/bin/bash
set -e
source $${TELEPORT_CONFD_DIR:-/etc/teleport.d}/conf
[[ -z "$TELEPORT_ATHENA_EVENTS_URI" ]] && exit 0
CONFIG=$${TELEPORT_CONFIG_PATH:-/etc/teleport.yaml}
while IFS= read -r line; do
  if [[ "$line" == "    audit_events_uri: dynamodb://"* ]]; then
    DYNAMO_URI="$${line#    audit_events_uri: }"
    case "$TELEPORT_ATHENA_MIGRATION_MODE" in
      dual_dynamo_primary)
        printf "    audit_events_uri: ['%s', '%s']\n" "$DYNAMO_URI" "$TELEPORT_ATHENA_EVENTS_URI" ;;
      dual_athena_primary)
        printf "    audit_events_uri: ['%s', '%s']\n" "$TELEPORT_ATHENA_EVENTS_URI" "$DYNAMO_URI" ;;
      *)
        printf "    audit_events_uri: '%s'\n" "$TELEPORT_ATHENA_EVENTS_URI" ;;
    esac
  else
    printf '%s\n' "$line"
  fi
done < "$CONFIG" > "$CONFIG.tmp" && mv "$CONFIG.tmp" "$CONFIG"
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
