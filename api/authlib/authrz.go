package authlib

import (
	"api/common/crypto"
)

var (
	dynKey crypto.MasterSecret
	//usersDB     dynUsers
)

// InitAuth sets master secret for the crypto functions
func InitAuth(secret string) {
	dynKey, _ = crypto.InitMasterSecretWith(secret)
	// !TODO: handle errors here
}

// VerifyLogin returns true if user/password match the domain record
func VerifyLogin(domain, user, password string) (bool, error) {

	refPassword := crypto.NewDNSRecPassword(user, domain)
	return refPassword.VerifyPassword(password, dynKey)
}
