package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

const (

	// CredsDelimiter is the character used to combine user and domain into a login string
	CredsDelimiter = ':'
)

var encoding = base64.RawURLEncoding

type MasterSecret []byte

type DNSRecPassword []byte

// MasterSecret Methods

func GenNewMasterSecret(pin string) (MasterSecret, error) {
	var bPin []byte

	if pin == "" {
		bPin := make([]byte, RandBytesLen)
		_, err := rand.Read(bPin)
		if err != nil {
			return nil, err
		}
	} else {
		bPin = []byte(pin)
	}
	key, err := NewArgon2(nil)
	if err != nil {
		return nil, err
	}
	key.GenKey(bPin)
	return key.Bytes(), nil
}

func InitMasterSecretWith(secret string) (MasterSecret, error) {

	return encoding.DecodeString(secret)
}

func (s MasterSecret) Bytes() []byte {
	return s
}

func (s MasterSecret) GetBase64() string {
	return encoding.EncodeToString(s)
}

// DNSRecPassword Methods

func NewDNSRecPassword(user, domain string) DNSRecPassword {
	return credsToBytes(user, domain)
}

func (p DNSRecPassword) Bytes() []byte {
	return p
}

func (p DNSRecPassword) GetBase64() string {
	return encoding.EncodeToString(p)
}

// ToHex returns hex encoded slice
func (p DNSRecPassword) ToHex() []byte {
	hb := make([]byte, hex.EncodedLen(len(p)))
	hex.Encode(hb, p)

	return hb
}

// credsToBytes constructs login string (user+CredDelimiter+domain) and returns it as slice
func credsToBytes(user, domain string) []byte {
	buf := bytes.NewBufferString(user)
	// TODO: handle potential memory errors here
	buf.WriteByte(CredsDelimiter)
	buf.WriteString(domain)
	return buf.Bytes()
}

func (p DNSRecPassword) VerifyPassword(password string, ms MasterSecret) (bool, error) {
	//decode passwrd to bytes
	passB, err := encoding.DecodeString(password)
	if err != nil {
		return false, err
	}
	// init Blake2B digest
	b2bh, err := NewB2BDigest(DigestSize, ms.Bytes())
	if err != nil {
		return false, err
	}
	// derive reference password
	refPassword, err := b2bh.GetSum(p)
	if err != nil {
		return false, err
	}
	// compare using constant time
	res := subtle.ConstantTimeCompare(refPassword, passB) == 1

	return res, nil
}
