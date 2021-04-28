/*
 * John Shields
 * Horton - API version: 1.0.0
 *
 * API Account
 * Handles User Registration, Login and Logout.
 *
 * References
 * Setup Generated by: OpenAPI Generator (https://openapi-generator.tech)
 * Refer to https://johnshields.github.io/horton.api.doc/ for more info.
 * https://dev.to/jacobsngoodwin/creating-signup-handler-in-gin-binding-data-3kb5
 * https://www.sohamkamani.com/blog/2018/02/25/golang-password-authentication-and-storage/
 * https://www.programmersought.com/article/28644788179/
 */

package openapi

import (
	"errors"
	"fmt"
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/config"
	"github.com/GIT_USER_ID/GIT_REPO_ID/go/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"log"
	"strings"
)

var wa models.WorkerAccount

// Login
// Works with verifyDetails, removeSession & createSessionId.
// Logs in a user by comparing the entered password with the hashed password in the database,
// removes the existing session for the user, creates a new one and sets a cookie for the user.
func Login(c *gin.Context) {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	// Object to bind user data too.
	var workerForm models.WorkerAccount

	if err := c.BindJSON(&workerForm); err != nil {
		fmt.Println(err.Error())
	}

	username := workerForm.Username
	password := workerForm.Password

	// Check if user exists in the database and check password is not null.
	if err := verifyDetails(username, password); err != nil {
		c.JSON(403, models.Error{Code: 403, Messages: "Username does not exist"})
		return // Return as there is issues with the username or password.
	}

	// Compare the hash in the db with the user's password provided in the request using golang.org/x/crypto/bcrypt.
	if err := bcrypt.CompareHashAndPassword([]byte(wa.Password), []byte(password)); err == nil {

		// Check for existing session, remove if one exits.
		if removeSession(wa.Id) {
			// Create new session ID for user who logged in.
			err, session := createSessionId(username)

			if err != nil {
				log.Print(err)
				c.JSON(500, models.Error{Code: 500, Messages: "Unable to create new session"})
			} else {
				// Set a cookie for logged in user.
				//c.SetCookie("session_id", session.Token, session.Expiry, "/", "repota-service.com", true, false) // Hosting
				c.SetCookie("session_id", session.Token, session.Expiry, "/", "", false, false) // Local
				// User has been logged in and cookie has been set.
				c.JSON(204, nil)
			}
		} else {
			log.Println("Unable remove old session", err)
			c.JSON(500, models.Error{Code: 500, Messages: "Unable to remove old session"})
		}
	} else {
		log.Println("Password is incorrect for User", err)
		c.JSON(401, models.Error{Code: 401, Messages: "Password is incorrect"})
	}
	defer db.Close()
}

// Register
// Works with RegisterNewUser & createSessionId.
// Function to register a new user and creates a session ID for the user.
func Register(c *gin.Context) {
	var user models.InlineObject

	// Bind user's data to object, else throw error.
	if err := c.BindJSON(&user); err != nil {
		log.Println(err.Error())
		c.JSON(500, nil)
	}

	username := user.Username
	password := user.Password

	// Register new user and hash the password in RegisterNewUser.
	if err := RegisterNewUser(c, username, user.Name, password); err == nil {

		// Create a session for the new user.
		err, session := createSessionId(username)

		if err != nil {
			log.Print( "Failed to Register User.", err)
			c.JSON(500, nil)
		} else {
			c.JSON(200, session) // Session has been created for user.
		}
	} else {
		// Issue with user's details - username or name taken.
		log.Println("\nUnable to complete request")
	}
}

// RegisterNewUser
// Works with isValidAccount (Check if details already exist).
// Function that registers a new user and hashes the password.
// User gets registered and are inserted into the database.
func RegisterNewUser(c *gin.Context, username, name, password string) error {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	// Check if password is null or if username is taken.
	if strings.TrimSpace(password) == "" {
		log.Println("\nPassword is null")
		return errors.New("password is null")
	} else if isValidAccount(username) {
		c.JSON(409, models.Error{Code: 409, Messages: "Username is already taken"})
		log.Println("\nUsername taken")
		return errors.New("username is already taken")
	}

	// Hash the password here using golang.org/x/crypto/bcrypt.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		c.JSON(500, nil)
		log.Fatal("\nHash Password failed: ", err)
	}

	// Insert new user in the workers table.
	insert, err := db.Prepare("INSERT INTO workers(username, worker_name, hash) VALUES (?, ?, ?)")

	if err != nil {
		c.JSON(500, nil)
		log.Println("\nMySQL Error: Error Preparing new user account:\n", err)
	}

	result, err := insert.Exec(username, name, hashedPassword)

	// Return MySQL error if there is a duplicate entry.
	// Most likely the name as password and username have already been checked.
	if err != nil {
		c.JSON(409, models.Error{Code: 409, Messages: "Please make your name more unique"})
		log.Println("\nMySQL Error: Duplicate entry:\n", err)
		return err
	}

	fmt.Println("\n[INFO] Printing MySQL Results for user account...\n", result)

	defer db.Close()
	return nil
}

// Function to do a database look up and check if a username matches one provided.
// Mainly used to check if a user tries to register with a taken username.
// And for selecting users for report functions in API Job Report.
func isValidAccount(username string) bool {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests

	// Check username from workers table.
	selDB, err := db.Query("SELECT * FROM workers WHERE username=?", username)

	if err != nil {
		log.Fatal(err) // error with Query.
		return false
	}

	// Check to see if a true user exists in the table, if not return false.
	if selDB.Next() {
		err = selDB.Scan(&wa.Id, &wa.Username, &wa.WorkerName, &wa.Password)

		if err != nil {
			// No matching username in table (user does not exist).
			log.Println("\nMySQL Error - no matching username:\n", err)
			return false
		}
		defer db.Close()
		return true
	} else {
		defer db.Close()
		return false
	}
}

// Function to check password for null and if user exists when users login
// with the help of isValidAccount.
func verifyDetails(username, password string) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password is null")
	} else if !isValidAccount(username) {
		return errors.New("username does not exist")
	} else {
		return nil
	}
}

// Logout
// Works with removeSession & createSessionId to remove the user's current session.
// Then creates a new one that expires in one second.
// Set a new Cookie with the new session to logout the user after one second.
func Logout(c *gin.Context) {
	db := config.DbConn()
	//db := mocks.MockDbConn() // for unit tests
	username := wa.Username

	// Check for existing session, remove if one exits.
	if removeSession(wa.Id) {
		// Create new session ID for user who logged out.
		err, session := createSessionId(username)

		if err != nil {
			log.Println("Could not logout User", err)
			c.JSON(500, models.Error{Code: 500, Messages: "Unable to logout User"})
		} else {
			// Set a cookie of one second for logged out user.
			//c.SetCookie("session_id", session.Token, 1, "/", "repota-service.com", true, false) // Hosting
			c.SetCookie("session_id", session.Token, 1, "/", "", false, false) // Local
			c.JSON(204, models.Error{Code: 204, Messages: "User has been logged out"})
			fmt.Println("User has been logged out.")
		}
	} else {
		log.Println("Could not logout User")
		c.JSON(500, models.Error{Code: 500, Messages: "Unable to logout User"})
	}
	defer db.Close()
}
