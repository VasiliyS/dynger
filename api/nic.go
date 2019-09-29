package handler

import (
	"api/zeitapi"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	realm                = "Dyn DNS"
	dynDNSStatusGoog     = "good"
	dynDNSStatusNochng   = "nochg"
	dynDNSStatusNoHost   = "nohost"
	dynDNSStatusBadAuth  = "badauth"
	dynDNSStatusNotFQDN  = "notfqdn"
	dynDNSStatusBadAgent = "badagent"
	dynDNSStatusAbuse    = "abuse"
	dynDNSStatus911      = "911"

	dynDNSHostParam = "hostname"
	dynDNSIPParam   = "myip"
)

// HandleNIC responds to /nic/update
func HandleNIC(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		log.Printf("Invalid authorization Header: %q", auth)
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, realm))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	log.Printf("Received request: %q , from: %q", r.URL, r.RemoteAddr)

	// Prepare response
	respB := bytes.NewBuffer(nil)
	// populate response with the current response on return
	defer func() {
		// Request handled correctly, report processing result in the body (Is this always correct?)
		w.WriteHeader(http.StatusOK)
		w.Write(respB.Bytes())
	}()

	if usr, psw, ok := r.BasicAuth(); ok {
		// TODO: verify login info
		log.Printf("\n User: %s, Password: %s ", usr, psw)
	} else {
		// TODO: log the event
		respB.WriteString(dynDNSStatusBadAuth)
		return
	}
	//get URL Query parameters
	q := r.URL.Query()
	//check IP
	ip := net.ParseIP(q[dynDNSIPParam][0])
	if ip == nil {
		// TODO: Log this
		ip = net.ParseIP(r.RemoteAddr)
	}
	// TODO: log if new ip is not the same as the address of the sender
	domain := q[dynDNSHostParam][0]
	if err := checkDomain(domain); err != nil {
		respB.WriteString(dynDNSStatusNotFQDN)
		// TODO: log the error
		return
	}
	isNew, err := zeitapi.DNS.SetAddressTo(domain, ip)
	if err != nil {
		respB.WriteString(dynDNSStatus911)
		// TODO: log error
		return
	}
	if !isNew {
		// TODO: log warning
		// TODO: track timer to panish updates for the same value ??
		writeStringsToB(respB, dynDNSStatusNochng, ip.String())
		return
	}
	// Everything went well!
	writeStringsToB(respB, dynDNSStatusGoog, ip.String())
	// TODO: log event
}

// ------- helper functions --------

// write strings to buffer and separate with a space
func writeStringsToB(b *bytes.Buffer, ss ...string) {
	for i, s := range ss {
		b.WriteString(s)
		if i < len(ss)-1 {
			b.WriteByte(' ')
		}
	}
}

func prettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}

// checkDomain returns an error if the domain name is not valid
// See https://tools.ietf.org/html/rfc1034#section-3.5 and
// https://tools.ietf.org/html/rfc1123#section-2.
// taken from https://gist.github.com/chmike/d4126a3247a6d9a70922fc0e8b4f4013
func checkDomain(name string) error {
	switch {
	case len(name) == 0:
		return nil // an empty domain name will result in a cookie without a domain restriction
	case len(name) > 255:
		return fmt.Errorf(" domain: name length is %d, can't exceed 255", len(name))
	}
	var l int // start of a label in the  domain name, '.'+1
	for i := 0; i < len(name); i++ {
		b := name[i]
		if b == '.' {
			// check domain labels validity
			switch {
			case i == l:
				return fmt.Errorf("domain: invalid character '%c' at offset %d: label can't begin with a period", b, i)
			case i-l > 63:
				return fmt.Errorf("domain: byte length of label '%s' is %d, can't exceed 63", name[l:i], i-l)
			case name[l] == '-':
				return fmt.Errorf("domain: label '%s' at offset %d begins with a hyphen", name[l:i], l)
			case name[i-1] == '-':
				return fmt.Errorf("domain: label '%s' at offset %d ends with a hyphen", name[l:i], l)
			}
			l = i + 1
			continue
		}
		// test label character validity, note: tests are ordered by decreasing validity frequency
		if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b >= 'A' && b <= 'Z') {
			// show the printable unicode character starting at byte offset i
			c, _ := utf8.DecodeRuneInString(name[i:])
			if c == utf8.RuneError {
				return fmt.Errorf("domain: invalid rune at offset %d", i)
			}
			return fmt.Errorf("domain: invalid character '%c' at offset %d", c, i)
		}
	}
	// check top level domain validity
	switch {
	case l == len(name):
		return fmt.Errorf("cdomain: missing top level domain, domain can't end with a period")
	case len(name)-l > 63:
		return fmt.Errorf("domain: byte length of top level domain '%s' is %d, can't exceed 63", name[l:], len(name)-l)
	case name[l] == '-':
		return fmt.Errorf("domain: top level domain '%s' at offset %d begins with a hyphen", name[l:], l)
	case name[len(name)-1] == '-':
		return fmt.Errorf("domain: top level domain '%s' at offset %d ends with a hyphen", name[l:], l)
	case name[l] >= '0' && name[l] <= '9':
		return fmt.Errorf("domain: top level domain '%s' at offset %d begins with a digit", name[l:], l)
	}
	return nil
}
