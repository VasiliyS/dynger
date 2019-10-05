package handler

import (
	"api/authlib"
	"api/common/utils"
	"api/zeitapi"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	dynMasterSecret = "DYN_MASTER_SECRET"
)

func init() {
	// setup logging parameters
	// TODO: set debug level from environment variable
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	//get secrets
	mSecret := os.Getenv(dynMasterSecret)
	if mSecret == "" {
		log.Error().Str(dynMasterSecret, "is empty").Msg("Couldn't initialize master secret ")
		authlib.InitAuth("no secret")
	}
	authlib.InitAuth(mSecret)
}

// HandleNIC responds to /nic/update
func HandleNIC(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		log.Warn().Str("header", auth).Msg("Invalid authorization Header")
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, realm))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	log.Debug().Str("method", r.Method).Str("url", r.URL.String()).Str("addr", r.RemoteAddr).Send()

	// Prepare response
	respB := bytes.NewBuffer(nil)
	// populate response with the current response on return
	defer func() {
		// Request handled correctly, report processing result in the body (Is this always correct?)
		w.WriteHeader(http.StatusOK)
		w.Write(respB.Bytes())
	}()

	//get URL Query parameters
	q := r.URL.Query()
	//check IP
	myip := q.Get(dynDNSIPParam)
	ip := net.ParseIP(myip)
	if ip == nil {
		log.Warn().Str("myip", myip).Str("remote IP", r.RemoteAddr).Msg("Couldn't parse 'myip' parameter, using remote ip")
		ip = net.ParseIP(r.RemoteAddr)
	}
	// TODO: log if new ip is not the same as the address of the sender
	domain := q.Get(dynDNSHostParam)
	if err := utils.CheckDomain(domain); err != nil {
		respB.WriteString(dynDNSStatusNotFQDN)
		log.Error().Err(err).Msg("Bad 'hostname' supplied")
		return
	}

	if usr, psw, ok := r.BasicAuth(); ok {

		// TODO: Handle an error here
		isVerified, _ := authlib.VerifyLogin(domain, usr, psw)
		log.Debug().Str("user", usr).Str("pswrd", psw).Str("dom", domain).Bool("ver", isVerified).Send()
		if !isVerified {
			// TODO: log the event
			respB.WriteString(dynDNSStatusBadAuth)
			return
		}
	} else {
		// TODO: log the event
		respB.WriteString(dynDNSStatusBadAuth)
		return
	}

	isNew, err := zeitapi.DNS.SetAddressTo(domain, ip)
	if err != nil {
		respB.WriteString(dynDNSStatus911)
		log.Error().Err(err).Msg("Address is not set!")
		return
	}
	if !isNew {
		log.Warn().Msg("Update request was using the existing address, request ignored.")
		// TODO: track timer to panish updates for the same value ??
		writeStringsToB(respB, dynDNSStatusNochng, ip.String())
		return
	}
	// Everything went well!
	writeStringsToB(respB, dynDNSStatusGoog, ip.String())
	log.Info().Str("hostname", domain).Str("myip", ip.String()).Msg("Update successful!")
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
