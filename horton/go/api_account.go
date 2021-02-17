/*
 * John Shields
 * Horton
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 *
 * Account API
 * Handles Register, Login & Generates Session Cookies
 *
 *
 * References
 * https://dev.to/jacobsngoodwin/creating-signup-handler-in-gin-binding-data-3kb5
 * https://godoc.org/github.com/kimiazhu/ginweb-contrib/sessions
 * https://www.sohamkamani.com/blog/2018/02/25/golang-password-authentication-and-storage/
 */

package openapi

import (
	"errors"
	"fmt"
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/config"
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"strings"
)

var wa models.WorkerAccount

// Login - Log in
// Endpoint - http://horton.eu-west-1.elasticbeanstalk.com/api/v1/login
func Login(c *gin.Context) {
	db := config.DbConn()

	// Object to bind data too
	var workerForm models.WorkerAccount

	if err := c.BindJSON(&workerForm); err != nil {
		fmt.Println(err.Error())
	}

	username := workerForm.Username
	password := workerForm.Password

	// Check if user exist in the database and check password is not null
	if err := verifyDetails(username, password); err != nil {
		fmt.Println("[ALERT] Credentials Error:", err)
		c.JSON(400, models.Error{Code: 400, Messages: "Credentials are incorrect"})
		return // Return as there is issues with the credentials
	}

	// Compare the hash in the db with the user password provided in the request
	if err := bcrypt.CompareHashAndPassword([]byte(wa.Password), []byte(password)); err == nil {
		fmt.Println("[INFO] User logged in.")

		// Check for existing session, remove if one exits
		if removeSession(wa.Id) {
			// Create new session_id for user who logged in
			err, session := createSessionId(username)

			if err != nil {
				fmt.Print(err)
				c.JSON(500, models.Error{Code: 500, Messages: "Could not create new session_id"})
			} else {
				// set a cookie for logged in user
				c.SetCookie("session_id", session.Token, session.Expiry, "/",
					"127.0.0.1", false, true)
				c.JSON(200, nil)
				CheckForCookie(c)
			}
		} else {
			fmt.Println("\n[ALERT] Could not remove old session")
			c.JSON(500, models.Error{Code: 500, Messages: "Could not remove old session"})
		}
	} else {
		fmt.Print(err)
		log.Println("\n[ALERT] User has not logged in!")
		c.JSON(401, models.Error{Code: 401, Messages: "User has not logged in!"})
	}
	defer db.Close()
}

// Register - Registers User
// Endpoint - http://horton.eu-west-1.elasticbeanstalk.com/api/v1/register
func Register(c *gin.Context) {
	var user models.InlineObject

	// Blind data to object, else throw error
	if err := c.BindJSON(&user); err != nil {
		fmt.Println(err.Error())
	}

	username := user.Username
	password := user.Password

	// register new user and hash the password
	if err := registerNewUser(username, user.Name, password); err == nil {

		// create a session for the user
		err, session := createSessionId(username)

		if err != nil {
			fmt.Print(err)
			c.JSON(500, models.Error{Code: 500, Messages: "Internal Server Error"})
		} else {
			c.JSON(200, session)
		}
	} else {
		log.Printf("\n[ALERT] Not completing request")
	}
}

// Function to create a session id for authenticated user.
// Session tables is updated with the session token(UUID) and expiry time of three days and that is tied to the user by
// by the users id.
// Returns either an error or a new Session object containing session token and expiry time.
func createSessionId(username string) (error, models.Session) {
	db := config.DbConn()

	// User has been created now set the following below
	token := generateSessionId() // Create a new session ID
	expiry := 3600 * 24 * 3      // 3600 * 24 * 3 = 3 days

	// INSERT QUERY to create an Account
	insert, err := db.Prepare("INSERT INTO session(id, user, expire_after) VALUES(?, ?, ?)")

	if err != nil {
		fmt.Println(err.Error())
		return errors.New(err.Error()), models.Session{}
	}

	fmt.Println("\n[INFO] Printing Worker Account details:", "\nSession Token:", token, "\nWorker ID:", wa.Id,
		"\nExpiry time in seconds:", expiry)

	// Check if user account exists
	if !isValidAccount(username) {
		log.Println("\n[ALERT] User has not logged in!", err)
	}

	// Execute query to db, handle errors if any
	if _, err = insert.Exec(token, wa.Id, expiry); err != nil {
		log.Println("[ALERT] MYSQL Error: Error creating new session record\n", err)
		defer db.Close()
		return errors.New("[ALERT] MYSQL Error: Error creating new session record"), models.Session{}
	} else {
		fmt.Println("\n[INFO] Session has been generated. Records:", "\nSession Token:", token, "\nWorker ID:", wa.Id,
			"\nExpiry time in seconds:", expiry)
		fmt.Println("\n[INFO] Worker Username:", username)
		defer db.Close()

		// Returns Session object
		return nil, models.Session{Token: token, Expiry: expiry}
	}
}

// Function to create a session ID using UUID for an authenticated user.
// This session id will be needed to allow the user to make requests from the client to the server.
func generateSessionId() string {
	return uuid.New().String()
}

// Function that registers a new user to the database.
func registerNewUser(username, name, password string) error {
	db := config.DbConn()

	fmt.Println("\n[INFO] Processing User Details...",
		"\nEntered username:", username, "\nEntered Password:", password)

	if strings.TrimSpace(password) == "" {
		log.Printf("\n[ALERT] password is null")
		return errors.New("password is null")
	} else if isValidAccount(username) {
		log.Printf("\n[ALERT] username taken")
		return errors.New("username is already taken")
	}

	// Hash the password here
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// TODO - return responses for each if
	if err != nil {
		log.Fatal("\n[ALERT] Hash Password failed: ", err)
	}

	insert, err := db.Prepare("INSERT INTO workers(username, worker_name, hash) VALUES (?, ?, ?)")

	if err != nil {
		log.Println("\n[ALERT] MySQL Error: Error Creating new user account:\n", err)
	}

	result, err := insert.Exec(username, name, hashedPassword)

	if err != nil {
		log.Println("\n[ALERT] MySQL Error: Error Creating new user account:\n", err)
	}

	fmt.Println("\n[INFO] Print MySQL Results for user account:\n", result)

	defer db.Close()
	// Everything is good
	return nil
}

// Function to do a database look up and check if a username matches one provided.
func isValidAccount(username string) bool {
	db := config.DbConn()

	selDB, err := db.Query("SELECT * FROM workers WHERE username=?", username)

	if err != nil {
		log.Fatal(err)
		//return false
	}

	if selDB.Next() {
		err = selDB.Scan(&wa.Id, &wa.Username, &wa.WorkerName, &wa.Password)

		if err != nil {
			// return false // No user matching username provided
			log.Println("\n[ALERT] MySQL Error - no matching username:\n", err)
			//return false
		}
		defer db.Close()
		// Username matches return false as its not valid
		return true
	} else {
		// no true exits
		defer db.Close()
		return false
	}
}

func removeSession(userId int) bool {
	db := config.DbConn()

	//Create query
	res, err := db.Exec("DELETE FROM session WHERE user=?", userId)

	if err != nil {
		fmt.Printf("Query error")
		return false
	}

	affectedRows, err := res.RowsAffected()

	if err != nil {
		// return false
		fmt.Printf("\n[ALERT] Error updating record for deleting session")
		return false
	}

	fmt.Printf("The statement affected %d rows\n", affectedRows)
	return true
}

// Function that registers a new user to the database.
// Check password for null
// Check if user exists
func verifyDetails(username, password string) error {

	fmt.Println("\n[INFO] Processing User Details...",
		"\nEntered username:", username, "\nEntered Password:", password)

	if strings.TrimSpace(password) == "" {
		log.Printf("\n[ALERT] password is null")
		return errors.New("password is null")
	} else if !isValidAccount(username) {
		log.Printf("\n[ALERT] unknown username")
		return errors.New("username does not exist")
	} else {
		return nil
	}
}
