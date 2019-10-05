package main

import (
	"crypto/rand"
	b64 "encoding/base64"
	"encoding/hex"
	stdFlag "flag"
	"fmt"
	"hash"
	"log"
	"os"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"

	"golang.org/x/crypto/blake2b"
)

type bytePrinterHelper interface {
	ToHex(src []byte) ([]byte, error)
	ToB64(src []byte) string
}

// dyngerKey helps to work with Argon2
type dyngerKey struct {
	key     []byte
	salt    []byte
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewArgon2(salt []byte) (*dyngerKey, error) {
	if salt == nil {
		salt = make([]byte, RandBytesLen)
		_, err := rand.Read(salt)
		if err != nil {
			return nil, err
		}
	}
	// set value recommended in the godoc:
	// The draft RFC recommends[2] time=1, and memory=64*1024 is a sensible number
	key := dyngerKey{
		salt:    salt,
		time:    1,
		memory:  64 * 1024,
		threads: uint8(runtime.NumCPU()),
		keyLen:  32,
	}
	return &key, nil
}

func (dk *dyngerKey) GenKey(password []byte) {
	dk.key = argon2.IDKey(password, dk.salt, dk.time, dk.memory, dk.threads, dk.keyLen)
}
func (dk *dyngerKey) Bytes() []byte {
	return dk.key
}

func (dk *dyngerKey) StringB64(enc *b64.Encoding) string {

	return enc.EncodeToString(dk.key)
}

// SaltedStringB64 will concatinate Salt and Key with delimiter, using Base64 encoding
func (dk *dyngerKey) SaltedStringB64(enc *b64.Encoding, delimiter byte) string {
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

func (d *Blake2bDigest) GetSum(src []byte) ([]byte, error) {

	_, err := d.hash.Write(src)
	if err != nil {
		return nil, fmt.Errorf("Error adding to Hash %w", err)
	}
	return d.hash.Sum(nil), nil
}

func (d *Blake2bDigest) ToHex(src []byte) ([]byte, error) {

	h, err := d.GetSum(src)
	if err != nil {
		return nil, fmt.Errorf("Error finalizing hash: \n%w", err)
	}
	hb := make([]byte, hex.EncodedLen(len(h)))
	hex.Encode(hb[:], h[:])

	return hb, nil
}

func (d *Blake2bDigest) ToB64(src []byte) (string, error) {

	hb, err := d.GetSum(src)
	if err != nil {
		return "", fmt.Errorf("Can't get digest, %w", err)
	}
	return b64.URLEncoding.EncodeToString(hb), nil
}

const (
	// size of randomly generated seed in bytes
	RandBytesLen = 32
	// size of Blake2b hash in bytes
	DigestSize = 32
)

func init() {
	flag = stdFlag.NewFlagSet("Dynger v0.0.1", stdFlag.ExitOnError)

}

var flag *stdFlag.FlagSet

func main() {

	sPin := flag.String("pin", "", "Please specify `password`. Random PIN will be used otherwise.")
	sCred := flag.String("cred", "", "Please specify `user:domain` to create a password. Requires PIN ")
	flag.Parse(os.Args[1:])

	fmt.Println("Generating dynger key...")
	var bPin []byte //will store bytes of the enetered or generated PIN

	if *sPin == "" {
		rPin := make([]byte, RandBytesLen)
		_, err := rand.Read(tPin)
		if err != nil {
			log.Fatal("Couldn't generate random PIN!")
		}
		bPin = rPin
		fmt.Printf("Using randomly generated PIN: %#x", rPin)
	} else {
		fmt.Println("... with PIN: ", *sPin)
		bPin = []byte(*sPin)
	}

	key, err := NewArgon2(nil)
	if err != nil {
		log.Fatalf("Couldn't create Argon2 key : %w ", err)
	}
	key.GenKey(bPin)
	fmt.Printf("\nGenerated master key (URLBase64): %s\n", key.StringB64(b64.RawURLEncoding))
	fmt.Printf("\nGenerated master key with salt  (URLBase64): %s\n", key.SaltedStringB64(b64.RawURLEncoding, '$'))

	if *sPin != "" && *sCred != "" {
		password, _ := NewB2BDigest(DigestSize, key.Bytes())
		sepInd := strings.IndexByte(*sCred, ':')
		if sepInd == -1 || sepInd == len(*sCred)-1 || sepInd == 0 {
			log.Fatalf("-cred's parameter should be 'user:domain', received: %q", *sCred)
		}
		b64P, _ := password.ToB64([]byte(*sCred))
		usr := (*sCred)[:sepInd]
		domain := (*sCred)[sepInd+1:]
		fmt.Printf("\nURL Encoded Password for User %q and Domain %q is : %s", usr, domain, b64P)
	}
}
