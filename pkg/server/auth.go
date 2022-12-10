package server

import (
	"encoding/base64"
	"mosi-docker-repo/pkg/config"
	"mosi-docker-repo/pkg/logging"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	tokenMaxAge = 3600000
)

type token struct {
	time                int64
	imagesAllowedToPull []string
	imagesAllowedToPush []string
}

var _tokens = map[string]token{}

func CheckAuth(r *http.Request, img string, allowAnonymous, wantPush bool) bool {

	if checkTokenAuth(r, img, wantPush) {
		return true
	}

	if checkBasicAuth(r, img, allowAnonymous, wantPush) {
		return true
	}

	return false
}

func checkTokenAuth(r *http.Request, img string, wantPush bool) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	tokenStr := auth[7:]
	if token, ok := _tokens[tokenStr]; ok {
		now := time.Now().UnixMilli()
		if now-token.time > tokenMaxAge {
			delete(_tokens, tokenStr)
			return false
		}
		return checkTokenAcessRights(&token, img, wantPush)
	}
	return false
}

func checkBasicAuth(r *http.Request, img string, allowAnonymous, wantPush bool) bool {
	_, token := createTokenFromBasicAuth(r, allowAnonymous, false)
	if token == nil {
		return false
	}
	return checkTokenAcessRights(token, img, wantPush)
}

func checkTokenAcessRights(token *token, img string, wantPush bool) bool {
	if wantPush {
		return isImageAccessAllowed(token.imagesAllowedToPush, img)
	} else {
		return isImageAccessAllowed(token.imagesAllowedToPull, img)
	}
}

func isImageAccessAllowed(allowedImages []string, wantedImage string) bool {
	for _, img := range allowedImages {
		if img == "*" || img == wantedImage {
			return true
		}
	}
	return false
}

func CreateAndStoreTokenFromBasicAuth(r *http.Request) string {
	tokenStr, _ := createTokenFromBasicAuth(r, true, true)
	return tokenStr
}

func createTokenFromBasicAuth(r *http.Request, allowAnonymous, store bool) (string, *token) {

	auth := r.Header.Get("Authorization")
	usr, pwd := getUsrAndPwd(auth)

	query := r.URL.Query()

	var imagesAllowedToPull []string = nil
	var imagesAllowedToPush []string = nil

	if scope := query.Get("scope"); scope != "" {
		// docker push/pull
		// scope: "repository:imagename:pull,push"

		a := strings.Split(scope, ":")
		image := a[1]
		imagesAllowedToPull, imagesAllowedToPush = config.GetScopeImageAccessRights(image, usr, pwd, allowAnonymous)
	} else {
		// docker login
		// query.Get("account")
		// query.Get("client_id")
		// query.Get("offline_token")
		// query.Get("service")

		imagesAllowedToPull, imagesAllowedToPush = config.GetAccountImageAccessRights(usr, pwd, allowAnonymous)
	}

	if imagesAllowedToPull == nil && imagesAllowedToPush == nil {
		return "", nil
	}

	tokenStr := "DockerToken." + uuid.New().String()
	token := token{
		time:                time.Now().UnixMilli(),
		imagesAllowedToPull: imagesAllowedToPull,
		imagesAllowedToPush: imagesAllowedToPush,
	}

	if store {
		cleanupTokens()
		_tokens[tokenStr] = token
	}

	return tokenStr, &token
}

func getUsrAndPwd(auth string) (string, string) {
	if !strings.HasPrefix(auth, "Basic ") {
		return "", ""
	}
	dec, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		logging.Error(LOG, "base64 decoding error: %s", err.Error())
		return "", ""
	}
	usrpwd := string(dec)
	sep := strings.Index(usrpwd, ":")
	if sep == -1 {
		return "", ""
	}
	return usrpwd[:sep], usrpwd[sep+1:]
}

func cleanupTokens() {
	now := time.Now().UnixMilli()
	for tokenStr, token := range _tokens {
		if now-token.time > tokenMaxAge {
			delete(_tokens, tokenStr)
		}
	}
}
