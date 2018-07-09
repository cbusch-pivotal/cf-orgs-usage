package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/jszwec/csvutil"
	"github.com/labstack/echo"
	"github.com/palantir/stacktrace"
	"github.com/parnurzeal/gorequest"
)

//AppUsage array of orgs usage
type AppUsage struct {
	Orgs []OrgAppUsage `json:"orgs" csv:"orgs"`
}

//OrgAppUsage Single org usage
type OrgAppUsage struct {
	OrganizationGUID string    `json:"organization_guid" csv:"organization_guid"`
	OrgName          string    `json:"organization_name" csv:"organization_name"`
	PeriodStart      time.Time `json:"period_start" csv:"period_start"`
	PeriodEnd        time.Time `json:"period_end" csv:"period_end"`
	AppUsages        []struct {
		SpaceGUID             string `json:"space_guid" csv:"space_guid"`
		SpaceName             string `json:"space_name" csv:"space_name"`
		AppName               string `json:"app_name" csv:"app_name"`
		AppGUID               string `json:"app_guid" csv:"app_guid"`
		InstanceCount         int    `json:"instance_count" csv:"instance_count"`
		MemoryInMbPerInstance int    `json:"memory_in_mb_per_instance" csv:"memory_in_mb_per_instance"`
		DurationInSeconds     int    `json:"duration_in_seconds" csv:"duration_in_seconds"`
	} `json:"app_usages" csv:"app_usages"`
}

// FlattenAppUsage flattened data for simple response with repeated org info
type FlattenAppUsage struct {
	Orgs []FlattenOrgAppUsage `json:"app_usages" csv:"app_usages"`
}

// FlattenOrgAppUsage flattened data for simple response usage
type FlattenOrgAppUsage struct {
	OrganizationGUID      string    `json:"organization_guid" csv:"organization_guid"`
	OrgName               string    `json:"organization_name" csv:"organization_name"`
	PeriodStart           time.Time `json:"period_start" csv:"period_start"`
	PeriodEnd             time.Time `json:"period_end" csv:"period_end"`
	SpaceGUID             string    `json:"space_guid" csv:"space_guid"`
	SpaceName             string    `json:"space_name" csv:"space_name"`
	AppName               string    `json:"app_name" csv:"app_name"`
	AppGUID               string    `json:"app_guid" csv:"app_guid"`
	InstanceCount         int       `json:"instance_count" csv:"instance_count"`
	MemoryInMbPerInstance int       `json:"memory_in_mb_per_instance" csv:"memory_in_mb_per_instance"`
	DurationInSeconds     int       `json:"duration_in_seconds" csv:"duration_in_seconds"`
}

// handles report formatting if CSV is specified
func appReportFormatter(c echo.Context, usageReport *FlattenAppUsage) error {
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

// AppUsageReportByRange handle a start and end date in the call
//  /app-usage?start=2017-11-01&end=2017-11-03
func AppUsageReportByRange(c echo.Context) error {

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
	usageReport, err := GenAppUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get app usage report for yesterday")
	}

	// return report
	return appReportFormatter(c, usageReport)
}

// AppUsageReportForToday handles the static nature of Apptio's Datalink
//  in order to gather app usage data for the previous day
func AppUsageReportForToday(c echo.Context) error {
	// format the date range
	dateToday := time.Now().Local()

	// format the start and end string
	dateRange := GenDateRange(dateToday, dateToday)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	usageReport, err := GenAppUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get app usage report for yesterday")
	}

	// return report
	return appReportFormatter(c, usageReport)
}

// AppUsageReportForYesterday handles the static nature of Apptio's Datalink
//  in order to gather app usage data for the previous day
func AppUsageReportForYesterday(c echo.Context) error {
	// format the date range
	dateToday := time.Now().Local()
	dateYesterday := dateToday.AddDate(0, 0, -1)

	// format the start and end string
	dateRange := GenDateRange(dateYesterday, dateYesterday)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	usageReport, err := GenAppUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get app usage report for yesterday")
	}

	// return report
	return appReportFormatter(c, usageReport)
}

// AppUsageReportForMonth handles the app-usage call validating the date
//  and executing the report creation
func AppUsageReportForMonth(c echo.Context) error {

	// first day of month and today's date
	dateToday := time.Now().Local()
	currentYear, currentMonth, _ := dateToday.Date()
	currentLocation := dateToday.Location()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// format the start and end string
	dateRange := GenDateRange(firstOfMonth, dateToday)
	fmt.Println("Date range is ", dateRange)

	// Generate the report for all orgs
	usageReport, err := GenAppUsageReport(cfClient, dateRange)
	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get app usage report for yesterday")
	}

	// return report
	return appReportFormatter(c, usageReport)
}

// GenAppUsageReport pulls the entire report together
func GenAppUsageReport(client *cfclient.Client, dateRange string) (*FlattenAppUsage, error) {

	// get a list of orgs within the foundation
	orgs, err := client.ListOrgs()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed getting list of orgs using client: %v", client)
	}

	report := AppUsage{}
	token, err := client.GetToken()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed getting token using client: %v", client)
	}

	// loop through orgs and get app usage report for each
	for _, org := range orgs {
		orgUsage, err := AppUsageForOrg(token, org, dateRange)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Failed getting app usage for org: "+org.Name)
		}
		orgUsage.OrgName = org.Name
		report.Orgs = append(report.Orgs, *orgUsage)
	}

	// flatten the complexity of report for ease of consumption
	flatReport, err := GetFlattenedAppOutput(&report)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Couldn't get app usage report")
	}

	return &flatReport, nil
}

// AppUsageForOrg queries apps manager app_usages API for the orgs app usage information
func AppUsageForOrg(token string, org cfclient.Org, dateRange string) (*OrgAppUsage, error) {
	usageAPI := os.Getenv("CF_USAGE_API")
	target := &OrgAppUsage{}
	request := gorequest.New()
	resp, _, err := request.Get(usageAPI+"/organizations/"+org.Guid+"/app_usages?"+dateRange).
		Set("Authorization", token).TLSClientConfig(&tls.Config{InsecureSkipVerify: cfSkipSsl}).
		EndStruct(&target)
	if err != nil {
		return nil, stacktrace.Propagate(err[0], "Failed to get app usage report %v", org)
	}

	if resp.StatusCode != 200 {
		return nil, stacktrace.NewError("Failed getting app usage report %v", resp)
	}

	return target, nil
}

// GetFlattenedAppOutput convert formatting to flattened output
func GetFlattenedAppOutput(usageReport *AppUsage) (FlattenAppUsage, error) {

	var flatUsage FlattenAppUsage

	for _, orgs := range usageReport.Orgs {
		for _, app := range orgs.AppUsages {
			appusage := FlattenOrgAppUsage{
				OrganizationGUID:      orgs.OrganizationGUID,
				OrgName:               orgs.OrgName,
				PeriodStart:           orgs.PeriodStart,
				PeriodEnd:             orgs.PeriodEnd,
				SpaceGUID:             app.SpaceGUID,
				SpaceName:             app.SpaceName,
				AppName:               app.AppName,
				AppGUID:               app.AppGUID,
				InstanceCount:         app.InstanceCount,
				MemoryInMbPerInstance: app.MemoryInMbPerInstance,
				DurationInSeconds:     app.DurationInSeconds,
			}
			flatUsage.Orgs = append(flatUsage.Orgs, appusage)
		}
	}
	return flatUsage, nil
}
