package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/jszwec/csvutil"
	"github.com/labstack/echo"
	"github.com/palantir/stacktrace"
	"github.com/parnurzeal/gorequest"
)

// ServiceUsage array of orgs usage
type ServiceUsage struct {
	Orgs []OrgServiceUsage `json:"orgs" csv:"orgs"`
}

// OrgServiceUsage Single org usage
type OrgServiceUsage struct {
	OrganizationGUID string    `json:"organization_guid" csv:"organization_guid"`
	OrgName          string    `json:"organization_name" csv:"organization_name"`
	PeriodStart      time.Time `json:"period_start" csv:"period_start"`
	PeriodEnd        time.Time `json:"period_end" csv:"period_end"`
	ServiceUsages    []struct {
		Deleted                 bool      `json:"deleted" csv:"deleted"`
		DurationInSeconds       float32   `json:"duration_in_seconds" csv:"duration_in_seconds"`
		SpaceGUID               string    `json:"space_guid" csv:"space_guid"`
		SpaceName               string    `json:"space_name" csv:"space_name"`
		ServiceInstanceGUID     string    `json:"service_instance_guid" csv:"service_instance_guid"`
		ServiceInstanceName     string    `json:"service_instance_name" csv:"service_instance_name"`
		ServiceInstanceType     string    `json:"service_instance_type" csv:"service_instance_type"`
		ServicePlanGUID         string    `json:"service_plan_guid" csv:"service_plan_guid"`
		ServicePlanName         string    `json:"service_plan_name" csv:"service_plan_name"`
		ServiceName             string    `json:"service_name" csv:"service_name"`
		ServiceGUID             string    `json:"service_guid" csv:"service_guid"`
		ServiceInstanceCreation time.Time `json:"service_instance_creation" csv:"service_instance_creation"`
		ServiceInstanceDeletion time.Time `json:"service_instance_deletion" csv:"service_instance_deletion"`
	} `json:"service_usages" csv:"service_usages"`
}

// FlattenServiceUsage flattened data for simple response with repeated org info
type FlattenServiceUsage struct {
	Orgs []FlattenOrgServiceUsage `json:"service_usages" csv:"service_usages"`
}

// FlattenOrgServiceUsage flattened data for simple response usage
type FlattenOrgServiceUsage struct {
	OrganizationGUID        string    `json:"organization_guid" csv:"organization_guid"`
	OrgName                 string    `json:"organization_name" csv:"organization_name"`
	PeriodStart             time.Time `json:"period_start" csv:"period_start"`
	PeriodEnd               time.Time `json:"period_end" csv:"period_end"`
	Deleted                 bool      `json:"deleted" csv:"deleted"`
	DurationInSeconds       float32   `json:"duration_in_seconds" csv:"duration_in_seconds"`
	SpaceGUID               string    `json:"space_guid" csv:"space_guid"`
	SpaceName               string    `json:"space_name" csv:"space_name"`
	ServiceInstanceGUID     string    `json:"service_instance_guid" csv:"service_instance_guid"`
	ServiceInstanceName     string    `json:"service_instance_name" csv:"service_instance_name"`
	ServiceInstanceType     string    `json:"service_instance_type" csv:"service_instance_type"`
	ServicePlanGUID         string    `json:"service_plan_guid" csv:"service_plan_guid"`
	ServicePlanName         string    `json:"service_plan_name" csv:"service_plan_name"`
	ServiceName             string    `json:"service_name" csv:"service_name"`
	ServiceGUID             string    `json:"service_guid" csv:"service_guid"`
	ServiceInstanceCreation time.Time `json:"service_instance_creation" csv:"service_instance_creation"`
	ServiceInstanceDeletion time.Time `json:"service_instance_deletion" csv:"service_instance_deletion"`
}

// handles report formatting if CSV is specified
func serviceReportFormatter(c echo.Context, usageReport *FlattenServiceUsage) error {
	var format = strings.ToLower(c.QueryParam("format"))
	if format == "csv" {
		fmt.Println("csv output requested")
		b, err := csvutil.Marshal(usageReport.Orgs)
		if err != nil {
			fmt.Println("error:", err)
		}
		return c.String(http.StatusOK, string(b))
	} else {
		return c.JSON(http.StatusOK, usageReport)
	}
}

// ServiceUsageReportByRange handle a start and end date in the call
//  /service-usage?start=2017-11-01&end=2017-11-03
func ServiceUsageReportByRange(c echo.Context) error {

	// format the date range
	fmt.Println("Start date is '" + c.QueryParam("start") + "'")
	start, err := time.Parse(dateFormat, c.QueryParam("start"))
	if err != nil {
		return stacktrace.Propagate(err, "Improper start date provided in the URL")
	}
	end, err := time.Parse(dateFormat, c.QueryParam("end"))
	if err != nil {
		return stacktrace.Propagate(err, "Improper end date provided in the URL")
	}

	// format the start and end string
	dateRange := GenDateRange(start, end)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	flatUsage, err := GetServiceUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't service service usage report for yesterday")
	}

	// return report
	return serviceReportFormatter(c, flatUsage)
}

// ServiceUsageReportForToday handles the static nature of Apptio's Datalink
//  in order to gather service usage data for the previous day
func ServiceUsageReportForToday(c echo.Context) error {
	// format the date range
	dateToday := time.Now().Local()

	// format the start and end string
	dateRange := GenDateRange(dateToday, dateToday)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	flatUsage, err := GetServiceUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get service usage report for yesterday")
	}

	// return report
	return serviceReportFormatter(c, flatUsage)
}

// ServiceUsageReportForYesterday handles the static nature of Apptio's Datalink
//  in order to gather service usage data for the previous day
func ServiceUsageReportForYesterday(c echo.Context) error {
	// format the date range
	dateToday := time.Now().Local()
	dateYesterday := dateToday.AddDate(0, 0, -1)

	// format the start and end string
	dateRange := GenDateRange(dateYesterday, dateYesterday)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	flatUsage, err := GetServiceUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get service usage report for yesterday")
	}

	// return report
	return serviceReportFormatter(c, flatUsage)
}

// ServiceUsageReportForMonth handles the service-usage call validating the date
//  and executing the report creation
func ServiceUsageReportForMonth(c echo.Context) error {

	// first day of month and today's date
	dateToday := time.Now().Local()
	currentYear, currentMonth, _ := dateToday.Date()
	currentLocation := dateToday.Location()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// format the start and end string
	dateRange := GenDateRange(firstOfMonth, dateToday)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	flatUsage, err := GetServiceUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get service usage report for yesterday")
	}

	// return report
	return serviceReportFormatter(c, flatUsage)
}

// GetServiceUsageReport pulls the entire report together
func GetServiceUsageReport(client *cfclient.Client, dateRange string) (*FlattenServiceUsage, error) {

	// get a list of orgs within the foundation
	orgs, err := client.ListOrgs()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed getting list of orgs using client: %v", client)
	}

	report := ServiceUsage{}
	token, err := client.GetToken()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed getting token using client: %v", client)
	}

	// loop through orgs and get service usage report for each
	for _, org := range orgs {
		orgUsage, err := GetServiceUsageForOrg(token, org, dateRange)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Failed getting service usage for org: "+org.Name)
		}
		orgUsage.OrgName = org.Name
		report.Orgs = append(report.Orgs, *orgUsage)
	}

	flatServiceReport, err := GetFlattenedServiceOutput(&report)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Couldn't get service usage report")
	}

	return &flatServiceReport, nil
}

// GetServiceUsageForOrg queries apps manager service_usages API for the orgs service usage information
func GetServiceUsageForOrg(token string, org cfclient.Org, dateRange string) (*OrgServiceUsage, error) {
	usageAPI := os.Getenv("CF_USAGE_API")
	target := &OrgServiceUsage{}
	request := gorequest.New()
	resp, _, err := request.Get(usageAPI+"/organizations/"+org.Guid+"/service_usages?"+dateRange).
		Set("Authorization", token).TLSClientConfig(&tls.Config{InsecureSkipVerify: cfSkipSsl}).
		EndStruct(&target)
	if err != nil {
		return nil, stacktrace.Propagate(err[0], "Failed to get service usage report %v", org)
	}

	if resp.StatusCode != 200 {
		return nil, stacktrace.NewError("Failed getting service usage report %v", resp)
	}
	return target, nil
}

//GetFlattenedServiceOutput convert formatting to flattened output
func GetFlattenedServiceOutput(usageReport *ServiceUsage) (FlattenServiceUsage, error) {

	var flatUsage FlattenServiceUsage

	for _, orgs := range usageReport.Orgs {
		for _, service := range orgs.ServiceUsages {
			serviceusage := FlattenOrgServiceUsage{
				OrganizationGUID:        orgs.OrganizationGUID,
				OrgName:                 orgs.OrgName,
				PeriodStart:             orgs.PeriodStart,
				PeriodEnd:               orgs.PeriodEnd,
				Deleted:                 service.Deleted,
				DurationInSeconds:       service.DurationInSeconds,
				SpaceGUID:               service.SpaceGUID,
				SpaceName:               service.SpaceName,
				ServiceInstanceGUID:     service.ServiceInstanceGUID,
				ServiceInstanceName:     service.ServiceInstanceName,
				ServiceInstanceType:     service.ServiceInstanceType,
				ServicePlanGUID:         service.ServicePlanGUID,
				ServicePlanName:         service.ServicePlanName,
				ServiceName:             service.ServiceName,
				ServiceGUID:             service.ServiceGUID,
				ServiceInstanceCreation: service.ServiceInstanceCreation,
				ServiceInstanceDeletion: service.ServiceInstanceDeletion,
			}
			flatUsage.Orgs = append(flatUsage.Orgs, serviceusage)
		}
	}
	return flatUsage, nil
}
