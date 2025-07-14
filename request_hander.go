package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func requestHandler(w http.ResponseWriter, r *http.Request) {

	m, h, p := getRequestDetails(r)

	service, ok := inMemRegistry[h]
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(400)
		w.Write([]byte("Service not registered"))
		return
	}

	// authorize
	ok, claims, err := authorizeRequest(r, service)
	if err != nil || !ok {
		w.WriteHeader(401)
		return
	}

	//TODO need to handle pasing claims as headers
	fmt.Println(claims)

	//forward to registry
	serviceRequest, err := http.NewRequest(m, fmt.Sprintf("%v/%v", service.BaseUrl, p), r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	serviceResponse, err := http.DefaultClient.Do(serviceRequest)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	defer serviceResponse.Body.Close()

	//issue token if service requests
	if _, ok = serviceResponse.Header[requireTokenHeader]; ok {
		issueToken(w, serviceResponse, service.Secret)
	}

	//respond to client
	copyHeaders(w, serviceResponse)
	w.WriteHeader(serviceResponse.StatusCode)
	_, err = io.Copy(w, serviceResponse.Body)
	if err != nil {
		fmt.Println(err)
	}
}

// returns method, host and path without leading "/"
func getRequestDetails(r *http.Request) (method string, host string, path string) {

	method = r.Method
	trimmed := strings.TrimPrefix(r.RequestURI, "/")

	parts := strings.SplitN(trimmed, "/", 2)

	host = parts[0]
	if len(parts) == 2 {
		path = parts[1]
	} else {
		path = ""
	}

	return method, host, path
}

func authorizeRequest(r *http.Request, s Service) (bool, jwt.MapClaims, error) {
	if !s.Secured {
		// service not secured
		return true, nil, nil
	}

	path := r.URL.Path

	// check public paths
	for _, pr := range s.PublicRoutes {
		fullRoute := fmt.Sprintf("/%v%v", s.PathAlias, pr)
		fmt.Println(fullRoute)
		if fullRoute == path {
			return true, nil, nil
		}
	}

	authVals, ok := r.Header["Authorization"]
	if !ok {
		return false, nil, nil
	}

	for _, v := range authVals {
		parts := strings.SplitN(v, " ", 2)
		//ensure correct format
		if len(parts) > 2 || strings.ToLower(parts[0]) != "bearer" {
			continue
		}

		return parseJWT(parts[1], s.Secret)
	}

	return false, nil, nil
}

func issueToken(w http.ResponseWriter, r *http.Response, secret string) {
	if tokenHeaders, ok := removeProxyTokenHeaders(r); ok {
		exp := (time.Hour * 2)
		expVals, ok := tokenHeaders[tokenExpHeader]
		if ok {
			parsedExp, err := time.ParseDuration(expVals[0])
			if err == nil {
				exp = parsedExp
			} else {
				//log invalid duration
			}
		}
		//strip non claim headers
		delete(tokenHeaders, requireTokenHeader)
		delete(tokenHeaders, tokenExpHeader)

		//map claims
		claims := map[string]any{}
		for k, v := range tokenHeaders {
			key := strings.TrimPrefix(k, tokenHeaderPrefix+"-")
			claims[key] = v
		}

		token, err := createJWT(claims, exp, secret)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Add("Authorization", fmt.Sprintf("Bearer %v", token))
	}
}

// removes and returns Headers with t he X-Proxy-Token prefix, token
func removeProxyTokenHeaders(r *http.Response) (headers map[string][]string, ok bool) {
	headers = map[string][]string{}

	for k, v := range r.Header {
		if strings.HasPrefix(strings.ToLower(k), strings.ToLower(tokenHeaderPrefix)) {
			headers[k] = v
			delete(r.Header, k)
		}
	}

	return headers, len(headers) > 0
}

func copyHeaders(w http.ResponseWriter, r *http.Response) {
	for k, v := range r.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
}

func createJWT(claims map[string]any, exp time.Duration, secret string) (string, error) {
	clms := jwt.MapClaims{
		"exp": time.Now().Add(exp).UnixMilli(),
	}

	for k, v := range claims {
		clms[k] = v
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, clms)
	return t.SignedString([]byte(secret))

}

func parseJWT(t string, s string) (bool, jwt.MapClaims, error) {
	parsed, err := jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
		return []byte(s), nil
	})
	if err != nil {
		return false, nil, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return false, nil, errors.New("invalid claims")
	}

	return parsed.Valid, claims, nil
}
