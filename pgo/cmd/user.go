package cmd

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/pgo/api"
	"github.com/crunchydata/postgres-operator/pgo/util"
	utiloperator "github.com/crunchydata/postgres-operator/util"

	log "github.com/sirupsen/logrus"
)

// userTextPadding contains the values for what the text padding should be
type userTextPadding struct {
	ClusterName  int
	ErrorMessage int
	Expires      int
	Password     int
	Username     int
	Status       int
}

// PasswordAgeDays password age flag
var PasswordAgeDays int

// Username is a postgres username
var Username string

// Expired expired flag
var Expired int

// PasswordLength password length flag
var PasswordLength int

// PasswordValidAlways allows a user to explicitly set that their passowrd
// is always valid (i.e. no expiration time)
var PasswordValidAlways bool

func createUser(args []string, ns string) {
	username := strings.TrimSpace(Username)

	// ensure the username is nonempty
	if username == "" {
		fmt.Println("Error: --username is required")
		os.Exit(1)
	}

	// check to see if this is a system account. if it is, do not let the request
	// go through
	if utiloperator.CheckPostgreSQLUserSystemAccount(username) {
		fmt.Println("Error:", username, "is a system account and cannot be used")
		os.Exit(1)
	}

	request := msgs.CreateUserRequest{
		AllFlag:         AllFlag,
		Clusters:        args,
		ManagedUser:     ManagedUser,
		Namespace:       ns,
		Password:        Password,
		PasswordAgeDays: PasswordAgeDays,
		PasswordLength:  PasswordLength,
		Username:        username,
		Selector:        Selector,
	}

	response, err := api.CreateUser(httpclient, &SessionCredentials, &request)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printCreateUserText(response)
	}
}

// deleteUser ...
func deleteUser(args []string, ns string) {

	log.Debugf("deleting user %s selector=%s args=%v", Username, Selector, args)

	if Username == "" {
		fmt.Println("Error: --username is required")
		return
	}

	r := new(msgs.DeleteUserRequest)
	r.Username = Username
	r.Clusters = args
	r.AllFlag = AllFlag
	r.Selector = Selector
	r.ClientVersion = msgs.PGO_VERSION
	r.Namespace = ns

	response, err := api.DeleteUser(httpclient, &SessionCredentials, r)

	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	if response.Status.Code == msgs.Ok {
		for _, result := range response.Results {
			fmt.Println(result)
		}
	} else {
		fmt.Println("Error: " + response.Status.Msg)
	}

}

// generateUserPadding returns the paddings based on the values of the response
func generateUserPadding(results []msgs.UserResponseDetail) userTextPadding {
	// make the interface for the users
	userInterface := makeUserInterface(results)

	// set up the text padding
	return userTextPadding{
		ClusterName:  getMaxLength(userInterface, headingCluster, "ClusterName"),
		ErrorMessage: getMaxLength(userInterface, headingErrorMessage, "ErrorMessage"),
		Expires:      getMaxLength(userInterface, headingExpires, "ValidUntil"),
		Password:     getMaxLength(userInterface, headingPassword, "Password"),
		Status:       len(headingStatus) + 1,
		Username:     getMaxLength(userInterface, headingUsername, "Username"),
	}
}

// makeUserInterface returns an interface slice of the avaialble values
// in pgo create user
func makeUserInterface(values []msgs.UserResponseDetail) []interface{} {
	// iterate through the list of values to make the interface
	userInterface := make([]interface{}, len(values))

	for i, value := range values {
		userInterface[i] = value
	}

	return userInterface
}

// printCreateUserText prints out the information that is created after
// pgo create user is called
func printCreateUserText(response msgs.CreateUserResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("No users created.")
		return
	}

	padding := generateUserPadding(response.Results)

	// print the header
	printUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUserTextRow(result, padding)
	}
}

// printUpdateUserText prints out the information from calling pgo update user
func printUpdateUserText(response msgs.UpdateUserResponse) {
	// if the request errored, return the message here and exit with an error
	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(1)
	}

	// if no results returned, return an error
	if len(response.Results) == 0 {
		fmt.Println("No users updated.")
		return
	}

	padding := generateUserPadding(response.Results)

	// print the header
	printUserTextHeader(padding)

	// iterate through the reuslts and print them out
	for _, result := range response.Results {
		printUserTextRow(result, padding)
	}
}

// printUserTextHeader prints out the header
func printUserTextHeader(padding userTextPadding) {
	// print the header
	fmt.Println("")
	fmt.Printf("%s", util.Rpad(headingCluster, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(headingUsername, " ", padding.Username))
	fmt.Printf("%s", util.Rpad(headingPassword, " ", padding.Password))
	fmt.Printf("%s", util.Rpad(headingExpires, " ", padding.Expires))
	fmt.Printf("%s", util.Rpad(headingStatus, " ", padding.Status))
	fmt.Printf("%s", util.Rpad(headingErrorMessage, " ", padding.ErrorMessage))
	fmt.Println("")

	// print the layer below the header...which prints out a bunch of "-" that's
	// 1 less than the padding value
	fmt.Println(
		strings.Repeat("-", padding.ClusterName-1),
		strings.Repeat("-", padding.Username-1),
		strings.Repeat("-", padding.Password-1),
		strings.Repeat("-", padding.Expires-1),
		strings.Repeat("-", padding.Status-1),
		strings.Repeat("-", padding.ErrorMessage-1),
	)
}

// printUserTextRow prints a row of the text data
func printUserTextRow(result msgs.UserResponseDetail, padding userTextPadding) {
	expires := result.ValidUntil

	// check for special values of expires, e.g. if the password matches special
	// values to indicate if it has expired or not
	switch {
	case expires == "" || expires == utiloperator.SQLValidUntilAlways:
		expires = "never"
	case expires == utiloperator.SQLValidUntilNever:
		expires = "expired"
	}

	password := result.Password

	// set the text-based status, and use it to drive some of the display
	status := "ok"

	if result.Error {
		expires = ""
		password = ""
		status = "error"
	}

	fmt.Printf("%s", util.Rpad(result.ClusterName, " ", padding.ClusterName))
	fmt.Printf("%s", util.Rpad(result.Username, " ", padding.Username))
	fmt.Printf("%s", util.Rpad(password, " ", padding.Password))
	fmt.Printf("%s", util.Rpad(expires, " ", padding.Expires))
	fmt.Printf("%s", util.Rpad(status, " ", padding.Status))
	fmt.Printf("%s", util.Rpad(result.ErrorMessage, " ", padding.ErrorMessage))
	fmt.Println("")
}

// showUsers ...
func showUser(args []string, ns string) {

	log.Debugf("showUser called %v", args)

	log.Debugf("selector is %s", Selector)
	if len(args) == 0 && Selector != "" {
		args = make([]string, 1)
		args[0] = "all"
	}

	r := msgs.ShowUserRequest{}
	r.Clusters = args
	r.ClientVersion = msgs.PGO_VERSION
	r.Selector = Selector
	r.Namespace = ns
	r.Expired = Expired
	r.AllFlag = AllFlag

	response, err := api.ShowUser(httpclient, &SessionCredentials, &r)
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(2)
	}

	if response.Status.Code != msgs.Ok {
		fmt.Println("Error: " + response.Status.Msg)
		os.Exit(2)
	}
	if len(response.Results) == 0 {
		fmt.Println("No clusters found.")
		return
	}

	if OutputFormat == "json" {
		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Println("Error: ", err)
		}
		fmt.Println(string(b))
		return
	}

	for _, clusterDetail := range response.Results {
		printUsers(&clusterDetail)
	}

}

// updateUser prepares the API call for updating attributes of a PostgreSQL
// user
func updateUser(clusterNames []string, namespace string) {
	// set up the reuqest
	request := msgs.UpdateUserRequest{
		AllFlag:             AllFlag,
		Clusters:            clusterNames,
		Expired:             Expired,
		ExpireUser:          ExpireUser,
		ManagedUser:         ManagedUser,
		Namespace:           namespace,
		Password:            Password,
		PasswordAgeDays:     PasswordAgeDays,
		PasswordLength:      PasswordLength,
		PasswordValidAlways: PasswordValidAlways,
		RotatePassword:      RotatePassword,
		Selector:            Selector,
		Username:            strings.TrimSpace(Username),
	}

	// check to see if EnableLogin or DisableLogin is set. If so, set a value
	// for the LoginState parameter
	if EnableLogin {
		request.LoginState = msgs.UpdateUserLoginEnable
	} else if DisableLogin {
		request.LoginState = msgs.UpdateUserLoginDisable
	}

	// check to see if this is a system account if a user name is passed in
	if request.Username != "" && utiloperator.CheckPostgreSQLUserSystemAccount(request.Username) {
		fmt.Println("Error:", request.Username, "is a system account and cannot be used")
		os.Exit(1)
	}

	response, err := api.UpdateUser(httpclient, &SessionCredentials, &request)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}

	// great! now we can work on interpreting the results and outputting them
	// per the user's desired output format
	// render the next bit based on the output type
	switch OutputFormat {
	case "json":
		printJSON(response)
	default:
		printUpdateUserText(response)
	}
}

// printUsers
// TODO: delete
func printUsers(detail *msgs.ShowUserDetail) {
	fmt.Println("")
	fmt.Println("cluster : " + detail.Cluster.Spec.Name)

	if detail.ExpiredOutput == false {
		for _, s := range detail.Secrets {
			fmt.Println("")
			fmt.Println("secret : " + s.Name)
			fmt.Println(TreeBranch + "username: " + s.Username)
			fmt.Println(TreeTrunk + "password: " + s.Password)
		}
	} else {
		fmt.Printf("\nuser passwords expiring within %d days:\n", detail.ExpiredDays)
		fmt.Println("")
		if len(detail.ExpiredMsgs) > 0 {
			for _, e := range detail.ExpiredMsgs {
				fmt.Println(e)
			}
		}
	}

}