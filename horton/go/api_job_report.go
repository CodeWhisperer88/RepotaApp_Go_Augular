/*
 * John Shields
 * Horton - API version: 1.0.0
 *
 * API Job Report
 * Handles all Job Reports activities - Create, Update, Delete & Getting Reports.
 *
 * References
 * Setup Generated by: OpenAPI Generator (https://openapi-generator.tech)
 * Refer to https://johnshields.github.io/horton.api.doc/ for more info.
 * https://www.golangprograms.com/example-of-golang-crud-using-mysql-from-scratch.html
 * https://levelup.gitconnected.com/build-a-rest-api-using-go-mysql-gorm-and-mux-a02e9a2865ee
 * https://semaphoreci.com/community/tutorials/building-go-web-applications-and-microservices-using-gin
 */

package openapi

import (
	"errors"
	"fmt"
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/config"
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/models"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
)

// CreateReport
// Works with CheckForCookie & InsertJobReport.
// If the user has a cookie call InsertJobReport to create a report from user input data.
func CreateReport(c *gin.Context) {
	var report models.JobReport

	// Bind entered JobReport data from user to object, else throw error.
	if err := c.BindJSON(&report); err != nil {
		fmt.Println(err.Error()) // Failed to Bind data.
		c.JSON(500, nil)
	}

	// Check for user's cookie - if they do not have one abort the request.
	// Status code handled by CheckForCookie.
	if !CheckForCookie(c) {
		log.Println("User is unauthorized to create a report")
		return
	}

	// Call InsertJobReport to create the report.
	if err := InsertJobReport(c, report, wa.Username); err == nil {
		c.JSON(201, models.Error{Code: 201, Messages: "Report created successfully"})
	} else {
		c.JSON(401, models.Error{Code: 401, Messages: "Not able to create Report"})
	}
}

// InsertJobReport
// Function that creates a new report by starting and committing a MySQL transaction
// with data inputted by user to insert into the tables, jobreports and customers.
func InsertJobReport(c *gin.Context, report models.JobReport, username string) error {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	fmt.Println("\n[INFO] Processing Report Details...")

	// Insert into the table jobreports.
	insertReport, err := db.Prepare(
		"INSERT INTO jobreports(worker_id, date_stamp, vehicle_model, vehicle_reg, vehicle_location, " +
			"miles_on_vehicle, warranty, breakdown, cause, correction, parts, work_hours, job_report_complete) " +
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

	// Insert into the table customers.
	insertCustomer, err := db.Prepare("INSERT INTO customers (job_report_id, customer_name, customer_complaint)" +
		" VALUES (LAST_INSERT_ID(), ?, ?)")

	if err != nil {
		c.JSON(500, nil)
		log.Println("\nMySQL Error: Error Preparing new Report:\n", err)
		return errors.New("error creating Report")
	}

	// Check logged in user - mainly for selecting the user's 'Worker Name' to add it to the report.
	if !isValidAccount(username) {
		log.Println("\nUser is not logged in")
		c.JSON(401, models.Error{Code: 401, Messages: "User is not logged in"})
		return errors.New("error creating Report")
	}

	// Begin MySQL transition to create a new report with input data from user.
	_, err = db.Query("BEGIN")
	// Execute insert into the table jobreports.
	reportResult, err := insertReport.Exec(wa.Id, report.Date, report.VehicleModel, report.VehicleReg, report.VehicleLocation,
		report.MilesOnVehicle, report.Warranty, report.Breakdown, report.Cause, report.Correction, report.Parts,
		report.WorkHours, report.JobComplete)
	// Execute insert into the table customers.
	customerResult, err := insertCustomer.Exec(report.CustomerName, report.Complaint)
	_, err = db.Query("COMMIT") // Commit MySQL transition.

	if err != nil {
		log.Println("\nMySQL Error: Error Inserting Report Details.\n", err)
		c.JSON(500, nil)
		return errors.New("error creating Report")
	}
	fmt.Println("\n[INFO] Printing MySQL Results for new Report...\n", reportResult, customerResult)

	defer db.Close()
	return nil
}

// GetReportById
// Works with CheckForCookie & isValidAccount.
// If the user has a cookie and owns the report,
// get it in the database with a JOIN QUERY by its requested ID and logged in user's username.
func GetReportById(c *gin.Context) {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	var res []models.JobReport
	worker := wa.Username

	if !CheckForCookie(c) {
		log.Println("User is unauthorized to get this Report")
		return
	}

	// Get ID from request.
	reportId := c.Params.ByName("jobReportId")
	fmt.Printf("Get Report with ID: " + reportId)

	// Check the user (worker) to put it in the SELECT Query.
	if !isValidAccount(worker) {
		log.Println("\nUser does not own this report.")
		c.JSON(401, models.Error{Code: 401, Messages: "User does not own this report"})
		return
	}

	// JOIN Query to get report by requested ID and username.
	selDB, err := db.Query("SELECT DISTINCT jr.job_report_id, jr.date_stamp, jr.vehicle_model, "+
		"jr.vehicle_reg, jr.miles_on_vehicle, jr.vehicle_location, jr.warranty, jr.breakdown, "+
		"cust.customer_name, cust.customer_complaint, jr.cause, jr.correction, jr.parts, jr.work_hours, "+
		"wkr.worker_name, jr.job_report_complete FROM jobreports jr INNER JOIN customers cust "+
		"ON jr.job_report_id = cust.job_report_id "+
		"INNER JOIN workers wkr ON jr.worker_id = wkr.worker_id "+
		"WHERE jr.job_report_id = ? AND wkr.username = ?", reportId, worker)

	if err != nil {
		log.Println("\nFailed to process Report.", err)
		c.JSON(500, nil)
		return // Problem with QUERY.
	}

	// Run through each record and read values - Get the requested report from the database.
	for selDB.Next() {
		var report models.JobReport

		err = selDB.Scan(&report.JobReportId, &report.Date, &report.VehicleModel, &report.VehicleReg, &report.MilesOnVehicle,
			&report.VehicleLocation, &report.Warranty, &report.Breakdown, &report.CustomerName, &report.Complaint, &report.Cause,
			&report.Correction, &report.Parts, &report.WorkHours, &report.WorkerName, &report.JobComplete)

		if err != nil {
			log.Println("\nFailed to load Report.")
			c.JSON(500, nil)
		}
		// Add each record to array.
		res = append(res, report)
		log.Printf(string(report.JobReportId))
	}
	// Return result values - send the report object to client for user.
	c.JSON(http.StatusOK, res)
	fmt.Println("\n[INFO] Report by ID Processed...")
	defer db.Close()
}

// GetReports
// Works with CheckForCookie & isValidAccount.
// If the user has a cookie and owns the reports,
// get all the reports belonging to the user in the database from a JOIN QUERY by the logged in user's username.
func GetReports(c *gin.Context) {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	worker := wa.Username
	var res []models.JobReport
	var report models.JobReport

	if !CheckForCookie(c) {
		log.Println("User is unauthorized to get these Reports")
		return
	}

	if !isValidAccount(worker) {
		log.Println("\nUser does not own these reports.")
		c.JSON(401, models.Error{Code: 401, Messages: "User does not own these reports"})
		return
	}

	// JOIN Query to get user's job reports.
	selDB, err := db.Query("SELECT DISTINCT jr.job_report_id, jr.date_stamp, jr.vehicle_model, "+
		"jr.vehicle_reg, jr.miles_on_vehicle, jr.vehicle_location, jr.warranty, jr.breakdown, "+
		"cust.customer_name, cust.customer_complaint, jr.cause, jr.correction, jr.parts, jr.work_hours, "+
		"wkr.worker_name, jr.job_report_complete FROM jobreports jr INNER JOIN customers cust "+
		"ON jr.job_report_id = cust.job_report_id "+
		"INNER JOIN workers wkr ON jr.worker_id = wkr.worker_id WHERE wkr.username = ?", worker)

	if err != nil {
		log.Println("\nFailed to process Reports.")
		c.JSON(500, nil)
		return
	}

	// Run through each record and read values - get the user's reports.
	for selDB.Next() {
		err = selDB.Scan(&report.JobReportId, &report.Date, &report.VehicleModel, &report.VehicleReg, &report.MilesOnVehicle,
			&report.VehicleLocation, &report.Warranty, &report.Breakdown, &report.CustomerName, &report.Complaint, &report.Cause,
			&report.Correction, &report.Parts, &report.WorkHours, &report.WorkerName, &report.JobComplete)

		if err != nil {
			log.Println("\nFailed to load Reports.")
			c.JSON(500, nil)
		}
		// Add each record to array.
		res = append(res, report)
		log.Printf(string(report.JobReportId))
	}
	// Return result values - send the report objects to client for user.
	c.JSON(http.StatusOK, res)
	fmt.Println("\n[INFO] Reports Processed...")
	defer db.Close()
}

// UpdateReport
// Works with CheckForCookie.
// If the user has a cookie allow them to update/edit report in the database by its requested ID.
func UpdateReport(c *gin.Context) {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests
	var report models.JobReport

	// Get ID from request.
	reportId := c.Params.ByName("jobReportId")
	fmt.Printf("Get Report with ID: " + reportId)

	if !CheckForCookie(c) {
		log.Println("User is unauthorized to update this Report")
		return
	}

	// Bind JobReport data to object, else throw error.
	if err := c.BindJSON(&report); err != nil {
		fmt.Println(err.Error())
	}

	// Read in values from client request and build object - update the report with the user's inputted data.
	update, err := db.Exec("UPDATE jobreports jr SET jr.date_stamp = ?, jr.vehicle_model = ?, "+
		"jr.vehicle_reg = ?, jr.vehicle_location = ?, jr.miles_on_vehicle = ?, jr.warranty = ?, "+
		"jr.breakdown = ?, jr.cause = ?, jr.correction = ?, jr.parts = ?, jr.work_hours = ?, "+
		"jr.job_report_complete = ? WHERE jr.job_report_id = ?", report.Date, report.VehicleModel, report.VehicleReg,
		report.VehicleLocation, report.MilesOnVehicle, report.Warranty, report.Breakdown, report.Cause, report.Correction,
		report.Parts, report.WorkHours, report.JobComplete, reportId)

	if err != nil {
		log.Println("\nMySQL Error: Error Updating Report:\n", err)
		c.JSON(503, models.Error{Code: 503, Messages: "Error Updating Report"})
	} else {
		fmt.Println("\n[INFO] Processing Job Report Details...", "\nReport ID:", reportId)
		// Report has been successfully updated.
		c.JSON(202, gin.H{})
		fmt.Println("\n[INFO] Print MySQL Results for Report:\n", update)
		defer db.Close()
	}
}

// DeleteReport
// Works with CheckForCookie.
// If the user has a cookie allow them to delete a report in the database by its requested ID.
func DeleteReport(c *gin.Context) {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	// Get ID from request.
	reportId := c.Params.ByName("jobReportId")
	fmt.Printf("Get Report with ID: " + reportId)

	if !CheckForCookie(c) {
		log.Println("User is unauthorized to delete this Report")
		return
	}

	// Create query to delete the report with its requested ID.
	res, err := db.Exec("DELETE FROM jobreports WHERE job_report_id=?", reportId)
	if err != nil {
		log.Printf("Report failed to delete.")
		c.JSON(500, nil)
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Report failed to delete.")
		c.JSON(500, nil)
	}

	fmt.Printf("\nThe statement affected %d rows\n", affectedRows)
	c.JSON(204, nil) // Report has been deleted successfully.
}
