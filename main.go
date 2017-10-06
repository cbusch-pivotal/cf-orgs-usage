package main

import (
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

// Main start point for the app
func main() {
	// save environment variables
	cfAPI = os.Getenv("CF_API")
	cfSkipSsl = os.Getenv("CF_SKIP_SSL_VALIDATION") == "true"
	cfUser = os.Getenv("CF_ADMIN_USER")
	cfPassword = os.Getenv("CF_ADMIN_PASSWORD")
	userBasic := os.Getenv("BASIC_USERNAME")
	passwordBasic := os.Getenv("BASIC_PASSWORD")

	// make sure no env variable is empty
	if userBasic == "" || passwordBasic == "" {
		log.Fatalf("Must set environment variables BASIC_USERNAME and BASIC_PASSWORD")
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

	// register ../xxx-usage/YYYY/MM endpoints
	e.GET("/app-usage/:year/:month", AppUsageReport)
	e.GET("/service-usage/:year/:month", ServiceUsageReport)
	//e.GET("/task-usage/:year/:month", TaskUsageReport)

	// confirm basic auth
	e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == userBasic && password == passwordBasic {
			return true, nil
		}
		return false, nil
	}))
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
	formatString := "2006-01-02"
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return "start=" + firstDay.Format(formatString) + "&end=" + lastDay.Format(formatString)
}
