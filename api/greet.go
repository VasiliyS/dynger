package handler

import (
	"fmt"
	"log"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	//currentTime := time.Now().Format(time.RFC850)
	fmt.Fprintf(w, "%s \n", r.RemoteAddr)
	log.Printf("paged opened by: %q\n", r.RemoteAddr)

}
