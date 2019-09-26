package handler

import (
	"fmt"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	//currentTime := time.Now().Format(time.RFC850)
	fmt.Fprintf(w, "%s \n", r.RemoteAddr)

}
