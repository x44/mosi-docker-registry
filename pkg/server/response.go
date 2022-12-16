package server

import (
	"encoding/json"
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
	var errors []any
	errors = append(errors, createError(code, msg))

	rsp := map[string]any{
		"errors": errors,
	}
	sendJson(w, status, &rsp)
}

func createError(code, msg string) any {
	err := map[string]any{
		"code":    code,
		"message": msg,
		"detail":  nil,
	}
	return err
}

func sendJson(w http.ResponseWriter, status int, rsp *map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(rsp)
}
