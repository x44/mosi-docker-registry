package server

import (
	"mosi-docker-registry/pkg/json"
	"net/http"
)

func setDefaultHeader(w http.ResponseWriter) {
	w.Header().Set("Server", "Mosi Docker Repository/0.1")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox allow-forms allow-modals allow-popups allow-presentation allow-scripts allow-top-navigation")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Docker-Distribution-Api-Tag", "registry/2.0")
}

func sendError(w http.ResponseWriter, status int, code, msg string) {
	errors := json.NewJsonArray(1)
	errors.Set(0, createError(code, msg))
	rsp := json.NewJsonObject()
	rsp.Put("errors", errors)
	sendJson(w, status, rsp)
}

func createError(code, msg string) *json.JsonObject {
	err := json.NewJsonObject()
	err.Put("code", code)
	err.Put("message", msg)
	err.Put("detail", nil)
	return err
}

func sendJson(w http.ResponseWriter, status int, rsp *json.JsonObject) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	rsp.EncodeWriter(w)
}
