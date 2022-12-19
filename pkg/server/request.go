package server

import (
	"mosi-docker-registry/pkg/logging"
	"net/http"
	"strings"
)

func printRequest(r *http.Request) {
	logging.Debug(LOG, "%s %s %s", r.Method, r.Host, r.RequestURI)
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

func splitPath(r *http.Request) []string {
	paths := strings.Split(r.URL.Path, "/")
	var ret []string
	for _, path := range paths {
		if path != "" {
			ret = append(ret, path)
		}
	}
	return ret
}
