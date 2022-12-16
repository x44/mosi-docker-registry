package server

import (
	"mosi-docker-registry/pkg/repo"
	"net/http"
	"strings"
)

// /v2/cli/...
func handleGetCli(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(w, r) {
		return
	}

	paths := splitPath(r)[2:]

	if len(paths) == 0 {
		sendError(w, 400, "BAD REQUEST", "Missing command.")
		return
	}

	cmd := paths[0]
	paths = paths[1:]

	switch cmd {
	case "ls":
		handleGetCliList(w, paths)
	default:
		sendError(w, 400, "BAD REQUEST", "Unknown command '"+cmd+"'")
	}
}

func handleGetCliList(w http.ResponseWriter, args []string) {
	img := "*"
	tag := "*"
	if len(args) > 0 {
		img, tag = getImageAndTag(args[0])
	}

	json, err := repo.List(img, tag)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	sendJson(w, 200, json)
}

func getImageAndTag(s string) (string, string) {
	if len(s) == 0 || s == ":" {
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
			// s = "img:" or "img", a = [img]
			img, tag = a[0], "*"
		}
	} else {
		img, tag = a[0], a[1]
	}
	if len(img) == 0 {
		img = "*"
	}
	if len(tag) == 0 {
		tag = "*"
	}
	return img, tag
}
