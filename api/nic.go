package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const realm = "Dyn DNS"

// HandleNIC responds to /nic/update
func HandleNIC(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		log.Print("Invalid authorization:", auth)
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, realm))
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	log.Printf("Received request: %s , from: %s", r.URL, r.RemoteAddr)
	//prettyPrint(r.Header)
	if usr, psw, ok := r.BasicAuth(); ok {
		log.Printf("\n User: %s, Password: %s ", usr, psw)
	}

	w.WriteHeader(http.StatusOK)
	//w.Write([]byte("<html><body><h1> Thanks for all the fish! </h1></body></html> "))
}

func prettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}
