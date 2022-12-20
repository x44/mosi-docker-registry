package server

import (
	"mosi-docker-registry/pkg/json"
	"mosi-docker-registry/pkg/logging"
	"mosi-docker-registry/pkg/repo"
	"net/http"
	"strings"
)

func parseRequest(w http.ResponseWriter, r *http.Request) (ok bool, cmd string, paths []string, args *json.JsonObject) {
	ok = false
	cmd = ""
	paths = nil
	args = nil

	if !checkAdminAuth(w, r) {
		return
	}

	paths = splitPath(r)[2:]

	if len(paths) == 0 {
		sendError(w, 400, "BAD REQUEST", "Missing command.")
		return
	}

	cmd = paths[0]
	paths = paths[1:]

	argsStr := r.Header.Get("args")
	if len(argsStr) > 0 {
		var err error
		args, err = json.DecodeString(argsStr)
		if err != nil {
			sendError(w, 400, "BAD REQUEST", "Invalid Json args in header")
			return
		}
	}
	ok = true
	return
}

// /v2/cli/...
func cliHandleGet(w http.ResponseWriter, r *http.Request) {
	ok, cmd, paths, args := parseRequest(w, r)
	if !ok {
		return
	}

	switch cmd {
	case "ls":
		cliHandleGetListImages(w, paths, args)
	default:
		sendError(w, 400, "BAD REQUEST", "Unknown command '"+cmd+"'")
	}
}

func cliHandleGetListImages(w http.ResponseWriter, paths []string, args *json.JsonObject) {
	img, tag := getImageAndTag(paths)

	json, err := repo.List(img, tag)

	if err != nil {
		logging.Error(LOG, err)
		w.WriteHeader(500)
		return
	}

	sendJson(w, 200, json)
}

// /v2/cli/...
func cliHandleDelete(w http.ResponseWriter, r *http.Request) {
	ok, cmd, paths, args := parseRequest(w, r)
	if !ok {
		return
	}

	switch cmd {
	case "rm":
		cliHandleDeleteImages(w, paths, args)
	default:
		sendError(w, 400, "BAD REQUEST", "Unknown command '"+cmd+"'")
	}
}

func cliHandleDeleteImages(w http.ResponseWriter, paths []string, args *json.JsonObject) {
	img, tag := getImageAndTag(paths)
	dry := args.GetBool("dry", false)

	json, err := repo.Delete(img, tag, dry)

	if err != nil {
		logging.Error(LOG, err)
		w.WriteHeader(500)
		return
	}

	sendJson(w, 200, json)
}

func getImageAndTag(paths []string) (string, string) {
	s := ""
	if len(paths) > 0 {
		s = paths[0]
	}

	if len(s) == 0 {
		return "*", ""
	}
	if s == ":" {
		return "*", "*"
	}

	var img string
	var tag string
	a := strings.Split(s, ":")
	if len(a) == 1 {
		if strings.HasPrefix(s, ":") {
			// s = ":tag", a = [tag]
			img, tag = "*", a[0]
		} else {
			// s = "img", a = [img]
			img, tag = a[0], ""
		}
	} else {
		img, tag = a[0], a[1]

		if len(a[1]) == 0 {
			// s = "img:", a = [img,""]
			tag = "*"
		}
	}
	if tag != "" && img == "" {
		img = "*"
	}
	return img, tag
}
