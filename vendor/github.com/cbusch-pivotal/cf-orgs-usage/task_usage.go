package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/labstack/echo"
	"github.com/palantir/stacktrace"
	"github.com/parnurzeal/gorequest"
)

// TaskUsage array of orgs usage
type TaskUsage struct {
	Orgs []OrgTaskUsage `json:"orgs"`
}

// OrgTaskUsage Single org usage
type OrgTaskUsage struct {
	OrganizationGUID string    `json:"organization_guid"`
	OrgName          string    `json:"organization_name"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	Spaces           []struct {
		SpaceGUID []struct {
			SpaceName     string `json:"space_name"`
			TaskSummaries []struct {
				ParentApplicationGUID              string `json:"parent_application_guid"`
				ParentApplicationName              string `json:"parent_application_name"`
				MemoryInMbPerInstance              int    `json:"memory_in_mb_per_instance"`
				TaskCountForRange                  int    `json:"task_count_for_range"`
				TotalDurationInSecondsForRange     int    `json:"total_duration_in_seconds_for_range"`
				MaxConcurrentTaskCountForParentApp int    `json:"max_concurrent_task_count_for_parent_app"`
			} `json:"task_summaries"`
		} `json:"space_guid"`
	} `json:"spaces"`
}

// TaskUsageReport handles the app-usage call validating the date
//  and executing the report creation
func TaskUsageReport(c echo.Context) error {
	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		return stacktrace.Propagate(err, "couldn't convert year to number")
	}
	month, err := strconv.Atoi(c.Param("month"))
	if err != nil {
		return stacktrace.Propagate(err, "couldn't convert month to number")
	}

	usageReport, err := GetTaskUsageReport(cfClient, year, month)

	if err != nil {
		return stacktrace.Propagate(err, "Couldn't get task usage report")
	}
	return c.JSON(http.StatusOK, usageReport)
}

// GetTaskUsageReport pulls the entire report together
func GetTaskUsageReport(client *cfclient.Client, year int, month int) (*TaskUsage, error) {
	if !(month >= 1 && month <= 12) {
		return nil, stacktrace.NewError("Month must be between 1-12")
	}

	// get a list of orgs within the foundation
	orgs, err := client.ListOrgs()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed getting list of orgs using client: %v", client)
	}

	report := TaskUsage{}
	token, err := client.GetToken()
	if err != nil {
		return nil, stacktrace.Propagate(err, "Failed getting token using client: %v", client)
	}

	// loop through orgs and get app usage report for each
	for _, org := range orgs {
		orgUsage, err := GetTaskUsageForOrg(token, org, year, month)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Failed getting task usage for org: "+org.Name)
		}
		orgUsage.OrgName = org.Name
		report.Orgs = append(report.Orgs, *orgUsage)
	}

	return &report, nil
}

// GetTaskUsageForOrg queries apps manager app_usages API for the orgs app usage information
func GetTaskUsageForOrg(token string, org cfclient.Org, year int, month int) (*OrgTaskUsage, error) {
	usageAPI := os.Getenv("CF_USAGE_API")
	target := &OrgTaskUsage{}
	request := gorequest.New()

	fmt.Println(usageAPI + "/organizations/" + org.Guid + "/task_usages?" + GenTimeParams(year, month))

	resp, _, err := request.Get(usageAPI+"/organizations/"+org.Guid+"/task_usages?"+GenTimeParams(year, month)).
		Set("Authorization", token).TLSClientConfig(&tls.Config{InsecureSkipVerify: cfSkipSsl}).
		EndStruct(&target)
	if err != nil {
		return nil, stacktrace.Propagate(err[0], "Failed to get task usage report %v", org)
	}

	if resp.StatusCode != 200 {
		return nil, stacktrace.NewError("Failed getting task usage report %v", resp)
	}
	return target, nil
}
