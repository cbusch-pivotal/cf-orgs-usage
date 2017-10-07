#!/bin/bash
set -e

# audit user information
AUDIT_USER="pcf-auditor"
AUDIT_PWD="auditor"
AUDIT_EMAIL="pcf@company.com"

# set target environment in which to create users
#uaac target uaa.system.<DOMAIN.COM> --skip-ssl-validation
uaac target uaa.system.mypcf.net --skip-ssl-validation

# Note: insert token after '-s' from Elastic Runtime tile -> Credentials tab -> UAA / Admin Client Credentials
# uaac token client get admin -s <UAA ADMIN CLIENT PASSWORD>
uaac token client get admin -s tohf0T95lFu_UBD3BWtUtK3G5bVVwpbA

# create audit user
uaac user add $AUDIT_USER -p $AUDIT_PWD --emails $AUDIT_EMAIL

# set if auditing specific orgs - and set OrgManager for $AUDIT_USER on each org
uaac group add usage_service.audit
uaac member add usage_service.audit $AUDIT_USER

# set if wanting system org and all orgs whether or not OrgManager role is set for $AUDIT_USER
uaac member add cloud_controller.admin $AUDIT_USER
