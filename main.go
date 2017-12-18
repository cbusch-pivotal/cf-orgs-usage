package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/palantir/stacktrace"
)

// global variables
var cfClient *cfclient.Client
var cfAPI string
var cfUser string
var cfPassword string
var cfSkipSsl bool
var enableBasicAuth bool
var dateFormat = "2006-01-02"

// Main start point for the app
func main() {
	// save environment variables
	cfAPI = os.Getenv("CF_API")
	cfSkipSsl = os.Getenv("CF_SKIP_SSL_VALIDATION") == "true"
	cfUser = os.Getenv("CF_ADMIN_USER")
	cfPassword = os.Getenv("CF_ADMIN_PASSWORD")
	userBasic := os.Getenv("BASIC_USERNAME")
	passwordBasic := os.Getenv("BASIC_PASSWORD")
	enableBasicAuth := os.Getenv("ENABLE_BASIC_AUTH") == "true"

	// make sure no env variable is empty
	if enableBasicAuth == true {
		fmt.Print("enableBasicAuth = true")
		if userBasic == "" || passwordBasic == "" {
			log.Fatalf("Must set environment variables BASIC_USERNAME and BASIC_PASSWORD")
		}
	} else {
		fmt.Print("enableBasicAuth = false")
	}
	if cfAPI == "" || os.Getenv("CF_USAGE_API") == "" {
		log.Fatalf("Must set environment variables CF_API and CF_USAGE_API")
	}
	if cfUser == "" || cfPassword == "" {
		log.Fatalf("Must set environment variables CF_ADMIN_USER and CF_ADMIN_PASSWORD")
		return
	}

	// log into PCF when the app starts - if the apptio auditor user changes,
	//   make sure the restart the app
	_, err := SetupCfClient()
	if err != nil {
		log.Fatalf("Error setting up client %v", err)
		return
	}

	// create a router
	e := echo.New()

	// app-usage endpoints
	e.GET("/app-usage", AppUsageReportByRange)
	e.GET("/app-usage/today", AppUsageReportForToday)
	e.GET("/app-usage/yesterday", AppUsageReportForYesterday)
	e.GET("/app-usage/thismonth", AppUsageReportForMonth)
	//e.GET("/app-usage/:year/:month", AppUsageReportForMonth)

	// service-usage endpoints
	e.GET("/service-usage", ServiceUsageReportByRange)
	e.GET("/service-usage/today", ServiceUsageReportForToday)
	e.GET("/service-usage/yesterday", ServiceUsageReportForYesterday)
	e.GET("/service-usage/:year/:month", ServiceUsageReport)

	// task-usage endpoints (need to meet with Pivotal engineers to fix issue)
	//e.GET("/task-usage/:year/:month", TaskUsageReport)

	// confirm basic auth
	if enableBasicAuth == true {
		fmt.Print("Using basic auth for user validation")
		e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
			if username == userBasic && password == passwordBasic {
				return true, nil
			}
			return false, nil
		}))
	}
	e.Logger.Fatal(e.Start(":8080"))
}

// SetupCfClient logs the Apptio Auditor user into PCF
func SetupCfClient() (*cfclient.Client, error) {

	// setup the login data
	c := &cfclient.Config{
		ApiAddress:        cfAPI,
		Username:          cfUser,
		Password:          cfPassword,
		SkipSslValidation: cfSkipSsl,
	}

	// login
	client, err := cfclient.NewClient(c)
	if err != nil {
		return nil, stacktrace.Propagate(err, "Error creating cf client")
	}
	cfClient = client
	return client, nil
}

// GenTimeParams generates the from and to dates for the app_usages call to apps manager
func GenTimeParams(year int, month int) string {
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return "start=" + firstDay.Format(dateFormat) + "&end=" + lastDay.Format(dateFormat)
}

// GenDateRange generates the from and to dates for the app_usages call to apps manager
func GenDateRange(start time.Time, end time.Time) string {
	return "start=" + start.Format(dateFormat) + "&end=" + end.Format(dateFormat)
}
