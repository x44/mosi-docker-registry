package server

import (
	"encoding/base64"
	"fmt"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/logging"
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
	admin               bool
	imagesAllowedToPull []string
	imagesAllowedToPush []string
}

var tokens = map[string]*token{}

func initAuth(w http.ResponseWriter, r *http.Request) bool {
	return checkAuth(w, r, "", false, false, false)
}

func checkAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	return checkAuth(w, r, "", false, false, true)
}

func checkPullAuth(w http.ResponseWriter, r *http.Request, img string) bool {
	return checkAuth(w, r, img, true, false, false)
}

func checkPushAuth(w http.ResponseWriter, r *http.Request, img string) bool {
	return checkAuth(w, r, img, true, true, false)
}

func checkAuth(w http.ResponseWriter, r *http.Request, img string, allowAnonymous, wantPush, wantAdmin bool) bool {
	if checkRequestAuth(r, img, allowAnonymous, wantPush, wantAdmin) {
		return true
	}

	setDefaultHeader(w)

	tokenUrl := config.ServerUrl(r) + config.ServerTokenPath()
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s", service="%s"`, tokenUrl, tokenUrl))
	if allowAnonymous {
		w.Header().Add("WWW-Authenticate", fmt.Sprintf(`BASIC realm="%s"`, "Mosi Docker Registry"))
	}

	sendError(w, 401, "UNAUTHORIZED", "access to the requested resource is not authorized")
	return false
}

func checkRequestAuth(r *http.Request, img string, allowAnonymous, wantPush, wantAdmin bool) bool {

	if checkTokenAuth(r, img, wantPush, wantAdmin) {
		return true
	}

	if checkBasicAuth(r, img, allowAnonymous, wantPush, wantAdmin) {
		return true
	}

	return false
}

func checkTokenAuth(r *http.Request, img string, wantPush, wantAdmin bool) bool {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	tokenStr := auth[7:]
	if token, ok := tokens[tokenStr]; ok {
		now := time.Now().UnixMilli()
		if tokenMaxAge >= 0 && now-token.time > tokenMaxAge {
			delete(tokens, tokenStr)
			return false
		}
		token.time = now
		return checkTokenAccessRights(token, img, wantPush, wantAdmin)
	}
	return false
}

func checkBasicAuth(r *http.Request, img string, allowAnonymous, wantPush, wantAdmin bool) bool {
	_, token := createTokenFromBasicAuth(r, allowAnonymous, false)
	if token == nil {
		return false
	}
	return checkTokenAccessRights(token, img, wantPush, wantAdmin)
}

func checkTokenAccessRights(token *token, img string, wantPush, wantAdmin bool) bool {
	if wantAdmin {
		return token.admin
	}
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

func createAndStoreTokenFromBasicAuth(r *http.Request) string {
	tokenStr, _ := createTokenFromBasicAuth(r, true, true)
	return tokenStr
}

func createTokenFromBasicAuth(r *http.Request, allowAnonymous, store bool) (string, *token) {

	auth := r.Header.Get("Authorization")
	usr, pwd := getUsrAndPwd(auth)

	query := r.URL.Query()

	admin := config.HasAdminAccessRights(usr, pwd)

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

	if !admin && imagesAllowedToPull == nil && imagesAllowedToPush == nil {
		return "", nil
	}

	tokenStr := "DockerToken." + uuid.New().String()
	token := token{
		time:                time.Now().UnixMilli(),
		admin:               admin,
		imagesAllowedToPull: imagesAllowedToPull,
		imagesAllowedToPush: imagesAllowedToPush,
	}

	if store {
		cleanupTokens()
		tokens[tokenStr] = &token
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
	for tokenStr, token := range tokens {
		if now-token.time > tokenMaxAge {
			delete(tokens, tokenStr)
		}
	}
}
