package main

import (
	"fmt"
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

	client := &fasthttp.Client{
		//MaxConnsPerHost: 50,
		//MaxIdemponentCallAttempts: 0,
		//ReadTimeout:     20 * time.Second,
		//WriteTimeout:    20 * time.Second,
	}
	apiClient := NewClient(client, urlParsed.String())
	//apiClient.Debug = true
	//apiClient.Slowlog = time.Second

	showStat := true

	miner := NewMiner(apiClient, 5, 7, 9, 5, showStat)

	err = miner.Start()
	fmt.Println(err)
}
