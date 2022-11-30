package server

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	tokenMaxAge = 3600000
)

var _tokens = map[string]int64{}

func CreateToken() string {
	cleanup()
	token := "DockerToken." + uuid.New().String()
	_tokens[token] = time.Now().UnixMilli()
	return token
}

func ValidateToken(token string) bool {
	if !strings.HasPrefix(token, "Bearer ") {
		return false
	}
	token = token[7:]
	if t, ok := _tokens[token]; ok {
		now := time.Now().UnixMilli()
		return (now-t <= tokenMaxAge)
	}
	return false
}

func cleanup() {
	now := time.Now().UnixMilli()
	for token, t := range _tokens {
		if now-t > tokenMaxAge {
			delete(_tokens, token)
		}
	}
}
