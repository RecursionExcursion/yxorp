package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/RecursionExcursion/gloader"
)

var loader = gloader.EnvLoader{}

var requireTokenHeader = loader.GetOrFallback("Proxy-Token-Required", "X-Proxy-Token-Required")

type Service struct {
	Name           string
	BaseUrl        string
	PathAlias      string
	ServiceToken   string
	Secret         string
	LastUsed       int64
	Enabled        bool
	AllowedMethods []string
}

// key will be Service PathAlias as they cannot be dupped
var inMemRegistry map[string]Service = map[string]Service{
	"dd-api": {
		Name:           "dd-gpi",
		BaseUrl:        "http://localhost:8080",
		PathAlias:      "dd-api",
		Secret:         "marshal",
		LastUsed:       -1,
		Enabled:        true,
		AllowedMethods: []string{},
	},
	"app1": {
		Name:           "App1",
		BaseUrl:        "https://app1/.com",
		PathAlias:      "app1",
		Secret:         "pass1",
		LastUsed:       -1,
		Enabled:        true,
		AllowedMethods: []string{},
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

func requestHandler(w http.ResponseWriter, r *http.Request) {

	m, h, p := getRequestDetails(r)

	s, ok := inMemRegistry[h]
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(400)
		w.Write([]byte("Service not registered"))
		return
	}

	//forward to registry
	serviceRequest, err := http.NewRequest(m, fmt.Sprintf("%v/%v", s.BaseUrl, p), r.Body)
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

	//copy headers
	for k, v := range serviceResponse.Header {
		fmt.Println(k)
		fmt.Println(v)
		for _, vv := range v {
			fmt.Println(vv)
			w.Header().Add(k, vv)
		}
	}

	//response to client

	w.WriteHeader(serviceResponse.StatusCode)
	size, err := io.Copy(w, serviceResponse.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(size)
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {

}

// returns method, host and path without leading "/"
func getRequestDetails(r *http.Request) (method string, host string, path string) {
	fmt.Println(r.RequestURI)
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
