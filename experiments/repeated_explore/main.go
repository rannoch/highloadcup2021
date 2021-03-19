package main

import (
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/model"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	getenv := os.Getenv("ADDRESS")
	if getenv == "" {
		getenv = "localhost"
	}

	urlParsed, err := url.Parse("http://" + getenv + ":8000")
	if err != nil {
		log.Fatal(err)
	}
	urlParsed.Scheme = "http"

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	fastHttpClient := &fasthttp.Client{
		//MaxConnsPerHost: 50,
		//MaxIdemponentCallAttempts: 0,
		//ReadTimeout:     20 * time.Second,
		//WriteTimeout:    20 * time.Second,
	}
	apiClient := api_client.NewClient(fastHttpClient, urlParsed.String())
}

func start(apiClient *api_client.Client) {
	var license model.License

	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {

		}
	}
}
