#!/bin/bash
set -e

# audit user information
AUDIT_USER="pcf-auditor"
AUDIT_PWD="auditor"
AUDIT_EMAIL="pcf@company.com"

# System Domain from PAS tile settings
SYSTEM_DOMAIN="system.pcf.example.com"

# From Elastic Runtime tile -> Credentials tab -> UAA / Admin Client Credentials
ADMIN_CLIENT_SECRET="

# set target environment in which to create users
uaac target uaa.${SYSTEM_DOMAIN}  --skip-ssl-validation

uaac token client get admin -s ${ADMIN_CLIENT_SECRET}

# create audit user
uaac user add $AUDIT_USER -p $AUDIT_PWD --emails $AUDIT_EMAIL

# set if auditing specific orgs - and set OrgManager for $AUDIT_USER on each org
uaac group add usage_service.audit
uaac member add usage_service.audit $AUDIT_USER

# set if wanting system org and all orgs whether or not OrgManager role is set for $AUDIT_USER
uaac member add cloud_controller.admin $AUDIT_USER
