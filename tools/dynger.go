package main

import (
	"crypto/rand"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"log"

	"golang.org/x/crypto/blake2b"
)

type HashHelper interface {
	ToHex(src []byte) ([]byte, error)
	ToB64(src []byte) string
	GetSum(src []byte) ([]byte, error)
}

type Blake2bDigest struct {
	hash hash.Hash
}

func NewB2BDigest(size int) (*Blake2bDigest, error) {
	h, err := blake2b.New(size, nil)
	if err != nil {
		return nil, fmt.Errorf("can't create Blake2B hash instance: \n%w", err)
	}
	return &Blake2bDigest{h}, nil
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
		return "", fmt.Errorf("Can't convert to hex, %w", err)
	}
	return b64.URLEncoding.EncodeToString(hb), nil
}

const (
	// size of randomly generated seed in bytes
	RandBytesLen = 32
	// size of Blake2b hash in bytes
	DigestSize = 32
)

func main() {
	fmt.Println("Generating dynger token...")

	var rb [RandBytesLen]byte

	_, err := rand.Read(rb[:])
	if err != nil {
		log.Fatal("Error encountered ..", err)
	}

	bd, err := NewB2BDigest(DigestSize)
	if err != nil {
		log.Fatal(err)
	}
	ts, err := bd.ToB64(rb[:])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Generated token (URLBase64): %s\n", ts)
}
