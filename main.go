package main

import (
	"fmt"
	"github.com/rannoch/highloadcup2021/api_client"
	"github.com/rannoch/highloadcup2021/miner"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	_ "net/http/pprof"
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
	//apiClient.Debug = true
	//apiClient.Slowlog = time.Second

	apiClientForLicensor := api_client.NewClient(&fasthttp.Client{}, urlParsed.String())

	showStat := true

	m := miner.NewMiner(
		apiClient,
		apiClientForLicensor,
		5,
		7,
		7,
		30,
		showStat,
	)

	err = m.Start()
	fmt.Println(err)
}
