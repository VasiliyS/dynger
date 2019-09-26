package zeitapi

import "os"

var (
	dynPassword string
	usersDB     dynUsers
)

const dynSecret = "DYN_SECRET"

// map of domains  to the user/password used to authenticate
// the authorized user
type dynUsers map[string]struct {
	name     string
	password string
}

func init() {

	//prepare for multiple users in the future
	dynPassword = os.Getenv(dynSecret)
	usersDB = dynUsers{"denkruum.ch": {"vvv", dynPassword}}
}

// VerifyLogin returns true if user/password match the domain record
func VerifyLogin(domain, user, password string) bool {
	rec, ok := usersDB[domain]
	if !ok {
		return false
	}
	if rec.name == user && rec.password == password {
		return true
	}
	return false
}
