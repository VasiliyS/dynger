package crypto

import (
	"crypto/rand"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/blake2b"
)

//set of parameters used for crypto functions
const (
	// size of randomly generated seed in bytes
	RandBytesLen = 32
	// size of Blake2b hash in bytes
	DigestSize = 32
)

// Argon2Key helps to work with Argon2
type Argon2Key struct {
	key     []byte
	salt    []byte
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewArgon2(salt []byte) (*Argon2Key, error) {
	if salt == nil {
		salt = make([]byte, RandBytesLen)
		_, err := rand.Read(salt)
		if err != nil {
			return nil, err
		}
	}
	// set value recommended in the godoc:
	// The draft RFC recommends[2] time=1, and memory=64*1024 is a sensible number
	key := Argon2Key{
		salt:    salt,
		time:    1,
		memory:  64 * 1024,
		threads: uint8(runtime.NumCPU()),
		keyLen:  32,
	}
	return &key, nil
}

func (dk *Argon2Key) GenKey(password []byte) {
	dk.key = argon2.IDKey(password, dk.salt, dk.time, dk.memory, dk.threads, dk.keyLen)
}
func (dk *Argon2Key) Bytes() []byte {
	return dk.key
}

func (dk *Argon2Key) StringB64(enc *b64.Encoding) string {

	return enc.EncodeToString(dk.key)
}

// SaltedStringB64 will concatinate Salt and Key with delimiter, using Base64 encoding
func (dk *Argon2Key) SaltedStringB64(enc *b64.Encoding, delimiter byte) string {
	var b strings.Builder
	b.WriteString(enc.EncodeToString(dk.salt))
	b.WriteByte(delimiter)
	b.WriteString(enc.EncodeToString(dk.key))
	return b.String()
}

// -----------------
type Blake2bDigest struct {
	hash hash.Hash
	size int
}

func NewB2BDigest(size int, key []byte) (*Blake2bDigest, error) {
	h, err := blake2b.New(size, key)
	if err != nil {
		return nil, fmt.Errorf("can't create Blake2B hash instance: \n%w", err)
	}
	return &Blake2bDigest{h, size}, nil
}

func (d *Blake2bDigest) Size() int {
	return d.size
}

// GetSum appends the current hash to src and returns the resulting slice
func (d *Blake2bDigest) GetSum(src []byte) ([]byte, error) {

	_, err := d.hash.Write(src)
	if err != nil {
		return nil, fmt.Errorf("Error adding to Hash %w", err)
	}
	return d.hash.Sum(nil), nil
}

// ToHex appends the current hash to src and returns the resulting hex ecoded string
func (d *Blake2bDigest) ToHex(src []byte) ([]byte, error) {

	h, err := d.GetSum(src)
	if err != nil {
		return nil, fmt.Errorf("Error finalizing hash: \n%w", err)
	}
	hb := make([]byte, hex.EncodedLen(len(h)))
	hex.Encode(hb[:], h[:])

	return hb, nil
}

// ToB64 appends the current hash to src and returns the resulting Base64 ecoded string
func (d *Blake2bDigest) ToB64(enc *b64.Encoding, src []byte) (string, error) {

	hb, err := d.GetSum(src)
	if err != nil {
		return "", fmt.Errorf("Can't get digest, %w", err)
	}
	return enc.EncodeToString(hb), nil
}
