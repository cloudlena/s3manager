package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/redis.v5"

	"github.com/mastertinner/adapters"
	"github.com/satori/go.uuid"

	"golang.org/x/oauth2"
)

const (
	cookieName      = "sess_cookie"
	tokenExpiration = 4 * 24 * time.Hour
)

// tokenWithScope is an OAuth2 token with its scope.
type tokenWithScope struct {
	Token *oauth2.Token
	Scope string
}

// Handler checks if a request is authenticated through OAuth2.
func Handler(cache *redis.Client, config *oauth2.Config, stateString string, tokenContextKey interface{}, scopeContextKey interface{}) adapters.Adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var sessionCookie *http.Cookie
			cookies := r.Cookies()

			// Get session cookie from cookies
			for _, c := range cookies {
				if strings.EqualFold(c.Name, cookieName) {
					sessionCookie = c
					break
				}
			}
			if sessionCookie == nil {
				url := config.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
				http.Redirect(w, r, url, http.StatusTemporaryRedirect)
				return
			}

			tok, err := tokenFromCache(cache, sessionCookie.Value)
			if err != nil || tok.Token == nil || !tok.Token.Valid() {
				url := config.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
				http.Redirect(w, r, url, http.StatusTemporaryRedirect)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, tokenContextKey, tok.Token)
			ctx = context.WithValue(ctx, scopeContextKey, tok.Scope)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// CallbackHandler creates a token and saves it to the cache.
func CallbackHandler(cache *redis.Client, config *oauth2.Config, stateString string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			code := http.StatusInternalServerError
			http.Error(w, http.StatusText(code), code)
			return
		}

		state := r.FormValue("state")
		if state != stateString {
			url := config.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return
		}

		code := r.FormValue("code")
		t, err := config.Exchange(context.Background(), code)
		if err != nil {
			url := config.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return
		}

		// Add scope to token info
		tok := &tokenWithScope{
			Token: t,
			Scope: t.Extra("scope").(string),
		}

		cookieVal := uuid.NewV4().String()

		// Setup the cookie and set it
		cookieToSend := &http.Cookie{
			Name:     cookieName,
			Value:    cookieVal,
			MaxAge:   0,
			Secure:   false,
			HttpOnly: false,
		}

		http.SetCookie(w, cookieToSend)

		// Serialize token and insert to cache
		srlzdToken, err := json.Marshal(&tok)
		if err != nil {
			log.Println("error marshalling token:", err.Error())
			url := config.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return
		}

		err = cache.Set(cookieVal, srlzdToken, tokenExpiration).Err()
		if err != nil {
			log.Println("error adding token to cache:", err.Error())
			url := config.AuthCodeURL(stateString, oauth2.AccessTypeOnline)
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})
}

// tokenFromCache retrieves a tokenURL from the cache.
func tokenFromCache(cache *redis.Client, cookieID string) (*tokenWithScope, error) {
	serializedToken, err := cache.Get(cookieID).Result()
	if err != nil {
		return nil, errors.New("error finding token in cache")
	}

	var tok *tokenWithScope
	err = json.Unmarshal([]byte(serializedToken), &tok)
	if err != nil || tok == nil {
		return nil, errors.New("error unmarshalling token")
	}

	return tok, nil
}
