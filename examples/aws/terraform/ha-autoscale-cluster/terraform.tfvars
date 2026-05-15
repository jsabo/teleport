region          = "us-east-2"
cluster_name    = "sabo-athena-test"
ami_name        = "teleport-ent-18.8.0-arm64*"
key_name        = "sabo-us-east-2"
route53_zone    = "teleportdemo.com"
route53_domain  = "sabo-athena-test.teleportdemo.com"
s3_bucket_name  = "sabo-athena-test"
email           = "jonathan.sabo@goteleport.com"
license_path    = "/Users/sabo/Downloads/license.pem"
use_acm            = false
use_tls_routing    = false
enable_athena      = true
teleport_auth_type = "local"
add_wildcard_route53_record = true

default_tags = {
  "teleport.dev/creator" = "jonathan.sabo@goteleport.com"
  "env"                  = "demo"
  "team"                 = "sales-eng"
}
