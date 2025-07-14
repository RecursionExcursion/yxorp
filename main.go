package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/RecursionExcursion/gloader"
)

var loader = gloader.EnvLoader{}

const tokenHeaderPrefix = "X-Proxy-Token"
const tokenRequiredSuffix = "-Required"
const tokenExpSuffix = "-Exp"

// claims will be is the format of X-Proxy-Token-<key>
const (
	requireTokenHeader = tokenHeaderPrefix + tokenRequiredSuffix
	tokenExpHeader     = tokenHeaderPrefix + tokenExpSuffix
)

// key will be Service PathAlias as they cannot be dupped
var inMemRegistry map[string]Service = map[string]Service{
	"dd-api": {
		Name:      "dd-gpi",
		BaseUrl:   "http://localhost:8080",
		PathAlias: "dd-api",
		Secret:    "marshal",
		LastUsed:  -1,
		Enabled:   true,
		Secured:   true,
		PublicRoutes: []string{
			"/hash",
			// "/",
		},
	},
	"app1": {
		Name:      "App1",
		BaseUrl:   "https://app1/.com",
		PathAlias: "app1",
		Secret:    "pass1",
		LastUsed:  -1,
		Enabled:   true,
	},
}

func main() {

	router := http.NewServeMux()

	router.HandleFunc("/", requestHandler)
	router.HandleFunc("/register", registrationHandler)

	p := loader.MustGet("PORT")

	s := http.Server{
		Addr:    fmt.Sprintf(":%v", p),
		Handler: router,
	}

	fmt.Printf("Server is listening on PORT:%v\n", p)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
