package server

import (
	"fmt"
	"net/http"
)

func PrintReq(r *http.Request) {
	fmt.Println(r.Method + " " + r.Host + " " + r.RequestURI)
	// for key := range r.Header {
	// 	vals := r.Header[key]
	// 	if len(vals) == 0 {
	// 		fmt.Println(key + ":")
	// 	} else {
	// 		for _, val := range vals {
	// 			fmt.Println(key + ": " + val)
	// 		}
	// 	}
	// }
}
