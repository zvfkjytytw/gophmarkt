package gophmarkthttpserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	storage "github.com/zvfkjytytw/gophmarkt/internal/server/storage"
)

const (
	cookieAuthToken     = "AuthToken"
	headerAuthorization = "Authorization"
)

type AuthBody struct {
	Login    string
	Password string
}

// user registration
func (h *HTTPServer) userRegistration(w http.ResponseWriter, r *http.Request) {
	contentType, ok := r.Header["Content-Type"]
	if !ok || contentType[0] != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("wrong Content-Type. Expect application/json"))
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed read body"))
		return
	}

	var registryData AuthBody
	err = json.Unmarshal(body, &registryData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed unmarshal body"))
		return
	}

	status, err := h.storage.AddUser(registryData.Login, registryData.Password)
	if err != nil {
		switch status {
		case storage.UserExist:
			h.logger.Sugar().Errorf("user %s registration failed: %v", registryData.Login, err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf("User %s is already exists", registryData.Login)))
			return
		case storage.UserPasswordWrong:
			h.logger.Sugar().Errorf("user %s registration failed: %v", registryData.Login, err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Password is unsuitable"))
			return
		case storage.UserOperationFailed:
			h.logger.Sugar().Errorf("user %s registration failed: %v", registryData.Login, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("User %s is not registered", registryData.Login)))
			return
		}
	}

	authToken := getAuthToken(registryData.Login)

	h.Lock()
	h.authUsers[authToken] = &authUser{
		ttl:   authTTL,
		login: registryData.Login,
	}
	h.Unlock()

	cookie := http.Cookie{Name: cookieAuthToken, Value: authToken, Expires: time.Now().Add(time.Hour)}
	http.SetCookie(w, &cookie)

	w.Header().Set(headerAuthorization, authToken)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("User %s is registered", registryData.Login)))
}

// user authorization
func (h *HTTPServer) userLogin(w http.ResponseWriter, r *http.Request) {
	contentType, ok := r.Header["Content-Type"]
	if !ok || contentType[0] != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("wrong Content-Type. Expect application/json"))
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed read body"))
		return
	}

	var authenticationData AuthBody
	err = json.Unmarshal(body, &authenticationData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed unmarshal body"))
		return
	}

	status, err := h.storage.CheckUser(authenticationData.Login, authenticationData.Password)
	if err != nil {
		switch status {
		case storage.UserNotFound:
			h.logger.Sugar().Errorf("user %s authentication failed: %v", authenticationData.Login, err)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(fmt.Sprintf("User %s is not found", authenticationData.Login)))
			return
		case storage.UserPasswordWrong:
			h.logger.Sugar().Errorf("user %s authentication failed: %v", authenticationData.Login, err)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Incorrect password"))
			return
		case storage.UserOperationFailed:
			h.logger.Sugar().Errorf("user %s authentication failed: %v", authenticationData.Login, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Authentication error"))
			return
		}
	}

	authToken := getAuthToken(authenticationData.Login)

	h.Lock()
	h.authUsers[authToken] = &authUser{
		ttl:   authTTL,
		login: authenticationData.Login,
	}
	h.Unlock()

	cookie := http.Cookie{Name: cookieAuthToken, Value: authToken, Expires: time.Now().Add(time.Hour)}
	http.SetCookie(w, &cookie)

	w.Header().Set(headerAuthorization, authToken)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("User %s is authenticated", authenticationData.Login)))
}

// generate authentication token
func getAuthToken(login string) string {
	buf := []byte(login)
	salt, err := time.Now().MarshalBinary()
	if err == nil {
		buf = append(buf, salt...)
	}

	return fmt.Sprintf("%x", sha256.Sum256(buf))
}

// checking authentication token
func (h *HTTPServer) authUserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		authorization, ok := r.Header[headerAuthorization]
		if ok {
			token = authorization[0]
		} else {
			tokenCookie, err := r.Cookie(cookieAuthToken)
			if err == nil {
				token = tokenCookie.Value
			}
		}
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("absent auth token"))
			return
		}

		h.RLock()
		defer h.RUnlock()
		user, ok := h.authUsers[token]
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid token"))
			return
		}
		user.ttl = authTTL

		ctx := context.WithValue(r.Context(), contextAuthUser, user.login)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
