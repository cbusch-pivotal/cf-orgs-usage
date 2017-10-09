# cf-orgs-usage
Service deployed to a cloud foundry foundation to easily provide app, service, or task usage reports for all orgs and spaces within the foundation.

##  Setup
### Auditor User
Auditor user is necessary for the application (service) to log into the necessary app_usage endpoint, part of Apps Manager. From the command line UAA utility, uaac, create the following user. The Auditor user must be created in each PCF foundation to be audited, i.e. SandPaaS, DevPaaS, STLPaaS, KSCPaaS, and EUROPaaS. Cloud PaaS’s will need added once they come on line. An email should be setup for the user, but is not needed.

### Audit user information
```
AUDIT_USER="pcf-auditor"
AUDIT_PWD="auditor"
AUDIT_EMAIL="pcf@customer.com"
```
### UAAC Script
Set target environment in which to create users.
```
uaac target uaa.<foundation-system-domain> --skip-ssl-validation
```

Admin client must be authenticated. Acquire the “Admin Client” credentials from “Elastic Runtime tile -> Credentials tab -> UAA / Admin Client Credentials”.
```
uaac token client get admin -s <UAA ADMIN CLIENT PASSWORD>
```

Create the auditor user.
```
uaac user add $AUDIT_USER -p $AUDIT_PWD --emails $AUDIT_EMAIL
```

Add the `usage_service.audit` group, which is evaluated for audit access on the auditor user by the `app-usage.<sysdomain>` service used by `cf-orgs-usage` services.
```
uaac group add usage_service.audit
```

Next, give the auditor the proper scopes (group) `usage_service.audit`.
```
uaac member add usage_service.audit $AUDIT_USER
```

If the auditor is going to only access specific orgs, then set the auditor as `OrgManager` for each org.
```
cf set-org-role $AUDIT_USER <ORGANIZATION> OrgManager
```

For the auditor to access the `system` org and to retrieve reports from all orgs whether or not the auditor has been added as an `OrgManager` to each org, the `cloud_controller.admin` scope must be set.
```
uaac member add cloud_controller.admin $AUDIT_USER
```

## Org and Space for Service
Since this is a system related app, it should be pushed into the system org. As a user with system administrator privileges, create an `usage-audit` space. This will be the location to which the application will be “pushed” later in this document.

## Audit Usage Service (the app)
The Audit Usage Service application was written by the Pivotal Cloud Foundry Services team specifically for customers. It is written in golang making it fast and easy to update.

This service returns the application usage information for all apps, in all spaces of all orgs with the foundation for a specific month to date. For example, calling the service on August 23, 2017 with the value URL `http://cf-orgs-usage.<app-domain>/app-usage/2017/08`, which is August 2017, will provide all app information for August 1st through August 23rd at the time it was called. Apps Manager updates monthly information roughly each hour of the day.

Audit usage performs roughly the following function, adding to the normal output of the Apps Manager app_usage endpoint.

1. At startup, the app logs into PCF foundation as the Auditor user
2. When called, the app checks basic authentication of the caller
3. Date value is validated.
4. A list of organizations is determined for the foundation.
5. Iterates for each organization retrieving app usage data for all spaces from the Apps Manager app-usage endpoint.
6. Adds the organization_name to the JSON since Apps Manager’s output does not.
7. Appends information for the organization to the foundation report
8. Returns the completed foundation report in JSON format to the caller.

## Service Configuration

### About manifest.yml
Change the `<SYSTEM-DOMAIN>` per the foundation in which the app is being deployed in the `CF_USAGE_API` and `CF_API` environment variables. These two variable could be set from within a pipeline script and removed from the `manifest.yml` to make them easier to change per foundation.

Change the `CF_ADMIN_USER` and `CF_ADMIN_PASSWORD` to make the Auditor user credentials set above.

Finally, the `BASIC_USERNAME` and `BASIC_PASSWORD` variables should be changed to provide basic user authentication from its consumer call, e.g. call to the service. For example: `http://basic:basic@cf-orgs-usage.apps.mypcf.net/app-usage/2017/10`

### File contents for manifest.yml
```
applications:
- name: cf-orgs-usage
  buildpack: go_buildpack
env:
  CF_USAGE_API: https://app-usage.<SYSTEM-DOMAIN>
  CF_API: https://api.<SYSTEM-DOMAIN>
  CF_SKIP_SSL_VALIDATION: true
  CF_ADMIN_USER: pcf-auditor
  CF_ADMIN_PASSWORD: auditor
  BASIC_USERNAME: basic
  BASIC_PASSWORD: basic
  GOPACKAGENAME: github.com/cbusch-pivotal/cf-orgs-usage
```

## Service Installation
### Build
There is no need to build the go project prior to pushing to Cloud Foundry. The go_buildpack will build the go executable as a Linux executable with all needed dependencies, i.e. `GOOS=linux GOARCH=amd64 go build`

### Push
Push the executable to PCF with the following command while logged into PCF as a system administrator capable of adding applications to the system org, `usage-audit` space.

`cf push`

### Testing
To test if the service is installed correctly, run the following `curl` commands.

__app-usage__
```
curl http://basic:basic@cf-orgs-usage.apps.mypcf.net/app-usage/2017/10 > app-usage.json
```

To further verify the service output, the following command can be run for each org in the foundation and compared. First log in as a user who can access audit information in each org.
```
curl "https://app-usage.system.mypcf.net/organizations/`cf org <ORG_NAME> --guid`/app_usages?start=2017-10-01&end=2017-10-31" -k -v -H "authorization: `cf oauth-token`" > app_usages.json
```

__service-usage__
```
curl http://basic:basic@cf-orgs-usage.apps.mypcf.net/service-usage/2017/10 > service-usage.json
```

To further verify the service output, the following command can be run for each org in the foundation and compared. First log in as a user who can access audit information in each org.
```
curl "https://app-usage.system.mypcf.net/organizations/`cf org <ORG_NAME> \
--guid`/service_usages?start=2017-10-01&end=2017-10-31" -k -v -H \
"authorization: `cf oauth-token`" > service_usages.json
```

__task-usage__
```
curl http://basic:basic@cf-orgs-usage.apps.mypcf.net/task-usage/2017/10 > task-usage.json
```

To further verify the service output, the following command can be run for each org in the foundation and compared. First log in as a user who can access audit information in each org.
```
curl "https://app-usage.system.mypcf.net/organizations/`cf org <ORG_NAME> \
--guid`/task_usages?start=2017-10-01&end=2017-10-31" -k -v -H \
"authorization: `cf oauth-token`" > task_usages.json
```