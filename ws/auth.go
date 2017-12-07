package ws

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"phonebook/lib"
	"strings"
)

// AuthenticateData is the struct with the username and password
// used for authentication
type AuthenticateData struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

// SvcAuthenticate generates a password hash from the supplied POST info and
//     along with the user name compares it to what is in the database. If
//     there is a match, then the response is {status: success}.  If it fails
//     then the response is {status: failed}.  No indication will be given
//     indicating whether the username is not recognized or the password for
//     the supplied username is not correct.
//
// INPUTS:
//     w = file descriptor to write result
//     r = http requrest
//     d = pointer to data parsed by service dispatcher
//
// RETURNS:
//     nothing at this time
//-----------------------------------------------------------------------------
func SvcAuthenticate(w http.ResponseWriter, r *http.Request, d *ServiceData) {

	var funcname = "saveReceipt"
	var err error
	var foo AuthenticateData

	lib.Console("Entered %s\n", funcname)

	data := []byte(d.data)
	if err = json.Unmarshal(data, &foo); err != nil {
		e := fmt.Errorf("%s: Error with json.Unmarshal:  %s", funcname, err.Error())
		SvcErrorReturn(w, e, funcname)
		return
	}

	lib.Console("User = %s\n", foo.User)
	lib.Console("Pass = %s\n", foo.Pass)

	ok, err := DoAuthentication(foo.User, foo.Pass)
	if err != nil {
		SvcErrorReturn(w, err, funcname)
		return
	}
	if ok {
		SvcSuccessReturn(w)
	} else {
		err := fmt.Errorf("login failed")
		SvcErrorReturn(w, err, funcname)
	}
}

// DoAuthentication builds a password hash out of the supplied user and password
// information. It then looks up the user in the database. If the password hashes
// match, then the login is successful
//
// INPUTS:
//  User = username
//  Pass = user's password
//
// RETURNS:
//  bool =  true if the login was successful, false otherwise
func DoAuthentication(User, Pass string) (bool, error) {
	myusername := strings.ToLower(User)
	password := []byte(Pass)
	sha := sha512.Sum512(password)
	mypasshash := fmt.Sprintf("%x", sha)

	// lookup the user
	q := fmt.Sprintf("SELECT passhash FROM People WHERE UserName=%q", myusername)
	var passhash string
	err := SvcCtx.db.QueryRow(q).Scan(&passhash)
	if err != nil {
		return false, err
	}
	if passhash != mypasshash {
		err := fmt.Errorf("login failed")
		return false, err
	}
	return true, nil // login is successful
}