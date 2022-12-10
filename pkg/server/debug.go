package server

import (
	"mosi-docker-repo/pkg/logging"
	"net/http"
)

func PrintReq(r *http.Request) {
	logging.Debug(LOG, r.Method+" "+r.Host+" "+r.RequestURI)
	// for key := range r.Header {
	// 	vals := r.Header[key]
	// 	if len(vals) == 0 {
	// 		logging.Debug(LOG, key + ":")
	// 	} else {
	// 		for _, val := range vals {
	// 			logging.Debug(LOG, key + ": " + val)
	// 		}
	// 	}
	// }
}
